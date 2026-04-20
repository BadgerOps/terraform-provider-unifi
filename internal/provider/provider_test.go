package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New("test")()),
}

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		if _, err := os.Stat("/usr/bin/terraform"); err == nil {
			_ = os.Setenv("TF_ACC_TERRAFORM_PATH", "/usr/bin/terraform")
		}
	}

	os.Exit(m.Run())
}

type mockUniFiAPI struct {
	server *httptest.Server

	mu sync.Mutex

	nextID int

	siteID                        string
	existingNetworkID             string
	existingZoneID                string
	existingFirewallPolicyID      string
	existingTrafficMatchingListID string
	existingWifiBroadcastID       string
	existingDNSPolicyID           string
	existingACLRuleID             string
	existingRadiusProfileID       string
	existingDeviceTagID           string
	existingVPNServerID           string
	existingSiteToSiteTunnelID    string
	existingDPIApplicationID      int64
	existingDPICategoryID         int64
	existingCountryCode           string
	existingSwitchDeviceID        string
	existingWANID                 string
	existingSwitchStackID         string
	existingMcLagDomainID         string
	existingSwitchStackLagID      string
	existingMcLagID               string
	existingDHCPReservationMAC    string
	existingAdoptedDeviceMAC      string

	sites                    map[string]client.Site
	networks                 map[string]map[string]client.Network
	wifiBroadcasts           map[string]map[string]client.WifiBroadcast
	firewallZones            map[string]map[string]client.FirewallZone
	firewallPolicies         map[string]map[string]client.FirewallPolicy
	firewallPolicyOrderings  map[string]map[string]client.FirewallPolicyOrdering
	trafficMatchingLists     map[string]map[string]client.TrafficMatchingList
	radiusProfiles           map[string]map[string]client.RadiusProfile
	deviceTags               map[string]map[string]client.DeviceTag
	vpnServers               map[string]map[string]client.VPNServer
	siteToSiteVPNTunnels     map[string]map[string]client.SiteToSiteVPNTunnel
	devices                  map[string]map[string]client.Device
	dpiApplications          map[int64]client.DPIApplication
	dpiApplicationCategories map[int64]client.DPIApplicationCategory
	countries                map[string]client.Country
	dnsPolicies              map[string]map[string]client.DNSPolicy
	aclRules                 map[string]map[string]client.ACLRule
	aclRuleOrderings         map[string]client.ACLRuleOrdering
	wans                     map[string]map[string]client.WAN
	switchStacks             map[string]map[string]client.SwitchStack
	mcLagDomains             map[string]map[string]client.McLagDomain
	lags                     map[string]map[string]client.Lag
	dhcpReservations         map[string]map[string]client.DHCPReservation
}

func newMockUniFiAPI(t *testing.T) *mockUniFiAPI {
	t.Helper()

	api := &mockUniFiAPI{
		nextID:                   1,
		sites:                    make(map[string]client.Site),
		networks:                 make(map[string]map[string]client.Network),
		wifiBroadcasts:           make(map[string]map[string]client.WifiBroadcast),
		firewallZones:            make(map[string]map[string]client.FirewallZone),
		firewallPolicies:         make(map[string]map[string]client.FirewallPolicy),
		firewallPolicyOrderings:  make(map[string]map[string]client.FirewallPolicyOrdering),
		trafficMatchingLists:     make(map[string]map[string]client.TrafficMatchingList),
		radiusProfiles:           make(map[string]map[string]client.RadiusProfile),
		deviceTags:               make(map[string]map[string]client.DeviceTag),
		vpnServers:               make(map[string]map[string]client.VPNServer),
		siteToSiteVPNTunnels:     make(map[string]map[string]client.SiteToSiteVPNTunnel),
		devices:                  make(map[string]map[string]client.Device),
		dpiApplications:          make(map[int64]client.DPIApplication),
		dpiApplicationCategories: make(map[int64]client.DPIApplicationCategory),
		countries:                make(map[string]client.Country),
		dnsPolicies:              make(map[string]map[string]client.DNSPolicy),
		aclRules:                 make(map[string]map[string]client.ACLRule),
		aclRuleOrderings:         make(map[string]client.ACLRuleOrdering),
		wans:                     make(map[string]map[string]client.WAN),
		switchStacks:             make(map[string]map[string]client.SwitchStack),
		mcLagDomains:             make(map[string]map[string]client.McLagDomain),
		lags:                     make(map[string]map[string]client.Lag),
		dhcpReservations:         make(map[string]map[string]client.DHCPReservation),
	}

	api.siteID = api.newID()
	api.sites[api.siteID] = client.Site{
		ID:                api.siteID,
		Name:              "Default",
		InternalReference: "default",
	}
	api.networks[api.siteID] = make(map[string]client.Network)
	api.wifiBroadcasts[api.siteID] = make(map[string]client.WifiBroadcast)
	api.firewallZones[api.siteID] = make(map[string]client.FirewallZone)
	api.firewallPolicies[api.siteID] = make(map[string]client.FirewallPolicy)
	api.firewallPolicyOrderings[api.siteID] = make(map[string]client.FirewallPolicyOrdering)
	api.trafficMatchingLists[api.siteID] = make(map[string]client.TrafficMatchingList)
	api.radiusProfiles[api.siteID] = make(map[string]client.RadiusProfile)
	api.deviceTags[api.siteID] = make(map[string]client.DeviceTag)
	api.vpnServers[api.siteID] = make(map[string]client.VPNServer)
	api.siteToSiteVPNTunnels[api.siteID] = make(map[string]client.SiteToSiteVPNTunnel)
	api.devices[api.siteID] = make(map[string]client.Device)
	api.dnsPolicies[api.siteID] = make(map[string]client.DNSPolicy)
	api.aclRules[api.siteID] = make(map[string]client.ACLRule)
	api.wans[api.siteID] = make(map[string]client.WAN)
	api.switchStacks[api.siteID] = make(map[string]client.SwitchStack)
	api.mcLagDomains[api.siteID] = make(map[string]client.McLagDomain)
	api.lags[api.siteID] = make(map[string]client.Lag)
	api.dhcpReservations["default"] = make(map[string]client.DHCPReservation)

	existingNetwork := client.Network{
		ID:                    api.newID(),
		Management:            "GATEWAY",
		Name:                  "existing-network",
		Enabled:               true,
		VLANID:                200,
		Default:               false,
		IsolationEnabled:      boolPtr(false),
		CellularBackupEnabled: boolPtr(false),
		InternetAccessEnabled: boolPtr(true),
		MDNSForwardingEnabled: boolPtr(false),
		IPv4Configuration: &client.IPv4Configuration{
			AutoScaleEnabled: false,
			HostIPAddress:    "10.200.0.1",
			PrefixLength:     24,
		},
	}
	api.existingNetworkID = existingNetwork.ID
	api.networks[api.siteID][existingNetwork.ID] = existingNetwork

	existingZone := client.FirewallZone{
		ID:         api.newID(),
		Name:       "existing-zone",
		NetworkIDs: []string{existingNetwork.ID},
	}
	api.existingZoneID = existingZone.ID
	api.firewallZones[api.siteID][existingZone.ID] = existingZone

	existingFirewallPolicy := client.FirewallPolicy{
		ID:          api.newID(),
		Enabled:     true,
		Name:        "existing-policy",
		Description: stringPtr("existing firewall policy"),
		Action: &client.FirewallPolicyAction{
			Type:               "ALLOW",
			AllowReturnTraffic: boolPtr(true),
		},
		Source: &client.FirewallPolicyEndpoint{
			ZoneID: existingZone.ID,
			TrafficFilter: &client.FirewallPolicyTrafficFilter{
				Type: "NETWORK",
				NetworkFilter: &client.FirewallPolicyNetworkFilter{
					NetworkIDs:    []string{existingNetwork.ID},
					MatchOpposite: false,
				},
			},
		},
		Destination: &client.FirewallPolicyEndpoint{
			ZoneID: existingZone.ID,
			TrafficFilter: &client.FirewallPolicyTrafficFilter{
				Type: "PORT",
				PortFilter: &client.FirewallPolicyPortFilter{
					Type:          "PORTS",
					MatchOpposite: false,
					Items: []client.PortMatch{
						portNumberMatch(443),
					},
				},
			},
		},
		IPProtocolScope: &client.FirewallPolicyIPProtocolScope{
			IPVersion: "IPV4",
			ProtocolFilter: &client.FirewallPolicyProtocolFilter{
				Type:          "NAMED_PROTOCOL",
				Protocol:      &client.FirewallPolicyNamedProtocol{Name: "tcp"},
				MatchOpposite: boolPtr(false),
			},
		},
		ConnectionStateFilter: []string{"NEW", "ESTABLISHED"},
		LoggingEnabled:        true,
		Index:                 0,
	}
	api.existingFirewallPolicyID = existingFirewallPolicy.ID
	api.firewallPolicies[api.siteID][existingFirewallPolicy.ID] = existingFirewallPolicy
	api.firewallPolicyOrderings[api.siteID][firewallPolicyOrderingKey(existingZone.ID, existingZone.ID)] = client.FirewallPolicyOrdering{
		OrderedFirewallPolicyIDs: client.FirewallPolicyOrderedIDs{
			BeforeSystemDefined: []string{existingFirewallPolicy.ID},
		},
	}
	api.reindexFirewallPolicies(api.siteID, existingZone.ID, existingZone.ID)

	existingTrafficMatchingList := client.TrafficMatchingList{
		ID:   api.newID(),
		Type: "PORTS",
		Name: "existing-web-ports",
		Items: []client.TrafficMatchingItem{
			portNumberTrafficMatch(80),
			portRangeTrafficMatch(443, 444),
		},
	}
	api.existingTrafficMatchingListID = existingTrafficMatchingList.ID
	api.trafficMatchingLists[api.siteID][existingTrafficMatchingList.ID] = existingTrafficMatchingList

	existingDNSPolicy := client.DNSPolicy{
		ID:       api.newID(),
		Type:     "TXT_RECORD",
		Enabled:  true,
		Domain:   stringPtr("existing.example.internal"),
		Text:     stringPtr("existing"),
		Metadata: map[string]any{"origin": "user"},
	}
	api.existingDNSPolicyID = existingDNSPolicy.ID
	api.dnsPolicies[api.siteID][existingDNSPolicy.ID] = existingDNSPolicy

	existingDHCPReservation := client.DHCPReservation{
		ClientID:                  api.newID(),
		MACAddress:                "aa:bb:cc:dd:ee:10",
		Enabled:                   false,
		Hostname:                  stringPtr("printer"),
		Name:                      stringPtr("office-printer"),
		LastIP:                    stringPtr("10.170.0.40"),
		LastConnectionNetworkName: stringPtr("mgmt"),
	}
	api.existingDHCPReservationMAC = existingDHCPReservation.MACAddress
	api.dhcpReservations["default"][existingDHCPReservation.ClientID] = existingDHCPReservation

	existingACLRule := client.ACLRule{
		ID:      api.newID(),
		Type:    "IPV4",
		Enabled: true,
		Name:    "existing-acl",
		Action:  "BLOCK",
		EnforcingDeviceFilter: &client.ACLRuleDeviceFilter{
			Type:      "DEVICES",
			DeviceIDs: []string{api.newID()},
		},
		ProtocolFilter: []string{"TCP"},
		SourceFilter: &client.ACLRuleEndpointFilter{
			Type:                 "IP_ADDRESSES_OR_SUBNETS",
			IPAddressesOrSubnets: []string{"10.0.0.0/8"},
			PortFilter:           []int64{443},
		},
		Index: 0,
	}
	api.existingACLRuleID = existingACLRule.ID
	api.aclRules[api.siteID][existingACLRule.ID] = existingACLRule
	api.aclRuleOrderings[api.siteID] = client.ACLRuleOrdering{
		OrderedACLRuleIDs: []string{existingACLRule.ID},
	}
	api.reindexACLRules(api.siteID)

	existingRadiusProfile := client.RadiusProfile{
		ID:   api.newID(),
		Name: "existing-radius",
	}
	api.existingRadiusProfileID = existingRadiusProfile.ID
	api.radiusProfiles[api.siteID][existingRadiusProfile.ID] = existingRadiusProfile

	existingDeviceTag := client.DeviceTag{
		ID:        api.newID(),
		Name:      "existing-tag",
		DeviceIDs: []string{api.newID()},
	}
	api.existingDeviceTagID = existingDeviceTag.ID
	api.deviceTags[api.siteID][existingDeviceTag.ID] = existingDeviceTag

	existingWifiBroadcast := client.WifiBroadcast{
		ID:      api.newID(),
		Type:    "STANDARD",
		Name:    "existing-wifi",
		Enabled: true,
		Network: &client.WifiNetworkReference{
			Type:      "SPECIFIC",
			NetworkID: existingNetwork.ID,
		},
		SecurityConfiguration: &client.WifiSecurityConfiguration{
			Type:       "WPA2_PERSONAL",
			Passphrase: stringPtr("existingpass"),
		},
		ClientIsolationEnabled:              false,
		HideName:                            false,
		UAPSDEnabled:                        true,
		MulticastToUnicastConversionEnabled: false,
		BroadcastingFrequenciesGHz:          []float64{2.4, 5},
		AdvertiseDeviceName:                 boolPtr(false),
		ARPProxyEnabled:                     boolPtr(false),
		BandSteeringEnabled:                 boolPtr(true),
		BSSTransitionEnabled:                boolPtr(true),
		BroadcastingDeviceFilter: &client.WifiBroadcastingDeviceFilter{
			Type:         "DEVICE_TAGS",
			DeviceTagIDs: []string{existingDeviceTag.ID},
		},
	}
	api.existingWifiBroadcastID = existingWifiBroadcast.ID
	api.wifiBroadcasts[api.siteID][existingWifiBroadcast.ID] = existingWifiBroadcast

	existingVPNServer := client.VPNServer{
		ID:       api.newID(),
		Enabled:  true,
		Name:     "existing-vpn-server",
		Type:     "OPENVPN",
		Metadata: client.ReferenceMetadata{Origin: "user"},
	}
	api.existingVPNServerID = existingVPNServer.ID
	api.vpnServers[api.siteID][existingVPNServer.ID] = existingVPNServer

	existingSiteToSiteTunnel := client.SiteToSiteVPNTunnel{
		ID:       api.newID(),
		Name:     "existing-site-to-site",
		Type:     "SITE_TO_SITE",
		Metadata: client.ReferenceMetadata{Origin: "user"},
	}
	api.existingSiteToSiteTunnelID = existingSiteToSiteTunnel.ID
	api.siteToSiteVPNTunnels[api.siteID][existingSiteToSiteTunnel.ID] = existingSiteToSiteTunnel

	existingDPIApplication := client.DPIApplication{
		ID:   720973,
		Name: "Zoom",
	}
	api.existingDPIApplicationID = existingDPIApplication.ID
	api.dpiApplications[existingDPIApplication.ID] = existingDPIApplication

	existingDPICategory := client.DPIApplicationCategory{
		ID:   5,
		Name: "Business tools",
	}
	api.existingDPICategoryID = existingDPICategory.ID
	api.dpiApplicationCategories[existingDPICategory.ID] = existingDPICategory

	existingCountry := client.Country{
		Code: "US",
		Name: "United States",
	}
	api.existingCountryCode = existingCountry.Code
	api.countries[existingCountry.Code] = existingCountry

	existingSwitchDevice := client.Device{
		ID:                api.newID(),
		Name:              "core-switch-a",
		Model:             "USW-Pro-24",
		MacAddress:        "AA:BB:CC:DD:EE:01",
		IPAddress:         "10.0.0.10",
		State:             "ONLINE",
		Supported:         true,
		FirmwareUpdatable: false,
		FirmwareVersion:   stringPtr("7.1.26"),
		Features:          []string{"switching"},
		Interfaces:        []string{"ports"},
	}
	api.existingSwitchDeviceID = existingSwitchDevice.ID
	api.existingAdoptedDeviceMAC = existingSwitchDevice.MacAddress
	api.devices[api.siteID][existingSwitchDevice.ID] = existingSwitchDevice

	existingWAN := client.WAN{
		ID:   api.newID(),
		Name: "Internet 1",
	}
	api.existingWANID = existingWAN.ID
	api.wans[api.siteID][existingWAN.ID] = existingWAN

	switchMemberA := api.newID()
	switchMemberB := api.newID()
	existingSwitchStackLag := client.Lag{
		ID:   api.newID(),
		Type: "SWITCH_STACK",
		Members: []client.LagMember{
			{DeviceID: switchMemberA, PortIdxs: []int64{1, 2}},
			{DeviceID: switchMemberB, PortIdxs: []int64{1, 2}},
		},
	}
	existingSwitchStackID := api.newID()
	existingSwitchStackLag.SwitchStackID = stringPtr(existingSwitchStackID)
	existingSwitchStack := client.SwitchStack{
		ID:   existingSwitchStackID,
		Name: "core-stack",
		Members: []client.SwitchStackMember{
			{DeviceID: switchMemberA},
			{DeviceID: switchMemberB},
		},
		Lags: []client.SwitchStackLag{
			{ID: existingSwitchStackLag.ID, Members: existingSwitchStackLag.Members},
		},
	}
	api.existingSwitchStackID = existingSwitchStack.ID
	api.existingSwitchStackLagID = existingSwitchStackLag.ID
	api.switchStacks[api.siteID][existingSwitchStack.ID] = existingSwitchStack
	api.lags[api.siteID][existingSwitchStackLag.ID] = existingSwitchStackLag

	mcPeerTop := api.newID()
	mcPeerBottom := api.newID()
	existingMcLag := client.Lag{
		ID:   api.newID(),
		Type: "MULTI_CHASSIS",
		Members: []client.LagMember{
			{DeviceID: mcPeerTop, PortIdxs: []int64{5}},
			{DeviceID: mcPeerBottom, PortIdxs: []int64{5}},
		},
	}
	existingMcLagDomainID := api.newID()
	existingMcLag.McLagDomainID = stringPtr(existingMcLagDomainID)
	existingMcLagDomain := client.McLagDomain{
		ID:   existingMcLagDomainID,
		Name: "leaf-domain",
		Peers: []client.McLagPeer{
			{Role: "TOP", DeviceID: mcPeerTop, LinkPortIdxs: []int64{49, 50}},
			{Role: "BOTTOM", DeviceID: mcPeerBottom, LinkPortIdxs: []int64{49, 50}},
		},
		Lags: []client.McLagLocalLag{
			{ID: existingMcLag.ID, Members: existingMcLag.Members},
		},
	}
	api.existingMcLagDomainID = existingMcLagDomain.ID
	api.existingMcLagID = existingMcLag.ID
	api.mcLagDomains[api.siteID][existingMcLagDomain.ID] = existingMcLagDomain
	api.lags[api.siteID][existingMcLag.ID] = existingMcLag

	api.server = httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	return api
}

func (api *mockUniFiAPI) Close() {
	api.server.Close()
}

func (api *mockUniFiAPI) URL() string {
	return api.server.URL
}

func (api *mockUniFiAPI) newID() string {
	id := fmt.Sprintf("00000000-0000-0000-0000-%012d", api.nextID)
	api.nextID++
	return id
}

func firewallPolicyOrderingKey(sourceZoneID, destinationZoneID string) string {
	return sourceZoneID + "/" + destinationZoneID
}

func removeString(values []string, target string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			output = append(output, value)
		}
	}

	return output
}

func appendMissingIDs(existing []string, candidates []string) []string {
	allowed := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		allowed[candidate] = struct{}{}
	}

	seen := make(map[string]struct{}, len(existing))
	output := make([]string, 0, len(candidates))
	for _, value := range existing {
		if _, ok := allowed[value]; !ok {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		output = append(output, value)
		seen[value] = struct{}{}
	}

	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		output = append(output, candidate)
		seen[candidate] = struct{}{}
	}

	return output
}

func (api *mockUniFiAPI) orderedACLRuleIDs(siteID string) []string {
	var remaining []client.ACLRule
	for _, rule := range api.aclRules[siteID] {
		remaining = append(remaining, rule)
	}
	sort.Slice(remaining, func(i, j int) bool {
		if remaining[i].Index == remaining[j].Index {
			return remaining[i].ID < remaining[j].ID
		}
		return remaining[i].Index < remaining[j].Index
	})

	candidates := make([]string, 0, len(remaining))
	for _, rule := range remaining {
		candidates = append(candidates, rule.ID)
	}

	return appendMissingIDs(api.aclRuleOrderings[siteID].OrderedACLRuleIDs, candidates)
}

func (api *mockUniFiAPI) reindexACLRules(siteID string) {
	orderedIDs := api.orderedACLRuleIDs(siteID)
	api.aclRuleOrderings[siteID] = client.ACLRuleOrdering{
		OrderedACLRuleIDs: orderedIDs,
	}

	for index, ruleID := range orderedIDs {
		rule, ok := api.aclRules[siteID][ruleID]
		if !ok {
			continue
		}
		rule.Index = int64(index)
		api.aclRules[siteID][ruleID] = rule
	}
}

func (api *mockUniFiAPI) firewallPolicyPairIDs(siteID, sourceZoneID, destinationZoneID string) []string {
	var policies []client.FirewallPolicy
	for _, policy := range api.firewallPolicies[siteID] {
		if policy.Source == nil || policy.Destination == nil {
			continue
		}
		if policy.Source.ZoneID != sourceZoneID || policy.Destination.ZoneID != destinationZoneID {
			continue
		}
		policies = append(policies, policy)
	}
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].Index == policies[j].Index {
			return policies[i].ID < policies[j].ID
		}
		return policies[i].Index < policies[j].Index
	})

	ids := make([]string, 0, len(policies))
	for _, policy := range policies {
		ids = append(ids, policy.ID)
	}

	return ids
}

func (api *mockUniFiAPI) firewallPolicyOrdering(siteID, sourceZoneID, destinationZoneID string) client.FirewallPolicyOrdering {
	key := firewallPolicyOrderingKey(sourceZoneID, destinationZoneID)
	ordering := api.firewallPolicyOrderings[siteID][key]
	pairIDs := api.firewallPolicyPairIDs(siteID, sourceZoneID, destinationZoneID)
	if len(ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined) == 0 && len(ordering.OrderedFirewallPolicyIDs.AfterSystemDefined) == 0 {
		return client.FirewallPolicyOrdering{
			OrderedFirewallPolicyIDs: client.FirewallPolicyOrderedIDs{
				BeforeSystemDefined: pairIDs,
			},
		}
	}

	allowed := make(map[string]struct{}, len(pairIDs))
	for _, id := range pairIDs {
		allowed[id] = struct{}{}
	}

	var beforeIDs []string
	for _, id := range ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined {
		if _, ok := allowed[id]; ok {
			beforeIDs = append(beforeIDs, id)
		}
	}

	var afterIDs []string
	for _, id := range ordering.OrderedFirewallPolicyIDs.AfterSystemDefined {
		if _, ok := allowed[id]; ok {
			afterIDs = append(afterIDs, id)
		}
	}

	used := make(map[string]struct{}, len(beforeIDs)+len(afterIDs))
	for _, id := range beforeIDs {
		used[id] = struct{}{}
	}
	for _, id := range afterIDs {
		used[id] = struct{}{}
	}

	for _, id := range pairIDs {
		if _, ok := used[id]; ok {
			continue
		}
		beforeIDs = append(beforeIDs, id)
	}

	return client.FirewallPolicyOrdering{
		OrderedFirewallPolicyIDs: client.FirewallPolicyOrderedIDs{
			BeforeSystemDefined: beforeIDs,
			AfterSystemDefined:  afterIDs,
		},
	}
}

func (api *mockUniFiAPI) reindexFirewallPolicies(siteID, sourceZoneID, destinationZoneID string) {
	key := firewallPolicyOrderingKey(sourceZoneID, destinationZoneID)
	ordering := api.firewallPolicyOrdering(siteID, sourceZoneID, destinationZoneID)
	api.firewallPolicyOrderings[siteID][key] = ordering

	index := 0
	for _, policyID := range ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined {
		policy, ok := api.firewallPolicies[siteID][policyID]
		if !ok {
			continue
		}
		policy.Index = int64(index)
		api.firewallPolicies[siteID][policyID] = policy
		index++
	}
	for _, policyID := range ordering.OrderedFirewallPolicyIDs.AfterSystemDefined {
		policy, ok := api.firewallPolicies[siteID][policyID]
		if !ok {
			continue
		}
		policy.Index = int64(index)
		api.firewallPolicies[siteID][policyID] = policy
		index++
	}
}

func (api *mockUniFiAPI) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("X-API-KEY") != "test-key" {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	if strings.HasPrefix(request.URL.Path, "/proxy/network/api/") {
		api.handleLegacyHTTP(writer, request)
		return
	}

	if !strings.HasPrefix(request.URL.Path, "/integration") {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	trimmedPath := strings.TrimPrefix(request.URL.Path, "/integration")
	segments := strings.Split(strings.Trim(trimmedPath, "/"), "/")
	if len(segments) < 1 || segments[0] != "v1" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch {
	case len(segments) == 2 && segments[1] == "countries":
		api.handleCountries(writer, request)
		return
	case len(segments) == 3 && segments[1] == "dpi" && segments[2] == "applications":
		api.handleDPIApplications(writer, request)
		return
	case len(segments) == 3 && segments[1] == "dpi" && segments[2] == "categories":
		api.handleDPIApplicationCategories(writer, request)
		return
	}

	if len(segments) < 2 || segments[1] != "sites" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch {
	case len(segments) == 2 && request.Method == http.MethodGet:
		writePage(writer, request, orderedSiteSlice(api.sites))
		return
	case len(segments) == 4 && segments[3] == "networks":
		api.handleNetworks(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "networks":
		api.handleNetwork(writer, request, segments[2], segments[4])
		return
	case len(segments) == 5 && segments[3] == "wifi" && segments[4] == "broadcasts":
		api.handleWifiBroadcasts(writer, request, segments[2])
		return
	case len(segments) == 6 && segments[3] == "wifi" && segments[4] == "broadcasts":
		api.handleWifiBroadcast(writer, request, segments[2], segments[5])
		return
	case len(segments) == 5 && segments[3] == "firewall" && segments[4] == "zones":
		api.handleFirewallZones(writer, request, segments[2])
		return
	case len(segments) == 6 && segments[3] == "firewall" && segments[4] == "zones":
		api.handleFirewallZone(writer, request, segments[2], segments[5])
		return
	case len(segments) == 5 && segments[3] == "firewall" && segments[4] == "policies":
		api.handleFirewallPolicies(writer, request, segments[2])
		return
	case len(segments) == 6 && segments[3] == "firewall" && segments[4] == "policies" && segments[5] == "ordering":
		api.handleFirewallPolicyOrdering(writer, request, segments[2])
		return
	case len(segments) == 6 && segments[3] == "firewall" && segments[4] == "policies":
		api.handleFirewallPolicy(writer, request, segments[2], segments[5])
		return
	case len(segments) == 4 && segments[3] == "traffic-matching-lists":
		api.handleTrafficMatchingLists(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "traffic-matching-lists":
		api.handleTrafficMatchingList(writer, request, segments[2], segments[4])
		return
	case len(segments) == 5 && segments[3] == "radius" && segments[4] == "profiles":
		api.handleRadiusProfiles(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "vpn" && segments[4] == "servers":
		api.handleVPNServers(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "vpn" && segments[4] == "site-to-site-tunnels":
		api.handleSiteToSiteVPNTunnels(writer, request, segments[2])
		return
	case len(segments) == 4 && segments[3] == "device-tags":
		api.handleDeviceTags(writer, request, segments[2])
		return
	case len(segments) == 4 && segments[3] == "devices":
		api.handleDevices(writer, request, segments[2])
		return
	case len(segments) == 4 && segments[3] == "wans":
		api.handleWANs(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "switching" && segments[4] == "switch-stacks":
		api.handleSwitchStacks(writer, request, segments[2])
		return
	case len(segments) == 6 && segments[3] == "switching" && segments[4] == "lags":
		api.handleLag(writer, request, segments[2], segments[5])
		return
	case len(segments) == 5 && segments[3] == "switching" && segments[4] == "lags":
		api.handleLags(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "switching" && segments[4] == "mc-lag-domains":
		api.handleMcLagDomains(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "dns" && segments[4] == "policies":
		api.handleDNSPolicies(writer, request, segments[2])
		return
	case len(segments) == 6 && segments[3] == "dns" && segments[4] == "policies":
		api.handleDNSPolicy(writer, request, segments[2], segments[5])
		return
	case len(segments) == 4 && segments[3] == "acl-rules":
		api.handleACLRules(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "acl-rules" && segments[4] == "ordering":
		api.handleACLRuleOrdering(writer, request, segments[2])
		return
	case len(segments) == 5 && segments[3] == "acl-rules":
		api.handleACLRule(writer, request, segments[2], segments[4])
		return
	default:
		writer.WriteHeader(http.StatusNotFound)
	}
}

func (api *mockUniFiAPI) handleLegacyHTTP(writer http.ResponseWriter, request *http.Request) {
	segments := strings.Split(strings.Trim(strings.TrimPrefix(request.URL.Path, "/proxy/network/api"), "/"), "/")
	if len(segments) != 4 || segments[0] != "s" || segments[2] != "rest" || segments[3] != "user" {
		if len(segments) == 5 && segments[0] == "s" && segments[2] == "rest" && segments[3] == "user" {
			api.handleDHCPReservation(writer, request, segments[1], segments[4])
			return
		}
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	api.handleDHCPReservations(writer, request, segments[1])
}

func (api *mockUniFiAPI) handleNetworks(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var networks []client.Network
		for _, network := range api.networks[siteID] {
			networks = append(networks, network)
		}
		sort.Slice(networks, func(i, j int) bool {
			return networks[i].ID < networks[j].ID
		})
		writePage(writer, request, networks)
	case http.MethodPost:
		var network client.Network
		api.decodeRequest(writer, request, &network)
		network.ID = api.newID()
		if network.Default == false {
			network.Default = false
		}
		api.networks[siteID][network.ID] = network
		api.writeJSON(writer, http.StatusCreated, network)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDHCPReservations(writer http.ResponseWriter, request *http.Request, siteReference string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if _, ok := api.siteIDByInternalReference(siteReference); !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		var reservations []client.DHCPReservation
		for _, reservation := range api.dhcpReservations[siteReference] {
			reservations = append(reservations, reservation)
		}
		sort.Slice(reservations, func(i, j int) bool {
			return reservations[i].ClientID < reservations[j].ClientID
		})
		api.writeLegacyJSON(writer, http.StatusOK, reservations)
	case http.MethodPost:
		var createRequest struct {
			MACAddress string  `json:"mac"`
			Name       *string `json:"name"`
		}
		api.decodeRequest(writer, request, &createRequest)
		clientID := api.newID()
		reservation := client.DHCPReservation{
			ClientID:   clientID,
			MACAddress: createRequest.MACAddress,
			Name:       createRequest.Name,
			Enabled:    false,
		}
		api.dhcpReservations[siteReference][clientID] = reservation
		api.writeLegacyJSON(writer, http.StatusCreated, []client.DHCPReservation{reservation})
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDHCPReservation(writer http.ResponseWriter, request *http.Request, siteReference, clientID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if _, ok := api.siteIDByInternalReference(siteReference); !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	reservation, ok := api.dhcpReservations[siteReference][clientID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodPut:
		var update client.DHCPReservation
		api.decodeRequest(writer, request, &update)
		if update.NetworkID != nil {
			reservation.NetworkID = update.NetworkID
		}
		if update.FixedIP != nil {
			reservation.FixedIP = update.FixedIP
		}
		reservation.Enabled = update.Enabled
		api.dhcpReservations[siteReference][clientID] = reservation
		api.writeLegacyJSON(writer, http.StatusOK, []client.DHCPReservation{})
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) siteIDByInternalReference(siteReference string) (string, bool) {
	for siteID, site := range api.sites {
		if site.InternalReference == siteReference {
			return siteID, true
		}
	}

	return "", false
}

func (api *mockUniFiAPI) handleNetwork(writer http.ResponseWriter, request *http.Request, siteID, networkID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	network, ok := api.networks[siteID][networkID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, network)
	case http.MethodPut:
		var updated client.Network
		api.decodeRequest(writer, request, &updated)
		updated.ID = networkID
		updated.Default = network.Default
		api.networks[siteID][networkID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.networks[siteID], networkID)
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleWifiBroadcasts(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var broadcasts []client.WifiBroadcast
		for _, broadcast := range api.wifiBroadcasts[siteID] {
			broadcasts = append(broadcasts, broadcast)
		}
		sort.Slice(broadcasts, func(i, j int) bool {
			return broadcasts[i].ID < broadcasts[j].ID
		})
		writePage(writer, request, broadcasts)
	case http.MethodPost:
		var broadcast client.WifiBroadcast
		api.decodeRequest(writer, request, &broadcast)
		broadcast.ID = api.newID()
		api.wifiBroadcasts[siteID][broadcast.ID] = broadcast
		api.writeJSON(writer, http.StatusCreated, broadcast)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleWifiBroadcast(writer http.ResponseWriter, request *http.Request, siteID, broadcastID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	broadcast, ok := api.wifiBroadcasts[siteID][broadcastID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, broadcast)
	case http.MethodPut:
		var updated client.WifiBroadcast
		api.decodeRequest(writer, request, &updated)
		updated.ID = broadcastID
		api.wifiBroadcasts[siteID][broadcastID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.wifiBroadcasts[siteID], broadcastID)
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleFirewallZones(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var zones []client.FirewallZone
		for _, zone := range api.firewallZones[siteID] {
			zones = append(zones, zone)
		}
		sort.Slice(zones, func(i, j int) bool {
			return zones[i].ID < zones[j].ID
		})
		writePage(writer, request, zones)
	case http.MethodPost:
		var zone client.FirewallZone
		api.decodeRequest(writer, request, &zone)
		zone.ID = api.newID()
		api.firewallZones[siteID][zone.ID] = zone
		api.writeJSON(writer, http.StatusCreated, zone)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleFirewallZone(writer http.ResponseWriter, request *http.Request, siteID, zoneID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	zone, ok := api.firewallZones[siteID][zoneID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, zone)
	case http.MethodPut:
		var updated client.FirewallZone
		api.decodeRequest(writer, request, &updated)
		updated.ID = zoneID
		api.firewallZones[siteID][zoneID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.firewallZones[siteID], zoneID)
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleFirewallPolicies(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var policies []client.FirewallPolicy
		for _, policy := range api.firewallPolicies[siteID] {
			policies = append(policies, policy)
		}
		sort.Slice(policies, func(i, j int) bool {
			if policies[i].Index == policies[j].Index {
				return policies[i].ID < policies[j].ID
			}
			return policies[i].Index < policies[j].Index
		})
		writePage(writer, request, policies)
	case http.MethodPost:
		var policy client.FirewallPolicy
		api.decodeRequest(writer, request, &policy)
		policy.ID = api.newID()
		api.firewallPolicies[siteID][policy.ID] = policy
		if policy.Source != nil && policy.Destination != nil {
			key := firewallPolicyOrderingKey(policy.Source.ZoneID, policy.Destination.ZoneID)
			ordering := api.firewallPolicyOrderings[siteID][key]
			ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined = append(ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined, policy.ID)
			api.firewallPolicyOrderings[siteID][key] = ordering
			api.reindexFirewallPolicies(siteID, policy.Source.ZoneID, policy.Destination.ZoneID)
		}
		api.writeJSON(writer, http.StatusCreated, policy)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleFirewallPolicy(writer http.ResponseWriter, request *http.Request, siteID, policyID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	policy, ok := api.firewallPolicies[siteID][policyID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, policy)
	case http.MethodPut:
		var updated client.FirewallPolicy
		api.decodeRequest(writer, request, &updated)
		oldSourceZoneID := ""
		oldDestinationZoneID := ""
		if policy.Source != nil {
			oldSourceZoneID = policy.Source.ZoneID
		}
		if policy.Destination != nil {
			oldDestinationZoneID = policy.Destination.ZoneID
		}
		newSourceZoneID := ""
		newDestinationZoneID := ""
		if updated.Source != nil {
			newSourceZoneID = updated.Source.ZoneID
		}
		if updated.Destination != nil {
			newDestinationZoneID = updated.Destination.ZoneID
		}
		updated.ID = policyID
		api.firewallPolicies[siteID][policyID] = updated
		if oldSourceZoneID != "" && oldDestinationZoneID != "" && (oldSourceZoneID != newSourceZoneID || oldDestinationZoneID != newDestinationZoneID) {
			key := firewallPolicyOrderingKey(oldSourceZoneID, oldDestinationZoneID)
			ordering := api.firewallPolicyOrderings[siteID][key]
			ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined = removeString(ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined, policyID)
			ordering.OrderedFirewallPolicyIDs.AfterSystemDefined = removeString(ordering.OrderedFirewallPolicyIDs.AfterSystemDefined, policyID)
			api.firewallPolicyOrderings[siteID][key] = ordering
			api.reindexFirewallPolicies(siteID, oldSourceZoneID, oldDestinationZoneID)
		}
		if newSourceZoneID != "" && newDestinationZoneID != "" {
			key := firewallPolicyOrderingKey(newSourceZoneID, newDestinationZoneID)
			ordering := api.firewallPolicyOrderings[siteID][key]
			if oldSourceZoneID != newSourceZoneID || oldDestinationZoneID != newDestinationZoneID {
				ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined = append(ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined, policyID)
			}
			api.firewallPolicyOrderings[siteID][key] = ordering
			api.reindexFirewallPolicies(siteID, newSourceZoneID, newDestinationZoneID)
			updated = api.firewallPolicies[siteID][policyID]
		}
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.firewallPolicies[siteID], policyID)
		if policy.Source != nil && policy.Destination != nil {
			key := firewallPolicyOrderingKey(policy.Source.ZoneID, policy.Destination.ZoneID)
			ordering := api.firewallPolicyOrderings[siteID][key]
			ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined = removeString(ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined, policyID)
			ordering.OrderedFirewallPolicyIDs.AfterSystemDefined = removeString(ordering.OrderedFirewallPolicyIDs.AfterSystemDefined, policyID)
			api.firewallPolicyOrderings[siteID][key] = ordering
			api.reindexFirewallPolicies(siteID, policy.Source.ZoneID, policy.Destination.ZoneID)
		}
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleFirewallPolicyOrdering(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	sourceZoneID := request.URL.Query().Get("sourceFirewallZoneId")
	destinationZoneID := request.URL.Query().Get("destinationFirewallZoneId")
	if sourceZoneID == "" || destinationZoneID == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, api.firewallPolicyOrdering(siteID, sourceZoneID, destinationZoneID))
	case http.MethodPut:
		var ordering client.FirewallPolicyOrdering
		api.decodeRequest(writer, request, &ordering)
		key := firewallPolicyOrderingKey(sourceZoneID, destinationZoneID)
		api.firewallPolicyOrderings[siteID][key] = ordering
		api.reindexFirewallPolicies(siteID, sourceZoneID, destinationZoneID)
		api.writeJSON(writer, http.StatusOK, api.firewallPolicyOrderings[siteID][key])
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleTrafficMatchingLists(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var lists []client.TrafficMatchingList
		for _, list := range api.trafficMatchingLists[siteID] {
			lists = append(lists, list)
		}
		sort.Slice(lists, func(i, j int) bool {
			return lists[i].ID < lists[j].ID
		})
		writePage(writer, request, lists)
	case http.MethodPost:
		var list client.TrafficMatchingList
		api.decodeRequest(writer, request, &list)
		list.ID = api.newID()
		api.trafficMatchingLists[siteID][list.ID] = list
		api.writeJSON(writer, http.StatusCreated, list)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleTrafficMatchingList(writer http.ResponseWriter, request *http.Request, siteID, listID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	list, ok := api.trafficMatchingLists[siteID][listID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, list)
	case http.MethodPut:
		var updated client.TrafficMatchingList
		api.decodeRequest(writer, request, &updated)
		updated.ID = listID
		api.trafficMatchingLists[siteID][listID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.trafficMatchingLists[siteID], listID)
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleRadiusProfiles(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var profiles []client.RadiusProfile
		for _, profile := range api.radiusProfiles[siteID] {
			profiles = append(profiles, profile)
		}
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].ID < profiles[j].ID
		})
		writePage(writer, request, profiles)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleVPNServers(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var servers []client.VPNServer
		for _, server := range api.vpnServers[siteID] {
			servers = append(servers, server)
		}
		sort.Slice(servers, func(i, j int) bool {
			return servers[i].ID < servers[j].ID
		})
		writePage(writer, request, servers)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleSiteToSiteVPNTunnels(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var tunnels []client.SiteToSiteVPNTunnel
		for _, tunnel := range api.siteToSiteVPNTunnels[siteID] {
			tunnels = append(tunnels, tunnel)
		}
		sort.Slice(tunnels, func(i, j int) bool {
			return tunnels[i].ID < tunnels[j].ID
		})
		writePage(writer, request, tunnels)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDeviceTags(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var tags []client.DeviceTag
		for _, tag := range api.deviceTags[siteID] {
			tags = append(tags, tag)
		}
		sort.Slice(tags, func(i, j int) bool {
			return tags[i].ID < tags[j].ID
		})
		writePage(writer, request, tags)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleCountries(writer http.ResponseWriter, request *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var countries []client.Country
		for _, country := range api.countries {
			countries = append(countries, country)
		}
		sort.Slice(countries, func(i, j int) bool {
			return countries[i].Code < countries[j].Code
		})
		writePage(writer, request, countries)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDPIApplications(writer http.ResponseWriter, request *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var applications []client.DPIApplication
		for _, application := range api.dpiApplications {
			applications = append(applications, application)
		}
		sort.Slice(applications, func(i, j int) bool {
			return applications[i].ID < applications[j].ID
		})
		writePage(writer, request, applications)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDPIApplicationCategories(writer http.ResponseWriter, request *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var categories []client.DPIApplicationCategory
		for _, category := range api.dpiApplicationCategories {
			categories = append(categories, category)
		}
		sort.Slice(categories, func(i, j int) bool {
			return categories[i].ID < categories[j].ID
		})
		writePage(writer, request, categories)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDevices(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var devices []client.Device
		for _, device := range api.devices[siteID] {
			devices = append(devices, device)
		}
		sort.Slice(devices, func(i, j int) bool {
			return devices[i].ID < devices[j].ID
		})
		writePage(writer, request, devices)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleWANs(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var wans []client.WAN
		for _, wan := range api.wans[siteID] {
			wans = append(wans, wan)
		}
		sort.Slice(wans, func(i, j int) bool {
			return wans[i].ID < wans[j].ID
		})
		writePage(writer, request, wans)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleSwitchStacks(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var stacks []client.SwitchStack
		for _, stack := range api.switchStacks[siteID] {
			stacks = append(stacks, stack)
		}
		sort.Slice(stacks, func(i, j int) bool {
			return stacks[i].ID < stacks[j].ID
		})
		writePage(writer, request, stacks)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleLags(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var lags []client.Lag
		for _, lag := range api.lags[siteID] {
			lags = append(lags, lag)
		}
		sort.Slice(lags, func(i, j int) bool {
			return lags[i].ID < lags[j].ID
		})
		writePage(writer, request, lags)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleLag(writer http.ResponseWriter, request *http.Request, siteID, lagID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	lag, ok := api.lags[siteID][lagID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, lag)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleMcLagDomains(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var domains []client.McLagDomain
		for _, domain := range api.mcLagDomains[siteID] {
			domains = append(domains, domain)
		}
		sort.Slice(domains, func(i, j int) bool {
			return domains[i].ID < domains[j].ID
		})
		writePage(writer, request, domains)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDNSPolicies(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var policies []client.DNSPolicy
		for _, policy := range api.dnsPolicies[siteID] {
			policies = append(policies, policy)
		}
		sort.Slice(policies, func(i, j int) bool {
			return policies[i].ID < policies[j].ID
		})
		writePage(writer, request, policies)
	case http.MethodPost:
		var policy client.DNSPolicy
		api.decodeRequest(writer, request, &policy)
		policy.ID = api.newID()
		api.dnsPolicies[siteID][policy.ID] = policy
		api.writeJSON(writer, http.StatusCreated, policy)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleDNSPolicy(writer http.ResponseWriter, request *http.Request, siteID, policyID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	policy, ok := api.dnsPolicies[siteID][policyID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, policy)
	case http.MethodPut:
		var updated client.DNSPolicy
		api.decodeRequest(writer, request, &updated)
		updated.ID = policyID
		api.dnsPolicies[siteID][policyID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.dnsPolicies[siteID], policyID)
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleACLRules(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		var rules []client.ACLRule
		for _, rule := range api.aclRules[siteID] {
			rules = append(rules, rule)
		}
		sort.Slice(rules, func(i, j int) bool {
			if rules[i].Index == rules[j].Index {
				return rules[i].ID < rules[j].ID
			}
			return rules[i].Index < rules[j].Index
		})
		writePage(writer, request, rules)
	case http.MethodPost:
		var rule client.ACLRule
		api.decodeRequest(writer, request, &rule)
		rule.ID = api.newID()
		api.aclRules[siteID][rule.ID] = rule
		ordering := api.aclRuleOrderings[siteID]
		ordering.OrderedACLRuleIDs = append(ordering.OrderedACLRuleIDs, rule.ID)
		api.aclRuleOrderings[siteID] = ordering
		api.reindexACLRules(siteID)
		rule = api.aclRules[siteID][rule.ID]
		api.writeJSON(writer, http.StatusCreated, rule)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleACLRule(writer http.ResponseWriter, request *http.Request, siteID, aclRuleID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	rule, ok := api.aclRules[siteID][aclRuleID]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, rule)
	case http.MethodPut:
		var updated client.ACLRule
		api.decodeRequest(writer, request, &updated)
		updated.ID = aclRuleID
		api.aclRules[siteID][aclRuleID] = updated
		api.reindexACLRules(siteID)
		updated = api.aclRules[siteID][aclRuleID]
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.aclRules[siteID], aclRuleID)
		ordering := api.aclRuleOrderings[siteID]
		ordering.OrderedACLRuleIDs = removeString(ordering.OrderedACLRuleIDs, aclRuleID)
		api.aclRuleOrderings[siteID] = ordering
		api.reindexACLRules(siteID)
		writer.WriteHeader(http.StatusOK)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) handleACLRuleOrdering(writer http.ResponseWriter, request *http.Request, siteID string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	switch request.Method {
	case http.MethodGet:
		api.writeJSON(writer, http.StatusOK, client.ACLRuleOrdering{
			OrderedACLRuleIDs: api.orderedACLRuleIDs(siteID),
		})
	case http.MethodPut:
		var ordering client.ACLRuleOrdering
		api.decodeRequest(writer, request, &ordering)
		api.aclRuleOrderings[siteID] = ordering
		api.reindexACLRules(siteID)
		api.writeJSON(writer, http.StatusOK, api.aclRuleOrderings[siteID])
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *mockUniFiAPI) decodeRequest(writer http.ResponseWriter, request *http.Request, target any) {
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
	}
}

func (api *mockUniFiAPI) writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func (api *mockUniFiAPI) writeLegacyJSON(writer http.ResponseWriter, statusCode int, payload any) {
	api.writeJSON(writer, statusCode, map[string]any{
		"meta": map[string]string{"rc": "ok"},
		"data": payload,
	})
}

func writePage[T any](writer http.ResponseWriter, request *http.Request, data []T) {
	offset, _ := strconv.Atoi(request.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = len(data)
	}
	if offset > len(data) {
		offset = len(data)
	}
	end := offset + limit
	if end > len(data) {
		end = len(data)
	}

	payload := map[string]any{
		"offset":     offset,
		"limit":      limit,
		"count":      end - offset,
		"totalCount": len(data),
		"data":       data[offset:end],
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(payload)
}

func orderedSiteSlice(input map[string]client.Site) []client.Site {
	output := make([]client.Site, 0, len(input))
	for _, site := range input {
		output = append(output, site)
	}
	sort.Slice(output, func(i, j int) bool {
		return output[i].ID < output[j].ID
	})
	return output
}

func providerConfig(baseURL string) string {
	return fmt.Sprintf(`
provider "unifi" {
  api_url        = %q
  api_key        = "test-key"
  allow_insecure = true
}
`, baseURL)
}

func siteLookupConfig(baseURL string) string {
	return providerConfig(baseURL) + `
data "unifi_site" "main" {
  name = "Default"
}
`
}

func testImportCompositeID(resourceName string, siteID string) resource.ImportStateIdFunc {
	return func(state *tfstate.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		return fmt.Sprintf("%s/%s", siteID, resourceState.Primary.ID), nil
	}
}

func testImportResourceID(resourceName string) resource.ImportStateIdFunc {
	return func(state *tfstate.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		return resourceState.Primary.ID, nil
	}
}

func testACLRuleOrderingImportID(siteID string) resource.ImportStateIdFunc {
	return func(_ *tfstate.State) (string, error) {
		return siteID, nil
	}
}

func testFirewallPolicyOrderingImportID(resourceName string, siteID string) resource.ImportStateIdFunc {
	return func(state *tfstate.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		sourceZoneID := resourceState.Primary.Attributes["source_zone_id"]
		destinationZoneID := resourceState.Primary.Attributes["destination_zone_id"]
		if sourceZoneID == "" || destinationZoneID == "" {
			return "", fmt.Errorf("resource %s is missing zone IDs", resourceName)
		}

		return fmt.Sprintf("%s/%s/%s", siteID, sourceZoneID, destinationZoneID), nil
	}
}

func TestAccDataSources(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(api.URL()) + `
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_device" "existing_switch" {
  site_id          = data.unifi_site.main.id
  name             = "core-switch-a"
  required_feature = "switching"
}

data "unifi_network" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-network"
}

data "unifi_wifi_broadcast" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-wifi"
}

data "unifi_firewall_zone" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-zone"
}

data "unifi_firewall_policy" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-policy"
}

data "unifi_traffic_matching_list" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-web-ports"
}

data "unifi_dns_policy" "existing" {
  site_id = data.unifi_site.main.id
  domain  = "existing.example.internal"
  type    = "TXT_RECORD"
}

data "unifi_acl_rule" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-acl"
}

data "unifi_radius_profile" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-radius"
}

data "unifi_device_tag" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-tag"
}

data "unifi_vpn_server" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-vpn-server"
}

data "unifi_site_to_site_vpn_tunnel" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-site-to-site"
}

data "unifi_dpi_application" "existing" {
  name = "Zoom"
}

data "unifi_dpi_application_category" "existing" {
  id = 5
}

data "unifi_country" "existing" {
  code = "US"
}

data "unifi_wan" "existing" {
  site_id = data.unifi_site.main.id
  name    = "Internet 1"
}

data "unifi_switch_stack" "existing" {
  site_id = data.unifi_site.main.id
  name    = "core-stack"
}

data "unifi_mc_lag_domain" "existing" {
  site_id = data.unifi_site.main.id
  name    = "leaf-domain"
}

data "unifi_lag" "existing" {
  site_id = data.unifi_site.main.id
  id      = "` + api.existingSwitchStackLagID + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_site.main", "id", api.siteID),
					resource.TestCheckResourceAttr("data.unifi_site.main", "internal_reference", "default"),
					resource.TestCheckResourceAttr("data.unifi_device.existing_switch", "id", api.existingSwitchDeviceID),
					resource.TestCheckResourceAttr("data.unifi_device.existing_switch", "name", "core-switch-a"),
					resource.TestCheckResourceAttr("data.unifi_device.existing_switch", "model", "USW-Pro-24"),
					resource.TestCheckResourceAttr("data.unifi_device.existing_switch", "mac_address", "AA:BB:CC:DD:EE:01"),
					resource.TestCheckResourceAttr("data.unifi_device.existing_switch", "ip_address", "10.0.0.10"),
					resource.TestCheckResourceAttr("data.unifi_device.existing_switch", "state", "ONLINE"),
					resource.TestCheckTypeSetElemAttr("data.unifi_device.existing_switch", "features.*", "switching"),
					resource.TestCheckTypeSetElemAttr("data.unifi_device.existing_switch", "interfaces.*", "ports"),
					resource.TestCheckResourceAttr("data.unifi_network.existing", "id", api.existingNetworkID),
					resource.TestCheckResourceAttr("data.unifi_network.existing", "management", "GATEWAY"),
					resource.TestCheckResourceAttr("data.unifi_network.existing", "vlan_id", "200"),
					resource.TestCheckResourceAttr("data.unifi_wifi_broadcast.existing", "id", api.existingWifiBroadcastID),
					resource.TestCheckResourceAttr("data.unifi_wifi_broadcast.existing", "name", "existing-wifi"),
					resource.TestCheckResourceAttr("data.unifi_wifi_broadcast.existing", "type", "STANDARD"),
					resource.TestCheckResourceAttr("data.unifi_wifi_broadcast.existing", "network.type", "SPECIFIC"),
					resource.TestCheckResourceAttr("data.unifi_wifi_broadcast.existing", "network.network_id", api.existingNetworkID),
					resource.TestCheckResourceAttr("data.unifi_wifi_broadcast.existing", "broadcasting_device_filter.type", "DEVICE_TAGS"),
					resource.TestCheckTypeSetElemAttr("data.unifi_wifi_broadcast.existing", "broadcasting_device_filter.device_tag_ids.*", api.existingDeviceTagID),
					resource.TestCheckResourceAttr("data.unifi_firewall_zone.existing", "id", api.existingZoneID),
					resource.TestCheckResourceAttr("data.unifi_firewall_zone.existing", "name", "existing-zone"),
					resource.TestCheckTypeSetElemAttr("data.unifi_firewall_zone.existing", "network_ids.*", api.existingNetworkID),
					resource.TestCheckResourceAttr("data.unifi_firewall_policy.existing", "id", api.existingFirewallPolicyID),
					resource.TestCheckResourceAttr("data.unifi_firewall_policy.existing", "action", "ALLOW"),
					resource.TestCheckResourceAttr("data.unifi_firewall_policy.existing", "source_filter.type", "NETWORK"),
					resource.TestCheckResourceAttr("data.unifi_firewall_policy.existing", "destination_filter.type", "PORT"),
					resource.TestCheckResourceAttr("data.unifi_firewall_policy.existing", "protocol_filter.type", "NAMED_PROTOCOL"),
					resource.TestCheckResourceAttr("data.unifi_traffic_matching_list.existing", "id", api.existingTrafficMatchingListID),
					resource.TestCheckResourceAttr("data.unifi_traffic_matching_list.existing", "type", "PORTS"),
					resource.TestCheckTypeSetElemAttr("data.unifi_traffic_matching_list.existing", "ports.*", "80"),
					resource.TestCheckTypeSetElemAttr("data.unifi_traffic_matching_list.existing", "ports.*", "443-444"),
					resource.TestCheckResourceAttr("data.unifi_dns_policy.existing", "id", api.existingDNSPolicyID),
					resource.TestCheckResourceAttr("data.unifi_dns_policy.existing", "type", "TXT_RECORD"),
					resource.TestCheckResourceAttr("data.unifi_dns_policy.existing", "text", "existing"),
					resource.TestCheckResourceAttr("data.unifi_acl_rule.existing", "id", api.existingACLRuleID),
					resource.TestCheckResourceAttr("data.unifi_acl_rule.existing", "type", "IPV4"),
					resource.TestCheckTypeSetElemAttr("data.unifi_acl_rule.existing", "protocol_filter.*", "TCP"),
					resource.TestCheckResourceAttr("data.unifi_radius_profile.existing", "id", api.existingRadiusProfileID),
					resource.TestCheckResourceAttr("data.unifi_radius_profile.existing", "name", "existing-radius"),
					resource.TestCheckResourceAttr("data.unifi_device_tag.existing", "id", api.existingDeviceTagID),
					resource.TestCheckResourceAttr("data.unifi_device_tag.existing", "name", "existing-tag"),
					resource.TestCheckResourceAttr("data.unifi_device_tag.existing", "device_ids.#", "1"),
					resource.TestCheckResourceAttr("data.unifi_vpn_server.existing", "id", api.existingVPNServerID),
					resource.TestCheckResourceAttr("data.unifi_vpn_server.existing", "type", "OPENVPN"),
					resource.TestCheckResourceAttr("data.unifi_vpn_server.existing", "enabled", "true"),
					resource.TestCheckResourceAttr("data.unifi_vpn_server.existing", "origin", "user"),
					resource.TestCheckResourceAttr("data.unifi_site_to_site_vpn_tunnel.existing", "id", api.existingSiteToSiteTunnelID),
					resource.TestCheckResourceAttr("data.unifi_site_to_site_vpn_tunnel.existing", "type", "SITE_TO_SITE"),
					resource.TestCheckResourceAttr("data.unifi_site_to_site_vpn_tunnel.existing", "origin", "user"),
					resource.TestCheckResourceAttr("data.unifi_dpi_application.existing", "id", fmt.Sprintf("%d", api.existingDPIApplicationID)),
					resource.TestCheckResourceAttr("data.unifi_dpi_application.existing", "name", "Zoom"),
					resource.TestCheckResourceAttr("data.unifi_dpi_application_category.existing", "id", fmt.Sprintf("%d", api.existingDPICategoryID)),
					resource.TestCheckResourceAttr("data.unifi_dpi_application_category.existing", "name", "Business tools"),
					resource.TestCheckResourceAttr("data.unifi_country.existing", "code", api.existingCountryCode),
					resource.TestCheckResourceAttr("data.unifi_country.existing", "name", "United States"),
					resource.TestCheckResourceAttr("data.unifi_wan.existing", "id", api.existingWANID),
					resource.TestCheckResourceAttr("data.unifi_wan.existing", "name", "Internet 1"),
					resource.TestCheckResourceAttr("data.unifi_switch_stack.existing", "id", api.existingSwitchStackID),
					resource.TestCheckResourceAttr("data.unifi_switch_stack.existing", "name", "core-stack"),
					resource.TestCheckResourceAttr("data.unifi_switch_stack.existing", "member_device_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.unifi_switch_stack.existing", "lag_ids.*", api.existingSwitchStackLagID),
					resource.TestCheckResourceAttr("data.unifi_mc_lag_domain.existing", "id", api.existingMcLagDomainID),
					resource.TestCheckResourceAttr("data.unifi_mc_lag_domain.existing", "name", "leaf-domain"),
					resource.TestCheckResourceAttr("data.unifi_mc_lag_domain.existing", "peer_device_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.unifi_mc_lag_domain.existing", "lag_ids.*", api.existingMcLagID),
					resource.TestCheckResourceAttr("data.unifi_lag.existing", "id", api.existingSwitchStackLagID),
					resource.TestCheckResourceAttr("data.unifi_lag.existing", "type", "SWITCH_STACK"),
					resource.TestCheckResourceAttr("data.unifi_lag.existing", "member_device_ids.#", "2"),
					resource.TestCheckResourceAttr("data.unifi_lag.existing", "switch_stack_id", api.existingSwitchStackID),
				),
			},
		},
	})
}

func TestAccResourceNetwork(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_network.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "test" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "trusted"
  enabled                 = true
  vlan_id                 = 20
  isolation_enabled       = false
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = true

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.20.0.1"
    prefix_length      = 24

    dhcp_configuration = {
      mode                            = "SERVER"
      start_ip_address                = "10.20.0.100"
      end_ip_address                  = "10.20.0.200"
      lease_time_seconds              = 86400
      ping_conflict_detection_enabled = false
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted"),
					resource.TestCheckResourceAttr(resourceName, "management", "GATEWAY"),
					resource.TestCheckResourceAttr(resourceName, "vlan_id", "20"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "test" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "trusted-updated"
  enabled                 = true
  vlan_id                 = 21
  isolation_enabled       = false
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = false

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.21.0.1"
    prefix_length      = 24

    dhcp_configuration = {
      mode                            = "SERVER"
      start_ip_address                = "10.21.0.100"
      end_ip_address                  = "10.21.0.200"
      lease_time_seconds              = 3600
      ping_conflict_detection_enabled = true
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-updated"),
					resource.TestCheckResourceAttr(resourceName, "vlan_id", "21"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"default"},
			},
		},
	})
}

func TestAccResourceFirewallZone(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_firewall_zone.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "test" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "zone-network"
  enabled                 = true
  vlan_id                 = 50
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.main.id
  name        = "custom-zone"
  network_ids = [unifi_network.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "custom-zone"),
					resource.TestCheckResourceAttr(resourceName, "network_ids.#", "1"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "test" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "zone-network"
  enabled                 = true
  vlan_id                 = 50
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.main.id
  name        = "custom-zone-updated"
  network_ids = [unifi_network.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "custom-zone-updated"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceTrafficMatchingList(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_traffic_matching_list.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.main.id
  type    = "PORTS"
  name    = "web-ports"
  ports   = ["80", "443", "8443-8444"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "PORTS"),
					resource.TestCheckResourceAttr(resourceName, "name", "web-ports"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "80"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "8443-8444"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.main.id
  type    = "PORTS"
  name    = "web-ports-updated"
  ports   = ["53", "443", "10000-10010"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "web-ports-updated"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "53"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "10000-10010"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceTrafficMatchingListIPAddresses(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_traffic_matching_list.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_traffic_matching_list" "test" {
  site_id        = data.unifi_site.main.id
  type           = "IPV4_ADDRESSES"
  name           = "protected-ipv4"
  ipv4_addresses = ["192.168.1.10", "192.168.1.0/24", "192.168.1.20-192.168.1.30"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "IPV4_ADDRESSES"),
					resource.TestCheckResourceAttr(resourceName, "name", "protected-ipv4"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ipv4_addresses.*", "192.168.1.10"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ipv4_addresses.*", "192.168.1.0/24"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ipv4_addresses.*", "192.168.1.20-192.168.1.30"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_traffic_matching_list" "test" {
  site_id        = data.unifi_site.main.id
  type           = "IPV6_ADDRESSES"
  name           = "protected-ipv6"
  ipv6_addresses = ["2001:db8::10", "2001:db8::/64"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "IPV6_ADDRESSES"),
					resource.TestCheckResourceAttr(resourceName, "name", "protected-ipv6"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ipv6_addresses.*", "2001:db8::10"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ipv6_addresses.*", "2001:db8::/64"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceDNSPolicy(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_dns_policy.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_dns_policy" "test" {
  site_id      = data.unifi_site.main.id
  type         = "A_RECORD"
  enabled      = true
  domain       = "printer.internal"
  ipv4_address = "10.30.0.50"
  ttl_seconds  = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "A_RECORD"),
					resource.TestCheckResourceAttr(resourceName, "domain", "printer.internal"),
					resource.TestCheckResourceAttr(resourceName, "ipv4_address", "10.30.0.50"),
					resource.TestCheckResourceAttr(resourceName, "ttl_seconds", "300"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_dns_policy" "test" {
  site_id       = data.unifi_site.main.id
  type          = "SRV_RECORD"
  enabled       = true
  domain        = "_ldap._tcp.example.internal"
  server_domain = "ldap01.example.internal"
  service       = "_ldap"
  protocol      = "_tcp"
  port          = 389
  priority      = 10
  weight        = 20
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "SRV_RECORD"),
					resource.TestCheckResourceAttr(resourceName, "server_domain", "ldap01.example.internal"),
					resource.TestCheckResourceAttr(resourceName, "service", "_ldap"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "_tcp"),
					resource.TestCheckResourceAttr(resourceName, "port", "389"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceDHCPReservation(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_dhcp_reservation.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_dhcp_reservation" "test" {
  site_id     = data.unifi_site.main.id
  mac_address = "` + api.existingDHCPReservationMAC + `"
  fixed_ip    = "10.170.0.14"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "site_id", api.siteID),
					resource.TestCheckResourceAttr(resourceName, "mac_address", strings.ToLower(api.existingDHCPReservationMAC)),
					resource.TestCheckResourceAttr(resourceName, "fixed_ip", "10.170.0.14"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s/%s", api.siteID, strings.ToLower(api.existingDHCPReservationMAC))),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_dhcp_reservation" "test" {
  site_id     = data.unifi_site.main.id
  mac_address = "` + api.existingDHCPReservationMAC + `"
  fixed_ip    = "10.170.0.15"
  enabled     = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "fixed_ip", "10.170.0.15"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testImportResourceID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceDHCPReservationCreatesConfiguredClientForAdoptedDevice(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_dhcp_reservation.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_dhcp_reservation" "test" {
  site_id     = data.unifi_site.main.id
  mac_address = "` + api.existingAdoptedDeviceMAC + `"
  fixed_ip    = "10.200.0.25"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "site_id", api.siteID),
					resource.TestCheckResourceAttr(resourceName, "mac_address", api.existingAdoptedDeviceMAC),
					resource.TestCheckResourceAttr(resourceName, "fixed_ip", "10.200.0.25"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					func(_ *tfstate.State) error {
						for _, reservation := range api.dhcpReservations["default"] {
							if strings.EqualFold(reservation.MACAddress, api.existingAdoptedDeviceMAC) {
								if reservation.FixedIP == nil || *reservation.FixedIP != "10.200.0.25" {
									return fmt.Errorf("expected adopted device reservation fixed_ip to be set, got %#v", reservation.FixedIP)
								}
								if reservation.NetworkID == nil || *reservation.NetworkID != api.existingNetworkID {
									return fmt.Errorf("expected adopted device reservation network_id %q, got %#v", api.existingNetworkID, reservation.NetworkID)
								}
								if !reservation.Enabled {
									return fmt.Errorf("expected adopted device reservation to be enabled")
								}
								return nil
							}
						}

						return fmt.Errorf("expected adopted device reservation for MAC %s to be created", api.existingAdoptedDeviceMAC)
					},
				),
			},
		},
	})
}

func TestAccResourceACLRule(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_acl_rule.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "iot" {
  site_id    = data.unifi_site.main.id
  management = "UNMANAGED"
  name       = "iot"
  enabled    = true
  vlan_id    = 80
}

resource "unifi_acl_rule" "test" {
  site_id         = data.unifi_site.main.id
  type            = "IPV4"
  enabled         = true
  name            = "block-iot-dns"
  action          = "BLOCK"
  protocol_filter = ["TCP", "UDP"]

  source_ip_filter = {
    type        = "NETWORKS"
    network_ids = [unifi_network.iot.id]
  }

  destination_ip_filter = {
    type  = "PORTS"
    ports = [53]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "IPV4"),
					resource.TestCheckResourceAttr(resourceName, "action", "BLOCK"),
					resource.TestCheckResourceAttr(resourceName, "source_ip_filter.type", "NETWORKS"),
					resource.TestCheckResourceAttr(resourceName, "destination_ip_filter.type", "PORTS"),
					resource.TestCheckTypeSetElemAttr(resourceName, "protocol_filter.*", "TCP"),
					resource.TestCheckTypeSetElemAttr(resourceName, "protocol_filter.*", "UDP"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "iot" {
  site_id    = data.unifi_site.main.id
  management = "UNMANAGED"
  name       = "iot"
  enabled    = true
  vlan_id    = 80
}

resource "unifi_acl_rule" "test" {
  site_id           = data.unifi_site.main.id
  type              = "MAC"
  enabled           = true
  name              = "allow-printer"
  action            = "ALLOW"
  network_id_filter = unifi_network.iot.id

  source_mac_filter = {
    type         = "MAC_ADDRESSES"
    mac_addresses = ["AA:BB:CC:DD:EE:FF"]
    prefix_length = 48
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "MAC"),
					resource.TestCheckResourceAttr(resourceName, "action", "ALLOW"),
					resource.TestCheckResourceAttrPair(resourceName, "network_id_filter", "unifi_network.iot", "id"),
					resource.TestCheckResourceAttr(resourceName, "source_mac_filter.type", "MAC_ADDRESSES"),
					resource.TestCheckTypeSetElemAttr(resourceName, "source_mac_filter.mac_addresses.*", "AA:BB:CC:DD:EE:FF"),
					resource.TestCheckResourceAttr(resourceName, "source_mac_filter.prefix_length", "48"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"index"},
			},
		},
	})
}

func TestAccResourceACLRuleOrdering(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_acl_rule_ordering.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_acl_rule" "allow_web" {
  site_id = data.unifi_site.main.id
  type    = "IPV4"
  enabled = true
  name    = "allow-web"
  action  = "ALLOW"

  source_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["10.10.0.0/16"]
  }

  destination_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["192.168.10.0/24"]
  }
}

resource "unifi_acl_rule" "block_dns" {
  site_id = data.unifi_site.main.id
  type    = "IPV4"
  enabled = true
  name    = "block-dns"
  action  = "BLOCK"

  source_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["10.20.0.0/16"]
  }

  destination_ip_filter = {
    type  = "PORTS"
    ports = [53]
  }
}

resource "unifi_acl_rule_ordering" "test" {
  site_id              = data.unifi_site.main.id
  ordered_acl_rule_ids = ["` + api.existingACLRuleID + `", unifi_acl_rule.block_dns.id, unifi_acl_rule.allow_web.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ordered_acl_rule_ids.0", api.existingACLRuleID),
					resource.TestCheckResourceAttrPair(resourceName, "ordered_acl_rule_ids.1", "unifi_acl_rule.block_dns", "id"),
					resource.TestCheckResourceAttrPair(resourceName, "ordered_acl_rule_ids.2", "unifi_acl_rule.allow_web", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testACLRuleOrderingImportID(api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceFirewallPolicy(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_firewall_policy.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "trusted" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "trusted"
  enabled                 = true
  vlan_id                 = 60
}

resource "unifi_network" "iot" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "iot"
  enabled                 = true
  vlan_id                 = 61
}

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted"
  network_ids = [unifi_network.trusted.id]
}

resource "unifi_firewall_zone" "iot" {
  site_id     = data.unifi_site.main.id
  name        = "iot"
  network_ids = [unifi_network.iot.id]
}

resource "unifi_firewall_policy" "test" {
  site_id              = data.unifi_site.main.id
  enabled              = true
  name                 = "trusted-to-iot"
  action               = "ALLOW"
  allow_return_traffic = false
  source_zone_id       = unifi_firewall_zone.trusted.id
  source_filter = {
    type                   = "NETWORK"
    network_ids            = [unifi_network.trusted.id]
    network_match_opposite = false
  }
  destination_zone_id = unifi_firewall_zone.iot.id
  destination_filter = {
    type                      = "IP_ADDRESS"
    ip_addresses              = ["10.61.0.0/24"]
    ip_address_match_opposite = false
    port_filter = {
      type           = "PORTS"
      match_opposite = false
      ports          = ["443", "8443-8444"]
    }
  }
  ip_version = "IPV4"
  protocol_filter = {
    type        = "PRESET"
    preset_name = "TCP_UDP"
  }
  connection_state_filter = ["NEW"]
  logging_enabled         = false
  schedule = {
    mode       = "EVERY_DAY"
    start_time = "08:00"
    stop_time  = "18:00"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-to-iot"),
					resource.TestCheckResourceAttr(resourceName, "action", "ALLOW"),
					resource.TestCheckResourceAttr(resourceName, "allow_return_traffic", "false"),
					resource.TestCheckResourceAttr(resourceName, "source_filter.type", "NETWORK"),
					resource.TestCheckResourceAttr(resourceName, "destination_filter.type", "IP_ADDRESS"),
					resource.TestCheckResourceAttr(resourceName, "protocol_filter.type", "PRESET"),
					resource.TestCheckResourceAttr(resourceName, "schedule.mode", "EVERY_DAY"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "trusted" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "trusted"
  enabled                 = true
  vlan_id                 = 60
}

resource "unifi_network" "iot" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "iot"
  enabled                 = true
  vlan_id                 = 61
}

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted"
  network_ids = [unifi_network.trusted.id]
}

resource "unifi_firewall_zone" "iot" {
  site_id     = data.unifi_site.main.id
  name        = "iot"
  network_ids = [unifi_network.iot.id]
}

resource "unifi_traffic_matching_list" "web" {
  site_id = data.unifi_site.main.id
  type    = "PORTS"
  name    = "web-ports"
  ports   = ["443", "8443", "10000-10010"]
}

resource "unifi_firewall_policy" "test" {
  site_id         = data.unifi_site.main.id
  enabled         = true
  name            = "trusted-to-iot-updated"
  action          = "BLOCK"
  source_zone_id  = unifi_firewall_zone.trusted.id
  source_filter = {
    type                      = "VPN_SERVER"
    vpn_server_ids            = ["00000000-0000-0000-0000-000000009999"]
    vpn_server_match_opposite = true
    port_filter = {
      type                     = "TRAFFIC_MATCHING_LIST"
      traffic_matching_list_id = unifi_traffic_matching_list.web.id
      match_opposite           = true
    }
  }
  destination_zone_id = unifi_firewall_zone.iot.id
  destination_filter = {
    type    = "DOMAIN"
    domains = ["example.com", "api.example.com"]
    port_filter = {
      type           = "PORTS"
      match_opposite = false
      ports          = ["443"]
    }
  }
  ip_version = "IPV4_AND_IPV6"
  protocol_filter = {
    type        = "PRESET"
    preset_name = "TCP_UDP"
  }
  logging_enabled = true
  schedule = {
    mode           = "CUSTOM"
    repeat_on_days = ["MONDAY", "WEDNESDAY"]
    start_date     = "2026-01-01"
    stop_date      = "2026-12-31"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-to-iot-updated"),
					resource.TestCheckResourceAttr(resourceName, "action", "BLOCK"),
					resource.TestCheckResourceAttr(resourceName, "logging_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "source_filter.type", "VPN_SERVER"),
					resource.TestCheckResourceAttrPair(resourceName, "source_filter.port_filter.traffic_matching_list_id", "unifi_traffic_matching_list.web", "id"),
					resource.TestCheckResourceAttr(resourceName, "destination_filter.type", "DOMAIN"),
					resource.TestCheckResourceAttr(resourceName, "protocol_filter.type", "PRESET"),
					resource.TestCheckResourceAttr(resourceName, "schedule.mode", "CUSTOM"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"index"},
			},
		},
	})
}

func TestAccResourceFirewallPolicyAllowReturnTrafficNetworkDestination(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_firewall_policy.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "trusted" {
  site_id    = data.unifi_site.main.id
  management = "UNMANAGED"
  name       = "trusted-gap"
  enabled    = true
  vlan_id    = 90
}

resource "unifi_network" "services" {
  site_id    = data.unifi_site.main.id
  management = "UNMANAGED"
  name       = "services-gap"
  enabled    = true
  vlan_id    = 91
}

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted-gap"
  network_ids = [unifi_network.trusted.id]
}

resource "unifi_firewall_zone" "services" {
  site_id     = data.unifi_site.main.id
  name        = "services-gap"
  network_ids = [unifi_network.services.id]
}

resource "unifi_firewall_policy" "test" {
  site_id              = data.unifi_site.main.id
  enabled              = true
  name                 = "trusted-to-services-gap"
  action               = "ALLOW"
  allow_return_traffic = true
  source_zone_id       = unifi_firewall_zone.trusted.id
  source_filter = {
    type                   = "NETWORK"
    network_ids            = [unifi_network.trusted.id]
    network_match_opposite = false
  }
  destination_zone_id = unifi_firewall_zone.services.id
  destination_filter = {
    type                   = "NETWORK"
    network_ids            = [unifi_network.services.id]
    network_match_opposite = false
  }
  ip_version = "IPV4"
  protocol_filter = {
    type        = "PRESET"
    preset_name = "TCP_UDP"
  }
  logging_enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "action", "ALLOW"),
					resource.TestCheckResourceAttr(resourceName, "allow_return_traffic", "true"),
					resource.TestCheckResourceAttr(resourceName, "source_filter.type", "NETWORK"),
					resource.TestCheckResourceAttr(resourceName, "destination_filter.type", "NETWORK"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "destination_filter.network_ids.*", "unifi_network.services", "id"),
				),
			},
		},
	})
}

func TestAccResourceFirewallPolicyOrdering(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_firewall_policy_ordering.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "trusted" {
  site_id    = data.unifi_site.main.id
  management = "UNMANAGED"
  name       = "trusted"
  enabled    = true
  vlan_id    = 70
}

resource "unifi_network" "iot" {
  site_id    = data.unifi_site.main.id
  management = "UNMANAGED"
  name       = "iot"
  enabled    = true
  vlan_id    = 71
}

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted"
  network_ids = [unifi_network.trusted.id]
}

resource "unifi_firewall_zone" "iot" {
  site_id     = data.unifi_site.main.id
  name        = "iot"
  network_ids = [unifi_network.iot.id]
}

resource "unifi_firewall_policy" "allow_https" {
  site_id              = data.unifi_site.main.id
  enabled              = true
  name                 = "allow-https"
  action               = "ALLOW"
  allow_return_traffic = false
  source_zone_id       = unifi_firewall_zone.trusted.id
  destination_zone_id  = unifi_firewall_zone.iot.id
  ip_version           = "IPV4"
  logging_enabled      = false
}

resource "unifi_firewall_policy" "block_dns" {
  site_id             = data.unifi_site.main.id
  enabled             = true
  name                = "block-dns"
  action              = "BLOCK"
  source_zone_id      = unifi_firewall_zone.trusted.id
  destination_zone_id = unifi_firewall_zone.iot.id
  ip_version          = "IPV4"
  logging_enabled     = true
}

resource "unifi_firewall_policy_ordering" "test" {
  site_id                           = data.unifi_site.main.id
  source_zone_id                    = unifi_firewall_zone.trusted.id
  destination_zone_id               = unifi_firewall_zone.iot.id
  before_system_defined_policy_ids  = [unifi_firewall_policy.block_dns.id, unifi_firewall_policy.allow_https.id]
  after_system_defined_policy_ids   = []
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "before_system_defined_policy_ids.0", "unifi_firewall_policy.block_dns", "id"),
					resource.TestCheckResourceAttrPair(resourceName, "before_system_defined_policy_ids.1", "unifi_firewall_policy.allow_https", "id"),
					resource.TestCheckResourceAttr(resourceName, "after_system_defined_policy_ids.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testFirewallPolicyOrderingImportID(resourceName, api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceWifiBroadcast(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_wifi_broadcast.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "test" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "wifi-network"
  enabled                 = true
  vlan_id                 = 70
}

data "unifi_device_tag" "existing" {
  site_id = data.unifi_site.main.id
  id      = "` + api.existingDeviceTagID + `"
}

resource "unifi_wifi_broadcast" "test" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "trusted"
  enabled                                 = true
  client_isolation_enabled                = false
  hide_name                               = false
  uapsd_enabled                           = true
  multicast_to_unicast_conversion_enabled = false
  broadcasting_frequencies_ghz            = [2.4, 5]
  advertise_device_name                   = false
  arp_proxy_enabled                       = false
  band_steering_enabled                   = true
  bss_transition_enabled                  = true

  network = {
    type       = "SPECIFIC"
    network_id = unifi_network.test.id
  }

  security_configuration = {
    type                      = "WPA2_WPA3_PERSONAL"
    passphrase                = "examplepass"
    pmf_mode                  = "OPTIONAL"
    wpa3_fast_roaming_enabled = false
    sae_configuration = {
      anticlogging_threshold_seconds = 5
      sync_time_seconds              = 5
    }
  }

  broadcasting_device_filter = {
    type           = "DEVICE_TAGS"
    device_tag_ids = [data.unifi_device_tag.existing.id]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted"),
					resource.TestCheckResourceAttr(resourceName, "type", "STANDARD"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.type", "DEVICE_TAGS"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.device_tag_ids.#", "1"),
				),
			},
			{
				Config: siteLookupConfig(api.URL()) + `
resource "unifi_network" "test" {
  site_id                 = data.unifi_site.main.id
  management              = "UNMANAGED"
  name                    = "wifi-network"
  enabled                 = true
  vlan_id                 = 70
}

data "unifi_device_tag" "existing" {
  site_id = data.unifi_site.main.id
  id      = "` + api.existingDeviceTagID + `"
}

resource "unifi_wifi_broadcast" "test" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "trusted-updated"
  enabled                                 = true
  client_isolation_enabled                = false
  hide_name                               = true
  uapsd_enabled                           = true
  multicast_to_unicast_conversion_enabled = false
  broadcasting_frequencies_ghz            = [5]
  advertise_device_name                   = true
  arp_proxy_enabled                       = false
  band_steering_enabled                   = false
  bss_transition_enabled                  = true

  network = {
    type       = "SPECIFIC"
    network_id = unifi_network.test.id
  }

  security_configuration = {
    type                      = "WPA2_WPA3_PERSONAL"
    passphrase                = "examplepass"
    pmf_mode                  = "OPTIONAL"
    wpa3_fast_roaming_enabled = false
    sae_configuration = {
      anticlogging_threshold_seconds = 5
      sync_time_seconds              = 5
    }
  }

  broadcasting_device_filter = {
    type           = "DEVICE_TAGS"
    device_tag_ids = [data.unifi_device_tag.existing.id]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-updated"),
					resource.TestCheckResourceAttr(resourceName, "hide_name", "true"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.type", "DEVICE_TAGS"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.device_tag_ids.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
				ImportStateVerify: true,
			},
		},
	})
}

func boolPtr(value bool) *bool {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func portNumberMatch(value int64) client.PortMatch {
	return client.PortMatch{
		Type:  "PORT_NUMBER",
		Value: int64Ptr(value),
	}
}

func portNumberTrafficMatch(value int64) client.TrafficMatchingItem {
	return client.TrafficMatchingItem{
		Type:  "PORT_NUMBER",
		Value: value,
	}
}

func portRangeTrafficMatch(start, stop int64) client.TrafficMatchingItem {
	return client.TrafficMatchingItem{
		Type:  "PORT_NUMBER_RANGE",
		Start: start,
		Stop:  stop,
	}
}
