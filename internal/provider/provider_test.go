package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

type mockUniFiAPI struct {
	server *httptest.Server

	mu sync.Mutex

	nextID int

	siteID                        string
	existingNetworkID             string
	existingZoneID                string
	existingTrafficMatchingListID string
	existingRadiusProfileID       string
	existingDeviceTagID           string

	sites                map[string]client.Site
	networks             map[string]map[string]client.Network
	wifiBroadcasts       map[string]map[string]client.WifiBroadcast
	firewallZones        map[string]map[string]client.FirewallZone
	firewallPolicies     map[string]map[string]client.FirewallPolicy
	trafficMatchingLists map[string]map[string]client.TrafficMatchingList
	radiusProfiles       map[string]map[string]client.RadiusProfile
	deviceTags           map[string]map[string]client.DeviceTag
	dnsPolicies          map[string]map[string]client.DNSPolicy
	aclRules             map[string]map[string]client.ACLRule
}

func newMockUniFiAPI(t *testing.T) *mockUniFiAPI {
	t.Helper()

	api := &mockUniFiAPI{
		nextID:               1,
		sites:                make(map[string]client.Site),
		networks:             make(map[string]map[string]client.Network),
		wifiBroadcasts:       make(map[string]map[string]client.WifiBroadcast),
		firewallZones:        make(map[string]map[string]client.FirewallZone),
		firewallPolicies:     make(map[string]map[string]client.FirewallPolicy),
		trafficMatchingLists: make(map[string]map[string]client.TrafficMatchingList),
		radiusProfiles:       make(map[string]map[string]client.RadiusProfile),
		deviceTags:           make(map[string]map[string]client.DeviceTag),
		dnsPolicies:          make(map[string]map[string]client.DNSPolicy),
		aclRules:             make(map[string]map[string]client.ACLRule),
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
	api.trafficMatchingLists[api.siteID] = make(map[string]client.TrafficMatchingList)
	api.radiusProfiles[api.siteID] = make(map[string]client.RadiusProfile)
	api.deviceTags[api.siteID] = make(map[string]client.DeviceTag)
	api.dnsPolicies[api.siteID] = make(map[string]client.DNSPolicy)
	api.aclRules[api.siteID] = make(map[string]client.ACLRule)

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

	existingTrafficMatchingList := client.TrafficMatchingList{
		ID:   api.newID(),
		Type: "PORTS",
		Name: "existing-web-ports",
		Items: []client.PortMatch{
			portNumberMatch(80),
			portRangeMatch(443, 444),
		},
	}
	api.existingTrafficMatchingListID = existingTrafficMatchingList.ID
	api.trafficMatchingLists[api.siteID][existingTrafficMatchingList.ID] = existingTrafficMatchingList

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

func (api *mockUniFiAPI) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("X-API-KEY") != "test-key" {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(request.URL.Path, "/integration") {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	trimmedPath := strings.TrimPrefix(request.URL.Path, "/integration")
	segments := strings.Split(strings.Trim(trimmedPath, "/"), "/")
	if len(segments) < 2 || segments[0] != "v1" || segments[1] != "sites" {
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
	case len(segments) == 4 && segments[3] == "device-tags":
		api.handleDeviceTags(writer, request, segments[2])
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
	case len(segments) == 5 && segments[3] == "acl-rules":
		api.handleACLRule(writer, request, segments[2], segments[4])
		return
	default:
		writer.WriteHeader(http.StatusNotFound)
	}
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
			return policies[i].ID < policies[j].ID
		})
		writePage(writer, request, policies)
	case http.MethodPost:
		var policy client.FirewallPolicy
		api.decodeRequest(writer, request, &policy)
		policy.ID = api.newID()
		policy.Index = int64(len(api.firewallPolicies[siteID]))
		api.firewallPolicies[siteID][policy.ID] = policy
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
		updated.ID = policyID
		updated.Index = policy.Index
		api.firewallPolicies[siteID][policyID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.firewallPolicies[siteID], policyID)
		writer.WriteHeader(http.StatusOK)
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
			return rules[i].ID < rules[j].ID
		})
		writePage(writer, request, rules)
	case http.MethodPost:
		var rule client.ACLRule
		api.decodeRequest(writer, request, &rule)
		rule.ID = api.newID()
		rule.Index = int64(len(api.aclRules[siteID]))
		api.aclRules[siteID][rule.ID] = rule
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
		updated.Index = rule.Index
		api.aclRules[siteID][aclRuleID] = updated
		api.writeJSON(writer, http.StatusOK, updated)
	case http.MethodDelete:
		delete(api.aclRules[siteID], aclRuleID)
		writer.WriteHeader(http.StatusOK)
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

func TestAccDataSources(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(api.URL()) + `
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_network" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-network"
}

data "unifi_firewall_zone" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-zone"
}

data "unifi_traffic_matching_list" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-web-ports"
}

data "unifi_radius_profile" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-radius"
}

data "unifi_device_tag" "existing" {
  site_id = data.unifi_site.main.id
  name    = "existing-tag"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_site.main", "id", api.siteID),
					resource.TestCheckResourceAttr("data.unifi_site.main", "internal_reference", "default"),
					resource.TestCheckResourceAttr("data.unifi_network.existing", "id", api.existingNetworkID),
					resource.TestCheckResourceAttr("data.unifi_network.existing", "management", "GATEWAY"),
					resource.TestCheckResourceAttr("data.unifi_network.existing", "vlan_id", "200"),
					resource.TestCheckResourceAttr("data.unifi_firewall_zone.existing", "id", api.existingZoneID),
					resource.TestCheckResourceAttr("data.unifi_firewall_zone.existing", "name", "existing-zone"),
					resource.TestCheckTypeSetElemAttr("data.unifi_firewall_zone.existing", "network_ids.*", api.existingNetworkID),
					resource.TestCheckResourceAttr("data.unifi_traffic_matching_list.existing", "id", api.existingTrafficMatchingListID),
					resource.TestCheckResourceAttr("data.unifi_traffic_matching_list.existing", "type", "PORTS"),
					resource.TestCheckTypeSetElemAttr("data.unifi_traffic_matching_list.existing", "ports.*", "80"),
					resource.TestCheckTypeSetElemAttr("data.unifi_traffic_matching_list.existing", "ports.*", "443-444"),
					resource.TestCheckResourceAttr("data.unifi_radius_profile.existing", "id", api.existingRadiusProfileID),
					resource.TestCheckResourceAttr("data.unifi_radius_profile.existing", "name", "existing-radius"),
					resource.TestCheckResourceAttr("data.unifi_device_tag.existing", "id", api.existingDeviceTagID),
					resource.TestCheckResourceAttr("data.unifi_device_tag.existing", "name", "existing-tag"),
					resource.TestCheckResourceAttr("data.unifi_device_tag.existing", "device_ids.#", "1"),
				),
			},
		},
	})
}

func TestAccResourceNetwork(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_network.test"

	resource.Test(t, resource.TestCase{
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

	resource.Test(t, resource.TestCase{
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

	resource.Test(t, resource.TestCase{
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

func TestAccResourceDNSPolicy(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_dns_policy.test"

	resource.Test(t, resource.TestCase{
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

func TestAccResourceACLRule(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_acl_rule.test"

	resource.Test(t, resource.TestCase{
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

func TestAccResourceFirewallPolicy(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_firewall_policy.test"

	resource.Test(t, resource.TestCase{
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
  site_id                 = data.unifi_site.main.id
  enabled                 = true
  name                    = "trusted-to-iot"
  action                  = "ALLOW"
  source_zone_id          = unifi_firewall_zone.trusted.id
  destination_zone_id     = unifi_firewall_zone.iot.id
  destination_network_ids = [unifi_network.iot.id]
  ip_version              = "IPV4_AND_IPV6"
  logging_enabled         = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-to-iot"),
					resource.TestCheckResourceAttr(resourceName, "action", "ALLOW"),
					resource.TestCheckResourceAttr(resourceName, "destination_network_ids.#", "1"),
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
  site_id             = data.unifi_site.main.id
  enabled             = true
  name                = "trusted-to-iot-updated"
  action              = "BLOCK"
  source_zone_id      = unifi_firewall_zone.trusted.id
  destination_zone_id = unifi_firewall_zone.iot.id
  destination_port_filter = {
    type                     = "TRAFFIC_MATCHING_LIST"
    traffic_matching_list_id = unifi_traffic_matching_list.web.id
    match_opposite           = true
  }
  ip_version      = "IPV4_AND_IPV6"
  logging_enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-to-iot-updated"),
					resource.TestCheckResourceAttr(resourceName, "action", "BLOCK"),
					resource.TestCheckResourceAttr(resourceName, "logging_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "destination_port_filter.type", "TRAFFIC_MATCHING_LIST"),
					resource.TestCheckResourceAttrPair(resourceName, "destination_port_filter.traffic_matching_list_id", "unifi_traffic_matching_list.web", "id"),
					resource.TestCheckResourceAttr(resourceName, "destination_port_filter.match_opposite", "true"),
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

func TestAccResourceWifiBroadcast(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_wifi_broadcast.test"

	resource.Test(t, resource.TestCase{
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
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted"),
					resource.TestCheckResourceAttr(resourceName, "type", "STANDARD"),
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
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-updated"),
					resource.TestCheckResourceAttr(resourceName, "hide_name", "true"),
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

func portNumberMatch(value int64) client.PortMatch {
	return client.PortMatch{
		Type:  "PORT_NUMBER",
		Value: int64Ptr(value),
	}
}

func portRangeMatch(start, stop int64) client.PortMatch {
	return client.PortMatch{
		Type:  "PORT_NUMBER_RANGE",
		Start: int64Ptr(start),
		Stop:  int64Ptr(stop),
	}
}
