package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
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
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create dns policy site id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode create dns policy request: %w", err)
	}

	response, err := c.apiClient.CreateDnsPolicyWithBodyWithResponse(ctx, siteUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("create dns policy: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusCreated); err != nil {
		return nil, err
	}

	policy, err := decodeBody[DNSPolicy](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode created dns policy: %w", err)
	}

	return policy, nil
}

func (c *Client) GetDNSPolicy(ctx context.Context, siteID, dnsPolicyID string) (*DNSPolicy, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get dns policy site id: %w", err)
	}
	policyUUID, err := parseUUID(dnsPolicyID)
	if err != nil {
		return nil, fmt.Errorf("get dns policy id: %w", err)
	}

	response, err := c.apiClient.GetDnsPolicyWithResponse(ctx, siteUUID, policyUUID)
	if err != nil {
		return nil, fmt.Errorf("get dns policy: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	policy, err := decodeBody[DNSPolicy](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode dns policy: %w", err)
	}

	return policy, nil
}

func (c *Client) UpdateDNSPolicy(ctx context.Context, siteID, dnsPolicyID string, request DNSPolicy) (*DNSPolicy, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update dns policy site id: %w", err)
	}
	policyUUID, err := parseUUID(dnsPolicyID)
	if err != nil {
		return nil, fmt.Errorf("update dns policy id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode update dns policy request: %w", err)
	}

	response, err := c.apiClient.UpdateDnsPolicyWithBodyWithResponse(ctx, siteUUID, policyUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("update dns policy: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	policy, err := decodeBody[DNSPolicy](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode updated dns policy: %w", err)
	}

	return policy, nil
}

func (c *Client) DeleteDNSPolicy(ctx context.Context, siteID, dnsPolicyID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete dns policy site id: %w", err)
	}
	policyUUID, err := parseUUID(dnsPolicyID)
	if err != nil {
		return fmt.Errorf("delete dns policy id: %w", err)
	}

	response, err := c.apiClient.DeleteDnsPolicyWithResponse(ctx, siteUUID, policyUUID)
	if err != nil {
		return fmt.Errorf("delete dns policy: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) ListDNSPolicies(ctx context.Context, siteID string) ([]DNSPolicy, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list dns policies site id: %w", err)
	}

	var policies []DNSPolicy
	offset := 0

	for {
		response, err := c.apiClient.GetDnsPolicyPageWithResponse(ctx, siteUUID, &generated.GetDnsPolicyPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list dns policies: %w", err)
		}

		if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
			return nil, err
		}

		page, err := decodeBody[page[DNSPolicy]](response.Body)
		if err != nil {
			return nil, fmt.Errorf("decode dns policy page: %w", err)
		}

		policies = append(policies, page.Data...)
		offset += len(page.Data)

		if len(page.Data) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return policies, nil
}

func (c *Client) CreateACLRule(ctx context.Context, siteID string, request ACLRule) (*ACLRule, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create acl rule site id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode create acl rule request: %w", err)
	}

	response, err := c.apiClient.CreateAclRuleWithBodyWithResponse(ctx, siteUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("create acl rule: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusCreated); err != nil {
		return nil, err
	}

	rule, err := decodeBody[ACLRule](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode created acl rule: %w", err)
	}

	return rule, nil
}

func (c *Client) GetACLRule(ctx context.Context, siteID, aclRuleID string) (*ACLRule, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get acl rule site id: %w", err)
	}
	ruleUUID, err := parseUUID(aclRuleID)
	if err != nil {
		return nil, fmt.Errorf("get acl rule id: %w", err)
	}

	response, err := c.apiClient.GetAclRuleWithResponse(ctx, siteUUID, ruleUUID)
	if err != nil {
		return nil, fmt.Errorf("get acl rule: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	rule, err := decodeBody[ACLRule](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode acl rule: %w", err)
	}

	return rule, nil
}

func (c *Client) UpdateACLRule(ctx context.Context, siteID, aclRuleID string, request ACLRule) (*ACLRule, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update acl rule site id: %w", err)
	}
	ruleUUID, err := parseUUID(aclRuleID)
	if err != nil {
		return nil, fmt.Errorf("update acl rule id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode update acl rule request: %w", err)
	}

	response, err := c.apiClient.UpdateAclRuleWithBodyWithResponse(ctx, siteUUID, ruleUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("update acl rule: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	rule, err := decodeBody[ACLRule](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode updated acl rule: %w", err)
	}

	return rule, nil
}

func (c *Client) DeleteACLRule(ctx context.Context, siteID, aclRuleID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete acl rule site id: %w", err)
	}
	ruleUUID, err := parseUUID(aclRuleID)
	if err != nil {
		return fmt.Errorf("delete acl rule id: %w", err)
	}

	response, err := c.apiClient.DeleteAclRuleWithResponse(ctx, siteUUID, ruleUUID)
	if err != nil {
		return fmt.Errorf("delete acl rule: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) ListACLRules(ctx context.Context, siteID string) ([]ACLRule, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list acl rules site id: %w", err)
	}

	var rules []ACLRule
	offset := 0

	for {
		response, err := c.apiClient.GetAclRulePageWithResponse(ctx, siteUUID, &generated.GetAclRulePageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list acl rules: %w", err)
		}

		if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
			return nil, err
		}

		page, err := decodeBody[page[ACLRule]](response.Body)
		if err != nil {
			return nil, fmt.Errorf("decode acl rule page: %w", err)
		}

		rules = append(rules, page.Data...)
		offset += len(page.Data)

		if len(page.Data) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return rules, nil
}

func (c *Client) ListRadiusProfiles(ctx context.Context, siteID string) ([]RadiusProfile, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list radius profiles site id: %w", err)
	}

	var profiles []RadiusProfile
	offset := 0

	for {
		response, err := c.apiClient.GetRadiusProfileOverviewPageWithResponse(ctx, siteUUID, &generated.GetRadiusProfileOverviewPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list radius profiles: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]RadiusProfile](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate radius profile page: %w", err)
		}

		profiles = append(profiles, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return profiles, nil
}

func (c *Client) ListDeviceTags(ctx context.Context, siteID string) ([]DeviceTag, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list device tags site id: %w", err)
	}

	var tags []DeviceTag
	offset := 0

	for {
		response, err := c.apiClient.GetDeviceTagPageWithResponse(ctx, siteUUID, &generated.GetDeviceTagPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list device tags: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]DeviceTag](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate device tag page: %w", err)
		}

		tags = append(tags, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return tags, nil
}
