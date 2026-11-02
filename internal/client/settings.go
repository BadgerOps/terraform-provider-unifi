package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SiteSetting is one document of the legacy `setting` collection, for example
// `rsyslogd` or `ntp`, as the controller stores it: every field, including the
// ones a resource doesn't model.
type SiteSetting map[string]json.RawMessage

// Decode unmarshals one field into target and reports whether it was present.
// A JSON null counts as absent.
func (s SiteSetting) Decode(field string, target any) (bool, error) {
	raw, ok := s[field]
	if !ok || string(raw) == "null" {
		return false, nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return true, fmt.Errorf("decode setting field %s: %w", field, err)
	}
	return true, nil
}

// GetSiteSetting reads the site's legacy setting document with the given key.
func (c *Client) GetSiteSetting(ctx context.Context, siteID, key string) (SiteSetting, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("read setting %s site: %w", key, err)
	}
	return c.getSiteSetting(ctx, site, key)
}

func (c *Client) getSiteSetting(ctx context.Context, site, key string) (SiteSetting, error) {
	var response legacyResponse[[]SiteSetting]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "get", "setting", key}, nil, &response); err != nil {
		return nil, fmt.Errorf("read setting %s: %w", key, err)
	}
	for _, setting := range response.Data {
		var settingKey string
		if found, err := setting.Decode("key", &settingKey); err == nil && found && settingKey == key {
			return setting, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("setting %s not found in site %s", key, site)}
}

// UpdateSiteSetting changes only the given fields of a legacy setting. The
// current document is read first and sent back whole with the fields overlaid,
// to preserve unmodelled fields, credentials among them, exactly as the
// controller returned them. A site that has never saved the setting has no
// document: set/setting creates it from the given fields. The stored document
// is read back, because some controllers acknowledge without returning it.
func (c *Client) UpdateSiteSetting(ctx context.Context, siteID, key string, fields map[string]any) (SiteSetting, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update setting %s site: %w", key, err)
	}
	existing, err := c.getSiteSetting(ctx, site, key)
	creating := false
	switch {
	case IsNotFound(err):
		creating = true
	case err != nil:
		return nil, err
	}
	payload := make(map[string]json.RawMessage, len(existing)+len(fields))
	for field, value := range existing {
		payload[field] = value
	}
	for field, value := range fields {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode setting field %s: %w", field, err)
		}
		payload[field] = encoded
	}
	payload["key"] = json.RawMessage(`"` + key + `"`)
	var response legacyResponse[[]SiteSetting]
	if err := c.doLegacyRequest(ctx, http.MethodPut, []string{"s", site, "set", "setting", key}, payload, &response); err != nil {
		if creating {
			return nil, fmt.Errorf("site %s has no %s setting and the controller refused to create it: %w", site, key, err)
		}
		return nil, fmt.Errorf("update setting %s: %w", key, err)
	}
	return c.getSiteSetting(ctx, site, key)
}
