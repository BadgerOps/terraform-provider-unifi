package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

type DHCPGuarding struct {
	TrustedDHCPServerIPAddresses []string `json:"trustedDhcpServerIpAddresses"`
}

type IPAddressRange struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type IPv4DHCPConfiguration struct {
	Mode                           string          `json:"mode"`
	IPAddressRange                 *IPAddressRange `json:"ipAddressRange,omitempty"`
	DHCPServerIPAddresses          []string        `json:"dhcpServerIpAddresses,omitempty"`
	GatewayIPAddressOverride       *string         `json:"gatewayIpAddressOverride,omitempty"`
	DNSServerIPAddressesOverride   []string        `json:"dnsServerIpAddressesOverride,omitempty"`
	LeaseTimeSeconds               *int64          `json:"leaseTimeSeconds,omitempty"`
	DomainName                     *string         `json:"domainName,omitempty"`
	PingConflictDetectionEnabled   *bool           `json:"pingConflictDetectionEnabled,omitempty"`
	AdditionalNTPServerIPAddresses []string        `json:"ntpServerIpAddresses,omitempty"`
}

type IPv4Configuration struct {
	AutoScaleEnabled         bool                   `json:"autoScaleEnabled"`
	HostIPAddress            string                 `json:"hostIpAddress"`
	PrefixLength             int64                  `json:"prefixLength"`
	AdditionalHostIPSubnets  []string               `json:"additionalHostIpSubnets,omitempty"`
	DHCPConfiguration        *IPv4DHCPConfiguration `json:"dhcpConfiguration,omitempty"`
	NATOutboundConfiguration []map[string]any       `json:"natOutboundIpAddressConfiguration,omitempty"`
}

type Network struct {
	ID                    string             `json:"id,omitempty"`
	Management            string             `json:"management"`
	Name                  string             `json:"name"`
	Enabled               bool               `json:"enabled"`
	VLANID                int64              `json:"vlanId"`
	Default               bool               `json:"default,omitempty"`
	DHCPGuarding          *DHCPGuarding      `json:"dhcpGuarding,omitempty"`
	IsolationEnabled      *bool              `json:"isolationEnabled,omitempty"`
	CellularBackupEnabled *bool              `json:"cellularBackupEnabled,omitempty"`
	ZoneID                *string            `json:"zoneId,omitempty"`
	DeviceID              *string            `json:"deviceId,omitempty"`
	InternetAccessEnabled *bool              `json:"internetAccessEnabled,omitempty"`
	MDNSForwardingEnabled *bool              `json:"mdnsForwardingEnabled,omitempty"`
	IPv4Configuration     *IPv4Configuration `json:"ipv4Configuration,omitempty"`
	IPv6Configuration     map[string]any     `json:"ipv6Configuration,omitempty"`
	Metadata              map[string]any     `json:"metadata,omitempty"`
}

type WifiNetworkReference struct {
	Type      string `json:"type"`
	NetworkID string `json:"networkId,omitempty"`
}

type SAEConfiguration struct {
	AnticloggingThresholdSeconds int64 `json:"anticloggingThresholdSeconds"`
	SyncTimeSeconds              int64 `json:"syncTimeSeconds"`
}

type WifiSecurityConfiguration struct {
	Type                      string            `json:"type"`
	Passphrase                *string           `json:"passphrase,omitempty"`
	PMFMode                   *string           `json:"pmfMode,omitempty"`
	FastRoamingEnabled        *bool             `json:"fastRoamingEnabled,omitempty"`
	GroupRekeyIntervalSeconds *int64            `json:"groupRekeyIntervalSeconds,omitempty"`
	SAEConfiguration          *SAEConfiguration `json:"saeConfiguration,omitempty"`
	WPA3FastRoamingEnabled    *bool             `json:"wpa3FastRoamingEnabled,omitempty"`
}

type WifiBroadcastingDeviceFilter struct {
	Type         string   `json:"type"`
	DeviceTagIDs []string `json:"deviceTagIds,omitempty"`
}

type WifiBroadcast struct {
	ID                                  string                        `json:"id,omitempty"`
	Type                                string                        `json:"type"`
	Name                                string                        `json:"name"`
	Enabled                             bool                          `json:"enabled"`
	Network                             *WifiNetworkReference         `json:"network,omitempty"`
	SecurityConfiguration               *WifiSecurityConfiguration    `json:"securityConfiguration,omitempty"`
	ClientIsolationEnabled              bool                          `json:"clientIsolationEnabled"`
	HideName                            bool                          `json:"hideName"`
	UAPSDEnabled                        bool                          `json:"uapsdEnabled"`
	MulticastToUnicastConversionEnabled bool                          `json:"multicastToUnicastConversionEnabled"`
	BroadcastingFrequenciesGHz          []float64                     `json:"broadcastingFrequenciesGHz,omitempty"`
	AdvertiseDeviceName                 *bool                         `json:"advertiseDeviceName,omitempty"`
	ARPProxyEnabled                     *bool                         `json:"arpProxyEnabled,omitempty"`
	BandSteeringEnabled                 *bool                         `json:"bandSteeringEnabled,omitempty"`
	BSSTransitionEnabled                *bool                         `json:"bssTransitionEnabled,omitempty"`
	MDNSProxyConfiguration              map[string]any                `json:"mdnsProxyConfiguration,omitempty"`
	MulticastFilteringPolicy            map[string]any                `json:"multicastFilteringPolicy,omitempty"`
	BroadcastingDeviceFilter            *WifiBroadcastingDeviceFilter `json:"broadcastingDeviceFilter,omitempty"`
	BasicDataRateKbpsByFrequencyGHz     map[string]any                `json:"basicDataRateKbpsByFrequencyGHz,omitempty"`
	ClientFilteringPolicy               map[string]any                `json:"clientFilteringPolicy,omitempty"`
	BlackoutScheduleConfiguration       map[string]any                `json:"blackoutScheduleConfiguration,omitempty"`
	Metadata                            map[string]any                `json:"metadata,omitempty"`
}

type FirewallZone struct {
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name"`
	NetworkIDs []string       `json:"networkIds"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type FirewallPolicyAction struct {
	Type string `json:"type"`
}

type FirewallPolicyNetworkFilter struct {
	NetworkIDs    []string `json:"networkIds"`
	MatchOpposite bool     `json:"matchOpposite"`
}

type PortMatch struct {
	Type  string `json:"type"`
	Value *int64 `json:"value,omitempty"`
	Start *int64 `json:"start,omitempty"`
	Stop  *int64 `json:"stop,omitempty"`
}

type TrafficMatchingList struct {
	ID    string      `json:"id,omitempty"`
	Type  string      `json:"type"`
	Name  string      `json:"name"`
	Items []PortMatch `json:"items,omitempty"`
}

type FirewallPolicyPortFilter struct {
	Type                  string      `json:"type"`
	MatchOpposite         bool        `json:"matchOpposite"`
	Items                 []PortMatch `json:"items,omitempty"`
	TrafficMatchingListID *string     `json:"trafficMatchingListId,omitempty"`
}

type FirewallPolicyTrafficFilter struct {
	Type          string                       `json:"type"`
	NetworkFilter *FirewallPolicyNetworkFilter `json:"networkFilter,omitempty"`
	PortFilter    *FirewallPolicyPortFilter    `json:"portFilter,omitempty"`
}

type FirewallPolicyEndpoint struct {
	ZoneID        string                       `json:"zoneId"`
	TrafficFilter *FirewallPolicyTrafficFilter `json:"trafficFilter,omitempty"`
}

type FirewallPolicyIPProtocolScope struct {
	IPVersion string `json:"ipVersion"`
}

type FirewallPolicy struct {
	ID                    string                         `json:"id,omitempty"`
	Enabled               bool                           `json:"enabled"`
	Name                  string                         `json:"name"`
	Description           *string                        `json:"description,omitempty"`
	Index                 int64                          `json:"index,omitempty"`
	Action                *FirewallPolicyAction          `json:"action"`
	Source                *FirewallPolicyEndpoint        `json:"source"`
	Destination           *FirewallPolicyEndpoint        `json:"destination"`
	IPProtocolScope       *FirewallPolicyIPProtocolScope `json:"ipProtocolScope"`
	ConnectionStateFilter []string                       `json:"connectionStateFilter,omitempty"`
	IPsecFilter           *string                        `json:"ipsecFilter,omitempty"`
	LoggingEnabled        bool                           `json:"loggingEnabled"`
	Schedule              map[string]any                 `json:"schedule,omitempty"`
	Metadata              map[string]any                 `json:"metadata,omitempty"`
}

func (c *Client) CreateNetwork(ctx context.Context, siteID string, request Network) (*Network, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create network site id: %w", err)
	}

	body, err := transcode[generated.CreateNetworkJSONRequestBody](request)
	if err != nil {
		return nil, fmt.Errorf("translate create network request: %w", err)
	}

	response, err := c.apiClient.CreateNetworkWithResponse(ctx, siteUUID, body)
	if err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}

	created, err := requireJSON(response.StatusCode(), response.Body, response.JSON201, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	network, err := transcode[Network](created)
	if err != nil {
		return nil, fmt.Errorf("translate created network: %w", err)
	}

	return &network, nil
}

func (c *Client) GetNetwork(ctx context.Context, siteID, networkID string) (*Network, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get network site id: %w", err)
	}
	networkUUID, err := parseUUID(networkID)
	if err != nil {
		return nil, fmt.Errorf("get network id: %w", err)
	}

	response, err := c.apiClient.GetNetworkDetailsWithResponse(ctx, siteUUID, networkUUID)
	if err != nil {
		return nil, fmt.Errorf("get network: %w", err)
	}

	details, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	network, err := transcode[Network](details)
	if err != nil {
		return nil, fmt.Errorf("translate network details: %w", err)
	}

	return &network, nil
}

func (c *Client) UpdateNetwork(ctx context.Context, siteID, networkID string, request Network) (*Network, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update network site id: %w", err)
	}
	networkUUID, err := parseUUID(networkID)
	if err != nil {
		return nil, fmt.Errorf("update network id: %w", err)
	}

	body, err := transcode[generated.UpdateNetworkJSONRequestBody](request)
	if err != nil {
		return nil, fmt.Errorf("translate update network request: %w", err)
	}

	response, err := c.apiClient.UpdateNetworkWithResponse(ctx, siteUUID, networkUUID, body)
	if err != nil {
		return nil, fmt.Errorf("update network: %w", err)
	}

	updated, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	network, err := transcode[Network](updated)
	if err != nil {
		return nil, fmt.Errorf("translate updated network: %w", err)
	}

	return &network, nil
}

func (c *Client) DeleteNetwork(ctx context.Context, siteID, networkID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete network site id: %w", err)
	}
	networkUUID, err := parseUUID(networkID)
	if err != nil {
		return fmt.Errorf("delete network id: %w", err)
	}

	response, err := c.apiClient.DeleteNetworkWithResponse(ctx, siteUUID, networkUUID, nil)
	if err != nil {
		return fmt.Errorf("delete network: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) ListNetworks(ctx context.Context, siteID string) ([]Network, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list networks site id: %w", err)
	}

	var networks []Network
	offset := 0

	for {
		response, err := c.apiClient.GetNetworksOverviewPageWithResponse(ctx, siteUUID, &generated.GetNetworksOverviewPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list networks: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]Network](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate network page: %w", err)
		}

		networks = append(networks, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return networks, nil
}

func (c *Client) CreateWifiBroadcast(ctx context.Context, siteID string, request WifiBroadcast) (*WifiBroadcast, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create wifi broadcast site id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode create wifi broadcast request: %w", err)
	}

	response, err := c.apiClient.CreateWifiBroadcastWithBodyWithResponse(ctx, siteUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("create wifi broadcast: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusCreated); err != nil {
		return nil, err
	}

	broadcast, err := decodeBody[WifiBroadcast](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode created wifi broadcast: %w", err)
	}

	return broadcast, nil
}

func (c *Client) GetWifiBroadcast(ctx context.Context, siteID, wifiBroadcastID string) (*WifiBroadcast, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get wifi broadcast site id: %w", err)
	}
	broadcastUUID, err := parseUUID(wifiBroadcastID)
	if err != nil {
		return nil, fmt.Errorf("get wifi broadcast id: %w", err)
	}

	response, err := c.apiClient.GetWifiBroadcastDetailsWithResponse(ctx, siteUUID, broadcastUUID)
	if err != nil {
		return nil, fmt.Errorf("get wifi broadcast: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	broadcast, err := decodeBody[WifiBroadcast](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode wifi broadcast details: %w", err)
	}

	return broadcast, nil
}

func (c *Client) UpdateWifiBroadcast(ctx context.Context, siteID, wifiBroadcastID string, request WifiBroadcast) (*WifiBroadcast, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update wifi broadcast site id: %w", err)
	}
	broadcastUUID, err := parseUUID(wifiBroadcastID)
	if err != nil {
		return nil, fmt.Errorf("update wifi broadcast id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode update wifi broadcast request: %w", err)
	}

	response, err := c.apiClient.UpdateWifiBroadcastWithBodyWithResponse(ctx, siteUUID, broadcastUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("update wifi broadcast: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	broadcast, err := decodeBody[WifiBroadcast](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode updated wifi broadcast: %w", err)
	}

	return broadcast, nil
}

func (c *Client) DeleteWifiBroadcast(ctx context.Context, siteID, wifiBroadcastID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete wifi broadcast site id: %w", err)
	}
	broadcastUUID, err := parseUUID(wifiBroadcastID)
	if err != nil {
		return fmt.Errorf("delete wifi broadcast id: %w", err)
	}

	response, err := c.apiClient.DeleteWifiBroadcastWithResponse(ctx, siteUUID, broadcastUUID, nil)
	if err != nil {
		return fmt.Errorf("delete wifi broadcast: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) CreateFirewallZone(ctx context.Context, siteID string, request FirewallZone) (*FirewallZone, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create firewall zone site id: %w", err)
	}

	body, err := transcode[generated.CreateFirewallZoneJSONRequestBody](request)
	if err != nil {
		return nil, fmt.Errorf("translate create firewall zone request: %w", err)
	}

	response, err := c.apiClient.CreateFirewallZoneWithResponse(ctx, siteUUID, body)
	if err != nil {
		return nil, fmt.Errorf("create firewall zone: %w", err)
	}

	created, err := requireJSON(response.StatusCode(), response.Body, response.JSON201, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	zone, err := transcode[FirewallZone](created)
	if err != nil {
		return nil, fmt.Errorf("translate created firewall zone: %w", err)
	}

	return &zone, nil
}

func (c *Client) GetFirewallZone(ctx context.Context, siteID, firewallZoneID string) (*FirewallZone, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get firewall zone site id: %w", err)
	}
	zoneUUID, err := parseUUID(firewallZoneID)
	if err != nil {
		return nil, fmt.Errorf("get firewall zone id: %w", err)
	}

	response, err := c.apiClient.GetFirewallZoneWithResponse(ctx, siteUUID, zoneUUID)
	if err != nil {
		return nil, fmt.Errorf("get firewall zone: %w", err)
	}

	details, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	zone, err := transcode[FirewallZone](details)
	if err != nil {
		return nil, fmt.Errorf("translate firewall zone: %w", err)
	}

	return &zone, nil
}

func (c *Client) UpdateFirewallZone(ctx context.Context, siteID, firewallZoneID string, request FirewallZone) (*FirewallZone, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update firewall zone site id: %w", err)
	}
	zoneUUID, err := parseUUID(firewallZoneID)
	if err != nil {
		return nil, fmt.Errorf("update firewall zone id: %w", err)
	}

	body, err := transcode[generated.UpdateFirewallZoneJSONRequestBody](request)
	if err != nil {
		return nil, fmt.Errorf("translate update firewall zone request: %w", err)
	}

	response, err := c.apiClient.UpdateFirewallZoneWithResponse(ctx, siteUUID, zoneUUID, body)
	if err != nil {
		return nil, fmt.Errorf("update firewall zone: %w", err)
	}

	updated, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	zone, err := transcode[FirewallZone](updated)
	if err != nil {
		return nil, fmt.Errorf("translate updated firewall zone: %w", err)
	}

	return &zone, nil
}

func (c *Client) DeleteFirewallZone(ctx context.Context, siteID, firewallZoneID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete firewall zone site id: %w", err)
	}
	zoneUUID, err := parseUUID(firewallZoneID)
	if err != nil {
		return fmt.Errorf("delete firewall zone id: %w", err)
	}

	response, err := c.apiClient.DeleteFirewallZoneWithResponse(ctx, siteUUID, zoneUUID)
	if err != nil {
		return fmt.Errorf("delete firewall zone: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) ListFirewallZones(ctx context.Context, siteID string) ([]FirewallZone, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list firewall zones site id: %w", err)
	}

	var zones []FirewallZone
	offset := 0

	for {
		response, err := c.apiClient.GetFirewallZonesWithResponse(ctx, siteUUID, &generated.GetFirewallZonesParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list firewall zones: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]FirewallZone](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate firewall zone page: %w", err)
		}

		zones = append(zones, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return zones, nil
}

func (c *Client) CreateFirewallPolicy(ctx context.Context, siteID string, request FirewallPolicy) (*FirewallPolicy, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create firewall policy site id: %w", err)
	}

	body, err := transcode[generated.CreateFirewallPolicyJSONRequestBody](request)
	if err != nil {
		return nil, fmt.Errorf("translate create firewall policy request: %w", err)
	}

	response, err := c.apiClient.CreateFirewallPolicyWithResponse(ctx, siteUUID, body)
	if err != nil {
		return nil, fmt.Errorf("create firewall policy: %w", err)
	}

	created, err := requireJSON(response.StatusCode(), response.Body, response.JSON201, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	policy, err := transcode[FirewallPolicy](created)
	if err != nil {
		return nil, fmt.Errorf("translate created firewall policy: %w", err)
	}

	return &policy, nil
}

func (c *Client) GetFirewallPolicy(ctx context.Context, siteID, firewallPolicyID string) (*FirewallPolicy, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy site id: %w", err)
	}
	policyUUID, err := parseUUID(firewallPolicyID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy id: %w", err)
	}

	response, err := c.apiClient.GetFirewallPolicyWithResponse(ctx, siteUUID, policyUUID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy: %w", err)
	}

	details, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	policy, err := transcode[FirewallPolicy](details)
	if err != nil {
		return nil, fmt.Errorf("translate firewall policy: %w", err)
	}

	return &policy, nil
}

func (c *Client) UpdateFirewallPolicy(ctx context.Context, siteID, firewallPolicyID string, request FirewallPolicy) (*FirewallPolicy, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy site id: %w", err)
	}
	policyUUID, err := parseUUID(firewallPolicyID)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy id: %w", err)
	}

	body, err := transcode[generated.UpdateFirewallPolicyJSONRequestBody](request)
	if err != nil {
		return nil, fmt.Errorf("translate update firewall policy request: %w", err)
	}

	response, err := c.apiClient.UpdateFirewallPolicyWithResponse(ctx, siteUUID, policyUUID, body)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy: %w", err)
	}

	updated, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	policy, err := transcode[FirewallPolicy](updated)
	if err != nil {
		return nil, fmt.Errorf("translate updated firewall policy: %w", err)
	}

	return &policy, nil
}

func (c *Client) DeleteFirewallPolicy(ctx context.Context, siteID, firewallPolicyID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete firewall policy site id: %w", err)
	}
	policyUUID, err := parseUUID(firewallPolicyID)
	if err != nil {
		return fmt.Errorf("delete firewall policy id: %w", err)
	}

	response, err := c.apiClient.DeleteFirewallPolicyWithResponse(ctx, siteUUID, policyUUID)
	if err != nil {
		return fmt.Errorf("delete firewall policy: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) CreateTrafficMatchingList(ctx context.Context, siteID string, request TrafficMatchingList) (*TrafficMatchingList, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("create traffic matching list site id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode create traffic matching list request: %w", err)
	}

	response, err := c.apiClient.CreateTrafficMatchingListWithBodyWithResponse(ctx, siteUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("create traffic matching list: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusCreated); err != nil {
		return nil, err
	}

	list, err := decodeBody[TrafficMatchingList](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode created traffic matching list: %w", err)
	}

	return list, nil
}

func (c *Client) GetTrafficMatchingList(ctx context.Context, siteID, trafficMatchingListID string) (*TrafficMatchingList, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get traffic matching list site id: %w", err)
	}
	listUUID, err := parseUUID(trafficMatchingListID)
	if err != nil {
		return nil, fmt.Errorf("get traffic matching list id: %w", err)
	}

	response, err := c.apiClient.GetTrafficMatchingListWithResponse(ctx, siteUUID, listUUID)
	if err != nil {
		return nil, fmt.Errorf("get traffic matching list: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	list, err := decodeBody[TrafficMatchingList](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode traffic matching list: %w", err)
	}

	return list, nil
}

func (c *Client) UpdateTrafficMatchingList(ctx context.Context, siteID, trafficMatchingListID string, request TrafficMatchingList) (*TrafficMatchingList, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("update traffic matching list site id: %w", err)
	}
	listUUID, err := parseUUID(trafficMatchingListID)
	if err != nil {
		return nil, fmt.Errorf("update traffic matching list id: %w", err)
	}

	body, err := jsonBodyReader(request)
	if err != nil {
		return nil, fmt.Errorf("encode update traffic matching list request: %w", err)
	}

	response, err := c.apiClient.UpdateTrafficMatchingListWithBodyWithResponse(ctx, siteUUID, listUUID, "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("update traffic matching list: %w", err)
	}

	if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
		return nil, err
	}

	list, err := decodeBody[TrafficMatchingList](response.Body)
	if err != nil {
		return nil, fmt.Errorf("decode updated traffic matching list: %w", err)
	}

	return list, nil
}

func (c *Client) DeleteTrafficMatchingList(ctx context.Context, siteID, trafficMatchingListID string) error {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return fmt.Errorf("delete traffic matching list site id: %w", err)
	}
	listUUID, err := parseUUID(trafficMatchingListID)
	if err != nil {
		return fmt.Errorf("delete traffic matching list id: %w", err)
	}

	response, err := c.apiClient.DeleteTrafficMatchingListWithResponse(ctx, siteUUID, listUUID)
	if err != nil {
		return fmt.Errorf("delete traffic matching list: %w", err)
	}

	return requireStatus(response.StatusCode(), response.Body, http.StatusOK, http.StatusNoContent)
}

func (c *Client) ListTrafficMatchingLists(ctx context.Context, siteID string) ([]TrafficMatchingList, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list traffic matching lists site id: %w", err)
	}

	var lists []TrafficMatchingList
	offset := 0

	for {
		response, err := c.apiClient.GetTrafficMatchingListsWithResponse(ctx, siteUUID, &generated.GetTrafficMatchingListsParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list traffic matching lists: %w", err)
		}

		if err := requireStatus(response.StatusCode(), response.Body, http.StatusOK); err != nil {
			return nil, err
		}

		page, err := decodeBody[page[TrafficMatchingList]](response.Body)
		if err != nil {
			return nil, fmt.Errorf("decode traffic matching list page: %w", err)
		}

		lists = append(lists, page.Data...)
		offset += len(page.Data)

		if len(page.Data) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return lists, nil
}
