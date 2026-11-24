package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// WANSLA is one WAN SLA of the Network application's v2 API
// (`/v2/api/site/<site>/wan-slas`, Settings > Internet > WAN SLA): the
// monitors that decide whether a WAN counts as up for failover and load
// balancing. A WAN refers to it by `_id` (legacy networkconf `wan_sla`).
type WANSLA struct {
	ID                string          `json:"_id,omitempty"`
	Name              string          `json:"name"`
	MonitorHealthType string          `json:"monitor_health_type,omitempty"`
	Monitors          []WANSLAMonitor `json:"monitors"`
}

// WANSLAMonitor is one probe of a WAN SLA.
type WANSLAMonitor struct {
	MonitorID  int64       `json:"monitor_id"`
	Type       string      `json:"type"`
	Target     string      `json:"target,omitempty"`
	Domain     string      `json:"domain,omitempty"`
	Interval   int64       `json:"interval"`
	TimePeriod int64       `json:"time_period"`
	Alert      WANSLAAlert `json:"alert"`
}

// WANSLAAlert holds a monitor's thresholds. A threshold the controller
// leaves out is not evaluated (the UI's "off").
type WANSLAAlert struct {
	LossThreshold    *int64 `json:"loss_threshold,omitempty"`
	LatencyThreshold *int64 `json:"latency_threshold,omitempty"`
	JitterThreshold  *int64 `json:"jitter_threshold,omitempty"`
	ThresholdPolicy  string `json:"threshold_policy,omitempty"`
}

// ListWANSLAs returns the site's WAN SLAs.
func (c *Client) ListWANSLAs(ctx context.Context, siteID string) ([]WANSLA, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list WAN SLAs site: %w", err)
	}
	slas, _, err := c.listWANSLAs(ctx, site)
	return slas, err
}

// listWANSLAs also returns every SLA as raw fields by ID, for a
// read-modify-write that keeps the fields this client doesn't model.
func (c *Client) listWANSLAs(ctx context.Context, site string) ([]WANSLA, map[string]map[string]json.RawMessage, error) {
	var body json.RawMessage
	if err := c.doV2Request(ctx, http.MethodGet, []string{"site", site, "wan-slas"}, nil, &body, http.StatusOK); err != nil {
		return nil, nil, fmt.Errorf("list WAN SLAs: %w", err)
	}
	var slas []WANSLA
	if err := json.Unmarshal(body, &slas); err != nil {
		return nil, nil, fmt.Errorf("decode WAN SLAs: %w", err)
	}
	var raws []map[string]json.RawMessage
	if err := json.Unmarshal(body, &raws); err != nil {
		return nil, nil, fmt.Errorf("decode WAN SLAs: %w", err)
	}
	byID := make(map[string]map[string]json.RawMessage, len(raws))
	for i, raw := range raws {
		byID[slas[i].ID] = raw
	}
	return slas, byID, nil
}

// GetWANSLA returns one WAN SLA by ID. The API has no documented single-item
// read, so it reads the list; a missing ID is a not-found error.
func (c *Client) GetWANSLA(ctx context.Context, siteID, id string) (*WANSLA, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("read WAN SLA site: %w", err)
	}
	return c.getWANSLA(ctx, site, id)
}

func (c *Client) getWANSLA(ctx context.Context, site, id string) (*WANSLA, error) {
	slas, _, err := c.listWANSLAs(ctx, site)
	if err != nil {
		return nil, err
	}
	for i := range slas {
		if slas[i].ID == id {
			return &slas[i], nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("WAN SLA %s not found in site %s", id, site)}
}

// CreateWANSLA creates a WAN SLA and returns it as the controller stored it.
// The new ID is taken from the answer, or else found by name in the list.
func (c *Client) CreateWANSLA(ctx context.Context, siteID string, sla WANSLA) (*WANSLA, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("create WAN SLA site: %w", err)
	}
	sla.ID = ""
	var created WANSLA
	if err := c.doV2Request(ctx, http.MethodPost, []string{"site", site, "wan-slas"}, sla, &created, http.StatusOK, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("create WAN SLA %s: %w", sla.Name, err)
	}
	if created.ID != "" {
		return c.getWANSLA(ctx, site, created.ID)
	}
	slas, _, err := c.listWANSLAs(ctx, site)
	if err != nil {
		return nil, err
	}
	for i := range slas {
		if slas[i].Name == sla.Name {
			return &slas[i], nil
		}
	}
	return nil, fmt.Errorf("created WAN SLA %s but can't find it in the list", sla.Name)
}

// UpdateWANSLA replaces the SLA's name, health type and monitors. The current
// document is sent back with those fields overlaid, so top-level fields this
// client doesn't model keep the controller's values.
func (c *Client) UpdateWANSLA(ctx context.Context, siteID string, sla WANSLA) (*WANSLA, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update WAN SLA site: %w", err)
	}
	_, raws, err := c.listWANSLAs(ctx, site)
	if err != nil {
		return nil, err
	}
	existing, ok := raws[sla.ID]
	if !ok {
		return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("WAN SLA %s not found in site %s", sla.ID, site)}
	}
	encoded, err := json.Marshal(sla)
	if err != nil {
		return nil, fmt.Errorf("encode WAN SLA: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, fmt.Errorf("encode WAN SLA: %w", err)
	}
	payload := make(map[string]json.RawMessage, len(existing)+len(fields))
	for k, v := range existing {
		payload[k] = v
	}
	for k, v := range fields {
		payload[k] = v
	}
	if err := c.doV2Request(ctx, http.MethodPut, []string{"site", site, "wan-slas", sla.ID}, payload, nil, http.StatusOK, http.StatusNoContent); err != nil {
		return nil, fmt.Errorf("update WAN SLA %s: %w", sla.ID, err)
	}
	return c.getWANSLA(ctx, site, sla.ID)
}

// DeleteWANSLA deletes a WAN SLA. A WAN that still uses it keeps the
// reference until it is changed; the controller decides whether to refuse.
func (c *Client) DeleteWANSLA(ctx context.Context, siteID, id string) error {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return fmt.Errorf("delete WAN SLA site: %w", err)
	}
	if err := c.doV2Request(ctx, http.MethodDelete, []string{"site", site, "wan-slas", id}, nil, nil, http.StatusOK, http.StatusNoContent); err != nil {
		return fmt.Errorf("delete WAN SLA %s: %w", id, err)
	}
	return nil
}

// v2BaseURL is the Network application's v2 API, a sibling of the legacy API:
// <...>/proxy/network/api becomes <...>/proxy/network/v2/api.
func (c *Client) v2BaseURL() *url.URL {
	base := *c.legacyBaseURL
	base.Path = joinURLPath(strings.TrimSuffix(strings.TrimRight(base.Path, "/"), "/api"), "v2", "api")
	return &base
}

// doV2Request sends a request to the v2 API. Unlike the legacy API it has no
// `meta` envelope: the status code says whether it worked, and an error body
// carries `code` and `message`.
func (c *Client) doV2Request(ctx context.Context, method string, pathElements []string, payload any, target any, expectedStatusCodes ...int) error {
	var bodyReader io.Reader
	if payload != nil {
		var err error
		bodyReader, err = jsonBodyReader(payload)
		if err != nil {
			return err
		}
	}
	base := c.v2BaseURL()
	endpoint := *base
	endpoint.Path = joinURLPath(base.Path, pathElements...)

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("build v2 request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-API-KEY", c.apiKey)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read v2 response body: %w", err)
	}
	if err := requireStatus(response.StatusCode, body, expectedStatusCodes...); err != nil {
		return err
	}
	if target == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode v2 response body: %w", err)
	}
	return nil
}
