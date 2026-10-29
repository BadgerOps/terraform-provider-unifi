package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// IPSSuppressionTracking limits a suppressed alert to an address: mode `ip`
// (one IPv4 address) or `subnet` (a CIDR prefix), matched on the source, the
// destination, or both (`src`, `dest`, `both`).
type IPSSuppressionTracking struct {
	Mode      string `json:"mode"`
	Value     string `json:"value"`
	Direction string `json:"direction"`
}

// IPSSuppressionAlert is one suppressed IPS signature, in the legacy
// `ips_suppression` setting's field names. `type` is `all` (suppress the
// signature everywhere) or `track` (only for the tracking entries).
type IPSSuppressionAlert struct {
	GID       int64                    `json:"gid"`
	ID        int64                    `json:"id"`
	Category  string                   `json:"category"`
	Signature string                   `json:"signature"`
	Type      string                   `json:"type"`
	Tracking  []IPSSuppressionTracking `json:"tracking,omitempty"` // absent on type "all", as the controller stores it
}

// IPSSuppression is the site's legacy `ips_suppression` setting. Only the
// alert list is modelled; the whitelist and every other controller field are
// kept from the last read and written back unchanged.
type IPSSuppression struct {
	Alerts    []IPSSuppressionAlert `json:"alerts"`
	rawFields map[string]json.RawMessage
}

func (s *IPSSuppression) UnmarshalJSON(data []byte) error {
	type wireIPSSuppression IPSSuppression
	var decoded wireIPSSuppression
	raw, err := unmarshalWithRawFields(data, &decoded)
	if err != nil {
		return err
	}
	decoded.rawFields = raw
	*s = IPSSuppression(decoded)
	return nil
}

const ipsSuppressionKey = "ips_suppression"

func (c *Client) GetIPSSuppression(ctx context.Context, siteID string) (*IPSSuppression, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("read IPS suppression site: %w", err)
	}
	return c.getIPSSuppression(ctx, site)
}

func (c *Client) getIPSSuppression(ctx context.Context, site string) (*IPSSuppression, error) {
	var response legacyResponse[[]IPSSuppression]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "get", "setting", ipsSuppressionKey}, nil, &response); err != nil {
		return nil, fmt.Errorf("read IPS suppression: %w", err)
	}
	for _, setting := range response.Data {
		if key, ok := setting.rawFields["key"]; ok && string(key) == `"`+ipsSuppressionKey+`"` {
			if setting.Alerts == nil {
				setting.Alerts = []IPSSuppressionAlert{}
			}
			return &setting, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("setting %s not found in site %s", ipsSuppressionKey, site)}
}

// SetIPSSuppression replaces the site's suppressed alerts. The legacy setting
// endpoint replaces the whole document, so the current one is read first and
// only `alerts` is overlaid: the whitelist and unmodelled fields survive. A
// site that has never had the setting gets the document created. An alert
// with an empty Signature keeps the description the controller already holds
// for the same gid and id.
func (c *Client) SetIPSSuppression(ctx context.Context, siteID string, alerts []IPSSuppressionAlert) (*IPSSuppression, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update IPS suppression site: %w", err)
	}
	existing, err := c.getIPSSuppression(ctx, site)
	switch {
	case IsNotFound(err):
		// set/setting creates a missing document; the controller assigns its
		// identity fields, so send only what we model.
		existing = &IPSSuppression{rawFields: map[string]json.RawMessage{}}
	case err != nil:
		return nil, err
	}
	descriptions := make(map[[2]int64]string, len(existing.Alerts))
	for _, alert := range existing.Alerts {
		descriptions[[2]int64{alert.GID, alert.ID}] = alert.Signature
	}
	merged := make([]IPSSuppressionAlert, 0, len(alerts))
	for _, alert := range alerts {
		if alert.Signature == "" {
			alert.Signature = descriptions[[2]int64{alert.GID, alert.ID}]
		}
		merged = append(merged, alert)
	}
	encoded, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("encode IPS suppression: %w", err)
	}
	payload := make(map[string]json.RawMessage, len(existing.rawFields)+1)
	for key, value := range existing.rawFields {
		payload[key] = value
	}
	payload["alerts"] = encoded
	payload["key"] = json.RawMessage(`"` + ipsSuppressionKey + `"`)
	var response legacyResponse[[]IPSSuppression]
	if err := c.doLegacyRequest(ctx, http.MethodPut, []string{"s", site, "set", "setting", ipsSuppressionKey}, payload, &response); err != nil {
		return nil, fmt.Errorf("update IPS suppression: %w", err)
	}
	// Some controllers acknowledge without returning the document; read it back
	// so state always reflects what the controller stored.
	return c.getIPSSuppression(ctx, site)
}
