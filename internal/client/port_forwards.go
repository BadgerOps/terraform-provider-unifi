package client

import (
	"context"
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
}

func (c *Client) ListPortForwards(ctx context.Context, siteID string) ([]PortForward, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list port forwards site: %w", err)
	}
	var response legacyResponse[[]PortForward]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "rest", "portforward"}, nil, &response); err != nil {
		return nil, fmt.Errorf("list port forwards: %w", err)
	}
	return response.Data, nil
}

func (c *Client) GetPortForward(ctx context.Context, siteID, id string) (*PortForward, error) {
	rules, err := c.ListPortForwards(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.ID == id {
			return &rule, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("port forward %s not found in site %s", id, siteID)}
}

func (c *Client) CreatePortForward(ctx context.Context, siteID string, rule PortForward) (*PortForward, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("create port forward site: %w", err)
	}
	rule.ID = ""
	rule.SourceLimitingEnabled = rule.Source != "any"
	// These unmodelled fields are initialized only on create. Sending them on
	// update would reset restrictions configured outside this resource.
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
	existing, err := c.GetPortForward(ctx, siteID, id)
	if err != nil {
		return nil, err
	}
	if existing.SourceFirewallGroupID != "" {
		return nil, fmt.Errorf("port forward %s uses a source firewall group, which this resource does not manage; remove the group restriction on the controller before updating this rule", id)
	}
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update port forward site: %w", err)
	}
	rule.ID = id
	rule.SourceFirewallGroupID = ""
	rule.SourceLimitingEnabled = rule.Source != "any"
	if err := c.doLegacyRequest(ctx, http.MethodPut, []string{"s", site, "rest", "portforward", id}, rule, nil); err != nil {
		return nil, fmt.Errorf("update port forward: %w", err)
	}
	return c.GetPortForward(ctx, siteID, id)
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
