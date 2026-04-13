package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

type ACLRuleOrdering struct {
	OrderedACLRuleIDs []string `json:"orderedAclRuleIds"`
}

type FirewallPolicyOrderedIDs struct {
	AfterSystemDefined  []string `json:"afterSystemDefined"`
	BeforeSystemDefined []string `json:"beforeSystemDefined"`
}

type FirewallPolicyOrdering struct {
	OrderedFirewallPolicyIDs FirewallPolicyOrderedIDs `json:"orderedFirewallPolicyIds"`
}

func (c *Client) GetACLRuleOrdering(ctx context.Context, siteID string) (*ACLRuleOrdering, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get acl rule ordering site id: %w", err)
	}

	response, err := c.apiClient.GetAclRuleOrderingWithResponse(ctx, siteUUID)
	if err != nil {
		return nil, fmt.Errorf("get acl rule ordering: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	ordering, err := decodeBody[ACLRuleOrdering](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode acl rule ordering: %w", err)
	}

	return ordering, nil
}

func (c *Client) UpdateACLRuleOrdering(ctx context.Context, siteID string, request ACLRuleOrdering) (*ACLRuleOrdering, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update acl rule ordering site id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode update acl rule ordering request: %w", err)
	}

	response, err := c.apiClient.UpdateAclRuleOrderingWithBodyWithResponse(ctx, siteUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("update acl rule ordering: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	ordering, err := decodeBody[ACLRuleOrdering](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode updated acl rule ordering: %w", err)
	}

	return ordering, nil
}

func (c *Client) GetFirewallPolicyOrdering(ctx context.Context, siteID, sourceZoneID, destinationZoneID string) (*FirewallPolicyOrdering, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy ordering site id: %w", err)
	}
	sourceZoneUUID, err := parseUUID(sourceZoneID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy ordering source zone id: %w", err)
	}
	destinationZoneUUID, err := parseUUID(destinationZoneID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy ordering destination zone id: %w", err)
	}

	response, err := c.apiClient.GetFirewallPolicyOrderingWithResponse(ctx, siteUUID, &generated.GetFirewallPolicyOrderingParams{
		SourceFirewallZoneId:      sourceZoneUUID,
		DestinationFirewallZoneId: destinationZoneUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("get firewall policy ordering: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	ordering, err := decodeBody[FirewallPolicyOrdering](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode firewall policy ordering: %w", err)
	}

	return ordering, nil
}

func (c *Client) UpdateFirewallPolicyOrdering(ctx context.Context, siteID, sourceZoneID, destinationZoneID string, request FirewallPolicyOrdering) (*FirewallPolicyOrdering, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy ordering site id: %w", err)
	}
	sourceZoneUUID, err := parseUUID(sourceZoneID)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy ordering source zone id: %w", err)
	}
	destinationZoneUUID, err := parseUUID(destinationZoneID)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy ordering destination zone id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode update firewall policy ordering request: %w", err)
	}

	response, err := c.apiClient.UpdateFirewallPolicyOrderingWithBodyWithResponse(ctx, siteUUID, &generated.UpdateFirewallPolicyOrderingParams{
		SourceFirewallZoneId:      sourceZoneUUID,
		DestinationFirewallZoneId: destinationZoneUUID,
	}, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy ordering: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	ordering, err := decodeBody[FirewallPolicyOrdering](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode updated firewall policy ordering: %w", err)
	}

	return ordering, nil
}
