package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type RadiusProfile struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type DeviceTag struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name"`
	DeviceIDs []string       `json:"deviceIds"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type DNSPolicy struct {
	ID               string         `json:"id,omitempty"`
	Type             string         `json:"type"`
	Enabled          bool           `json:"enabled"`
	Domain           *string        `json:"domain,omitempty"`
	IPv4Address      *string        `json:"ipv4Address,omitempty"`
	IPv6Address      *string        `json:"ipv6Address,omitempty"`
	TargetDomain     *string        `json:"targetDomain,omitempty"`
	MailServerDomain *string        `json:"mailServerDomain,omitempty"`
	Priority         *int64         `json:"priority,omitempty"`
	Text             *string        `json:"text,omitempty"`
	ServerDomain     *string        `json:"serverDomain,omitempty"`
	Service          *string        `json:"service,omitempty"`
	Protocol         *string        `json:"protocol,omitempty"`
	Port             *int64         `json:"port,omitempty"`
	Weight           *int64         `json:"weight,omitempty"`
	IPAddress        *string        `json:"ipAddress,omitempty"`
	TTLSeconds       *int64         `json:"ttlSeconds,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type ACLRuleDeviceFilter struct {
	Type      string   `json:"type"`
	DeviceIDs []string `json:"deviceIds,omitempty"`
}

type ACLRuleEndpointFilter struct {
	Type                 string   `json:"type"`
	IPAddressesOrSubnets []string `json:"ipAddressesOrSubnets,omitempty"`
	NetworkIDs           []string `json:"networkIds,omitempty"`
	PortFilter           []int64  `json:"portFilter,omitempty"`
	MacAddresses         []string `json:"macAddresses,omitempty"`
	PrefixLength         *int64   `json:"prefixLength,omitempty"`
}

type ACLRule struct {
	ID                    string                 `json:"id,omitempty"`
	Type                  string                 `json:"type"`
	Enabled               bool                   `json:"enabled"`
	Name                  string                 `json:"name"`
	Description           *string                `json:"description,omitempty"`
	Action                string                 `json:"action"`
	EnforcingDeviceFilter *ACLRuleDeviceFilter   `json:"enforcingDeviceFilter,omitempty"`
	Index                 int64                  `json:"index,omitempty"`
	SourceFilter          *ACLRuleEndpointFilter `json:"sourceFilter,omitempty"`
	DestinationFilter     *ACLRuleEndpointFilter `json:"destinationFilter,omitempty"`
	ProtocolFilter        []string               `json:"protocolFilter,omitempty"`
	NetworkIDFilter       *string                `json:"networkIdFilter,omitempty"`
	Metadata              map[string]any         `json:"metadata,omitempty"`
}

func (c *Client) CreateDNSPolicy(ctx context.Context, siteID string, request DNSPolicy) (*DNSPolicy, error) {
	var response DNSPolicy
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sites/%s/dns/policies", siteID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetDNSPolicy(ctx context.Context, siteID, dnsPolicyID string) (*DNSPolicy, error) {
	var response DNSPolicy
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/dns/policies/%s", siteID, dnsPolicyID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdateDNSPolicy(ctx context.Context, siteID, dnsPolicyID string, request DNSPolicy) (*DNSPolicy, error) {
	var response DNSPolicy
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/sites/%s/dns/policies/%s", siteID, dnsPolicyID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteDNSPolicy(ctx context.Context, siteID, dnsPolicyID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/sites/%s/dns/policies/%s", siteID, dnsPolicyID), nil, nil, nil)
}

func (c *Client) ListDNSPolicies(ctx context.Context, siteID string) ([]DNSPolicy, error) {
	var policies []DNSPolicy
	offset := 0

	for {
		var response page[DNSPolicy]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/dns/policies", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		policies = append(policies, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return policies, nil
}

func (c *Client) CreateACLRule(ctx context.Context, siteID string, request ACLRule) (*ACLRule, error) {
	var response ACLRule
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sites/%s/acl-rules", siteID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetACLRule(ctx context.Context, siteID, aclRuleID string) (*ACLRule, error) {
	var response ACLRule
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/acl-rules/%s", siteID, aclRuleID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdateACLRule(ctx context.Context, siteID, aclRuleID string, request ACLRule) (*ACLRule, error) {
	var response ACLRule
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/sites/%s/acl-rules/%s", siteID, aclRuleID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteACLRule(ctx context.Context, siteID, aclRuleID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/sites/%s/acl-rules/%s", siteID, aclRuleID), nil, nil, nil)
}

func (c *Client) ListACLRules(ctx context.Context, siteID string) ([]ACLRule, error) {
	var rules []ACLRule
	offset := 0

	for {
		var response page[ACLRule]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/acl-rules", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		rules = append(rules, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return rules, nil
}

func (c *Client) ListRadiusProfiles(ctx context.Context, siteID string) ([]RadiusProfile, error) {
	var profiles []RadiusProfile
	offset := 0

	for {
		var response page[RadiusProfile]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/radius/profiles", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		profiles = append(profiles, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return profiles, nil
}

func (c *Client) ListDeviceTags(ctx context.Context, siteID string) ([]DeviceTag, error) {
	var tags []DeviceTag
	offset := 0

	for {
		var response page[DeviceTag]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/device-tags", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		tags = append(tags, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return tags, nil
}
