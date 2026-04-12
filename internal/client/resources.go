package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

type WifiBroadcast struct {
	ID                                  string                     `json:"id,omitempty"`
	Type                                string                     `json:"type"`
	Name                                string                     `json:"name"`
	Enabled                             bool                       `json:"enabled"`
	Network                             *WifiNetworkReference      `json:"network,omitempty"`
	SecurityConfiguration               *WifiSecurityConfiguration `json:"securityConfiguration,omitempty"`
	ClientIsolationEnabled              bool                       `json:"clientIsolationEnabled"`
	HideName                            bool                       `json:"hideName"`
	UAPSDEnabled                        bool                       `json:"uapsdEnabled"`
	MulticastToUnicastConversionEnabled bool                       `json:"multicastToUnicastConversionEnabled"`
	BroadcastingFrequenciesGHz          []float64                  `json:"broadcastingFrequenciesGHz,omitempty"`
	AdvertiseDeviceName                 *bool                      `json:"advertiseDeviceName,omitempty"`
	ARPProxyEnabled                     *bool                      `json:"arpProxyEnabled,omitempty"`
	BandSteeringEnabled                 *bool                      `json:"bandSteeringEnabled,omitempty"`
	BSSTransitionEnabled                *bool                      `json:"bssTransitionEnabled,omitempty"`
	MDNSProxyConfiguration              map[string]any             `json:"mdnsProxyConfiguration,omitempty"`
	MulticastFilteringPolicy            map[string]any             `json:"multicastFilteringPolicy,omitempty"`
	BroadcastingDeviceFilter            map[string]any             `json:"broadcastingDeviceFilter,omitempty"`
	BasicDataRateKbpsByFrequencyGHz     map[string]any             `json:"basicDataRateKbpsByFrequencyGHz,omitempty"`
	ClientFilteringPolicy               map[string]any             `json:"clientFilteringPolicy,omitempty"`
	BlackoutScheduleConfiguration       map[string]any             `json:"blackoutScheduleConfiguration,omitempty"`
	Metadata                            map[string]any             `json:"metadata,omitempty"`
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

type FirewallPolicyNetworkTrafficFilter struct {
	Type          string                      `json:"type"`
	NetworkFilter FirewallPolicyNetworkFilter `json:"networkFilter"`
}

type FirewallPolicyEndpoint struct {
	ZoneID        string                              `json:"zoneId"`
	TrafficFilter *FirewallPolicyNetworkTrafficFilter `json:"trafficFilter,omitempty"`
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
	var response Network
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sites/%s/networks", siteID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetNetwork(ctx context.Context, siteID, networkID string) (*Network, error) {
	var response Network
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/networks/%s", siteID, networkID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdateNetwork(ctx context.Context, siteID, networkID string, request Network) (*Network, error) {
	var response Network
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/sites/%s/networks/%s", siteID, networkID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteNetwork(ctx context.Context, siteID, networkID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/sites/%s/networks/%s", siteID, networkID), nil, nil, nil)
}

func (c *Client) CreateWifiBroadcast(ctx context.Context, siteID string, request WifiBroadcast) (*WifiBroadcast, error) {
	var response WifiBroadcast
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sites/%s/wifi/broadcasts", siteID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetWifiBroadcast(ctx context.Context, siteID, wifiBroadcastID string) (*WifiBroadcast, error) {
	var response WifiBroadcast
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/wifi/broadcasts/%s", siteID, wifiBroadcastID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdateWifiBroadcast(ctx context.Context, siteID, wifiBroadcastID string, request WifiBroadcast) (*WifiBroadcast, error) {
	var response WifiBroadcast
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/sites/%s/wifi/broadcasts/%s", siteID, wifiBroadcastID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteWifiBroadcast(ctx context.Context, siteID, wifiBroadcastID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/sites/%s/wifi/broadcasts/%s", siteID, wifiBroadcastID), nil, nil, nil)
}

func (c *Client) CreateFirewallZone(ctx context.Context, siteID string, request FirewallZone) (*FirewallZone, error) {
	var response FirewallZone
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sites/%s/firewall/zones", siteID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetFirewallZone(ctx context.Context, siteID, firewallZoneID string) (*FirewallZone, error) {
	var response FirewallZone
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/firewall/zones/%s", siteID, firewallZoneID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdateFirewallZone(ctx context.Context, siteID, firewallZoneID string, request FirewallZone) (*FirewallZone, error) {
	var response FirewallZone
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/sites/%s/firewall/zones/%s", siteID, firewallZoneID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteFirewallZone(ctx context.Context, siteID, firewallZoneID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/sites/%s/firewall/zones/%s", siteID, firewallZoneID), nil, nil, nil)
}

func (c *Client) CreateFirewallPolicy(ctx context.Context, siteID string, request FirewallPolicy) (*FirewallPolicy, error) {
	var response FirewallPolicy
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sites/%s/firewall/policies", siteID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetFirewallPolicy(ctx context.Context, siteID, firewallPolicyID string) (*FirewallPolicy, error) {
	var response FirewallPolicy
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/firewall/policies/%s", siteID, firewallPolicyID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdateFirewallPolicy(ctx context.Context, siteID, firewallPolicyID string, request FirewallPolicy) (*FirewallPolicy, error) {
	var response FirewallPolicy
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/sites/%s/firewall/policies/%s", siteID, firewallPolicyID), nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteFirewallPolicy(ctx context.Context, siteID, firewallPolicyID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/sites/%s/firewall/policies/%s", siteID, firewallPolicyID), nil, nil, nil)
}

func (c *Client) GetWithQuery(ctx context.Context, requestPath string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, requestPath, query, nil, out)
}
