package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PortForward uses the legacy Network API field names. Ports deliberately remain
// strings: the controller accepts individual ports, ranges, and comma lists.
type PortForward struct {
	ID                    string `json:"_id,omitempty"`
	Name                  string `json:"name"`
	Enabled               bool   `json:"enabled"`
	WANInterface          string `json:"pfwd_interface"`
	Protocol              string `json:"proto"`
	Source                string `json:"src"`
	SourceLimitingEnabled bool   `json:"src_limiting_enabled"`
	SourceFirewallGroupID string `json:"src_firewall_group_id,omitempty"`
	DestinationPort       string `json:"dst_port"`
	ForwardIP             string `json:"fwd"`
	ForwardPort           string `json:"fwd_port"`
	LoggingEnabled        bool   `json:"log"`
	rawFields             map[string]json.RawMessage
}

// Retain every controller field for read-modify-write, including fields this
// provider does not expose. RawMessage also preserves numbers without float conversion.
func (p *PortForward) UnmarshalJSON(data []byte) error {
	type wirePortForward PortForward
	var decoded wirePortForward
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &decoded.rawFields); err != nil {
		return err
	}
	*p = PortForward(decoded)
	return nil
}

func (c *Client) ListPortForwards(ctx context.Context, siteID string) ([]PortForward, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list port forwards site: %w", err)
	}
	return c.listPortForwards(ctx, site)
}

func (c *Client) listPortForwards(ctx context.Context, site string) ([]PortForward, error) {
	var response legacyResponse[[]PortForward]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "rest", "portforward"}, nil, &response); err != nil {
		return nil, fmt.Errorf("list port forwards: %w", err)
	}
	// A port-forward refresh must not forget state on an incomplete list. Keep
	// this endpoint-specific so DHCP retains its existing empty-table behavior.
	if response.Data == nil {
		return nil, fmt.Errorf("port forward list has missing or null data")
	}
	return response.Data, nil
}

func (c *Client) GetPortForward(ctx context.Context, siteID, id string) (*PortForward, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, err
	}
	return c.getPortForward(ctx, site, id)
}

func (c *Client) getPortForward(ctx context.Context, site, id string) (*PortForward, error) {
	rules, err := c.listPortForwards(ctx, site)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.ID == id {
			return &rule, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("port forward %s not found in site %s", id, site)}
}

func (c *Client) CreatePortForward(ctx context.Context, siteID string, rule PortForward) (*PortForward, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("create port forward site: %w", err)
	}
	rule.ID = ""
	rule.SourceLimitingEnabled = rule.Source != "any"
	// Initialize unmodelled fields on create; updates preserve controller values.
	payload := struct {
		PortForward
		DestinationIP         string   `json:"destination_ip"`
		DestinationIPs        []string `json:"destination_ips"`
		SourceFirewallGroupID string   `json:"src_firewall_group_id"`
	}{PortForward: rule, DestinationIP: "any", DestinationIPs: []string{}}
	var response legacyResponse[[]PortForward]
	if err := c.doLegacyRequestWithExpectedStatus(ctx, http.MethodPost, []string{"s", site, "rest", "portforward"}, payload, &response, http.StatusOK, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("create port forward: %w", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID == "" {
		return nil, fmt.Errorf("create port forward: expected one rule with an _id in response")
	}
	return &response.Data[0], nil
}

func (c *Client) UpdatePortForward(ctx context.Context, siteID, id string, rule PortForward) (*PortForward, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update port forward site: %w", err)
	}
	existing, err := c.getPortForward(ctx, site, id)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	rule.SourceFirewallGroupID = existing.SourceFirewallGroupID
	rule.SourceLimitingEnabled = existing.SourceLimitingEnabled
	if rule.Source != existing.Source {
		rule.SourceLimitingEnabled = rule.Source != "any" || (existing.SourceFirewallGroupID != "" && existing.SourceLimitingEnabled)
	}
	modelled, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("encode port forward: %w", err)
	}
	// Overlay only modelled fields onto the complete fetched object. Omitted
	// destination filters, source groups, and future fields survive full replacement.
	payload := existing.rawFields
	if err := json.Unmarshal(modelled, &payload); err != nil {
		return nil, fmt.Errorf("merge port forward: %w", err)
	}
	var response legacyResponse[[]PortForward]
	if err := c.doLegacyRequest(ctx, http.MethodPut, []string{"s", site, "rest", "portforward", id}, payload, &response); err != nil {
		return nil, fmt.Errorf("update port forward: %w", err)
	}
	if len(response.Data) == 1 && response.Data[0].ID == id && completePortForwardState(response.Data[0]) {
		return &response.Data[0], nil
	}
	// Some controllers return only an acknowledgement rather than the rule.
	return c.getPortForward(ctx, site, id)
}

func completePortForwardState(rule PortForward) bool {
	for _, key := range []string{"name", "enabled", "pfwd_interface", "proto", "src", "dst_port", "fwd", "fwd_port", "log"} {
		if _, ok := rule.rawFields[key]; !ok {
			return false
		}
	}
	return true
}

func (c *Client) DeletePortForward(ctx context.Context, siteID, id string) error {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return fmt.Errorf("delete port forward site: %w", err)
	}
	err = c.doLegacyRequestWithExpectedStatus(ctx, http.MethodDelete, []string{"s", site, "rest", "portforward", id}, nil, nil, http.StatusOK, http.StatusNoContent)
	if IsNotFound(err) {
		return nil
	}
	return err
}
