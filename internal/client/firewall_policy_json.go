package client

import "encoding/json"

func (f FirewallPolicyTrafficFilter) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type                      string                                   `json:"type"`
		NetworkFilter             *FirewallPolicyNetworkFilter             `json:"networkFilter,omitempty"`
		PortFilter                *FirewallPolicyPortFilter                `json:"portFilter,omitempty"`
		IPAddressFilter           *FirewallPolicyIPAddressFilter           `json:"ipAddressFilter,omitempty"`
		IPv6IIDFilter             *FirewallPolicyIPv6IIDFilter             `json:"ipv6IidFilter,omitempty"`
		RegionFilter              *FirewallPolicyRegionFilter              `json:"regionFilter,omitempty"`
		VPNServerFilter           *FirewallPolicyVPNServerFilter           `json:"vpnServerFilter,omitempty"`
		SiteToSiteVPNTunnelFilter *FirewallPolicySiteToSiteVPNTunnelFilter `json:"siteToSiteVpnTunnelFilter,omitempty"`
		DomainFilter              *FirewallPolicyDomainFilter              `json:"domainFilter,omitempty"`
		ApplicationFilter         *FirewallPolicyApplicationFilter         `json:"applicationFilter,omitempty"`
		ApplicationCategoryFilter *FirewallPolicyApplicationCategoryFilter `json:"applicationCategoryFilter,omitempty"`
	}

	payload := map[string]any{
		"type": f.Type,
	}

	if encoded, err := json.Marshal(alias{
		Type:                      f.Type,
		NetworkFilter:             f.NetworkFilter,
		PortFilter:                f.PortFilter,
		IPAddressFilter:           f.IPAddressFilter,
		IPv6IIDFilter:             f.IPv6IIDFilter,
		RegionFilter:              f.RegionFilter,
		VPNServerFilter:           f.VPNServerFilter,
		SiteToSiteVPNTunnelFilter: f.SiteToSiteVPNTunnelFilter,
		DomainFilter:              f.DomainFilter,
		ApplicationFilter:         f.ApplicationFilter,
		ApplicationCategoryFilter: f.ApplicationCategoryFilter,
	}); err != nil {
		return nil, err
	} else if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, err
	}

	if f.MACAddress != nil {
		payload["macAddressFilter"] = *f.MACAddress
	} else if f.MACAddressFilter != nil {
		payload["macAddressFilter"] = f.MACAddressFilter
	}

	return json.Marshal(payload)
}

func (f *FirewallPolicyTrafficFilter) UnmarshalJSON(data []byte) error {
	type alias struct {
		Type                      string                                   `json:"type"`
		NetworkFilter             *FirewallPolicyNetworkFilter             `json:"networkFilter,omitempty"`
		PortFilter                *FirewallPolicyPortFilter                `json:"portFilter,omitempty"`
		IPAddressFilter           *FirewallPolicyIPAddressFilter           `json:"ipAddressFilter,omitempty"`
		IPv6IIDFilter             *FirewallPolicyIPv6IIDFilter             `json:"ipv6IidFilter,omitempty"`
		RegionFilter              *FirewallPolicyRegionFilter              `json:"regionFilter,omitempty"`
		VPNServerFilter           *FirewallPolicyVPNServerFilter           `json:"vpnServerFilter,omitempty"`
		SiteToSiteVPNTunnelFilter *FirewallPolicySiteToSiteVPNTunnelFilter `json:"siteToSiteVpnTunnelFilter,omitempty"`
		DomainFilter              *FirewallPolicyDomainFilter              `json:"domainFilter,omitempty"`
		ApplicationFilter         *FirewallPolicyApplicationFilter         `json:"applicationFilter,omitempty"`
		ApplicationCategoryFilter *FirewallPolicyApplicationCategoryFilter `json:"applicationCategoryFilter,omitempty"`
		MACAddressFilter          json.RawMessage                          `json:"macAddressFilter,omitempty"`
	}

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*f = FirewallPolicyTrafficFilter{
		Type:                      decoded.Type,
		NetworkFilter:             decoded.NetworkFilter,
		PortFilter:                decoded.PortFilter,
		IPAddressFilter:           decoded.IPAddressFilter,
		IPv6IIDFilter:             decoded.IPv6IIDFilter,
		RegionFilter:              decoded.RegionFilter,
		VPNServerFilter:           decoded.VPNServerFilter,
		SiteToSiteVPNTunnelFilter: decoded.SiteToSiteVPNTunnelFilter,
		DomainFilter:              decoded.DomainFilter,
		ApplicationFilter:         decoded.ApplicationFilter,
		ApplicationCategoryFilter: decoded.ApplicationCategoryFilter,
	}

	if len(decoded.MACAddressFilter) == 0 {
		return nil
	}

	var macAddress string
	if err := json.Unmarshal(decoded.MACAddressFilter, &macAddress); err == nil {
		f.MACAddress = &macAddress
		return nil
	}

	var macAddresses FirewallPolicyMACAddressListFilter
	if err := json.Unmarshal(decoded.MACAddressFilter, &macAddresses); err != nil {
		return err
	}
	f.MACAddressFilter = &macAddresses
	return nil
}
