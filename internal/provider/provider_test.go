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

	siteID            string
	existingNetworkID string
	existingZoneID    string

	sites            map[string]client.Site
	networks         map[string]map[string]client.Network
	wifiBroadcasts   map[string]map[string]client.WifiBroadcast
	firewallZones    map[string]map[string]client.FirewallZone
	firewallPolicies map[string]map[string]client.FirewallPolicy
}

func newMockUniFiAPI(t *testing.T) *mockUniFiAPI {
	t.Helper()

	api := &mockUniFiAPI{
		nextID:           1,
		sites:            make(map[string]client.Site),
		networks:         make(map[string]map[string]client.Network),
		wifiBroadcasts:   make(map[string]map[string]client.WifiBroadcast),
		firewallZones:    make(map[string]map[string]client.FirewallZone),
		firewallPolicies: make(map[string]map[string]client.FirewallPolicy),
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

resource "unifi_firewall_policy" "test" {
  site_id                     = data.unifi_site.main.id
  enabled                     = true
  name                        = "trusted-to-iot-updated"
  action                      = "BLOCK"
  source_zone_id              = unifi_firewall_zone.trusted.id
  destination_zone_id         = unifi_firewall_zone.iot.id
  destination_network_ids     = [unifi_network.iot.id]
  destination_network_match_opposite = false
  ip_version                  = "IPV4_AND_IPV6"
  logging_enabled             = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "trusted-to-iot-updated"),
					resource.TestCheckResourceAttr(resourceName, "action", "BLOCK"),
					resource.TestCheckResourceAttr(resourceName, "logging_enabled", "true"),
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
