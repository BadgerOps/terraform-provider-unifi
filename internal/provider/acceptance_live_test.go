package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

type liveAcceptanceConfig struct {
	APIURL        string
	APIKey        string
	AllowInsecure bool
	SiteID        string
	SiteName      string
	NamePrefix    string
}

func TestAccLiveSiteDataSource(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet("data.unifi_site.target", "id"),
		resource.TestCheckResourceAttrSet("data.unifi_site.target", "name"),
		resource.TestCheckResourceAttrSet("data.unifi_site.target", "internal_reference"),
	}
	if config.SiteID != "" {
		checks = append(checks, resource.TestCheckResourceAttr("data.unifi_site.target", "id", config.SiteID))
	}
	if config.SiteName != "" {
		checks = append(checks, resource.TestCheckResourceAttr("data.unifi_site.target", "name", config.SiteName))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config),
				Check:  resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccLiveResourceNetwork(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_network.test"
	networkName := liveAcceptanceName(config, "network")
	updatedName := networkName + "-updated"
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyNetwork(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveNetworkResourceConfig(config, networkName, vlanID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "management", "UNMANAGED"),
					resource.TestCheckResourceAttr(resourceName, "name", networkName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "vlan_id", strconv.FormatInt(vlanID, 10)),
				),
			},
			{
				Config: liveNetworkResourceConfig(config, updatedName, vlanID+1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "vlan_id", strconv.FormatInt(vlanID+1, 10)),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       liveImportCompositeID(resourceName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"default"},
			},
		},
	})
}

func TestAccLiveDataSourceNetwork(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_network.test"
	dataSourceName := "data.unifi_network.lookup"
	networkName := liveAcceptanceName(config, "netds")
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyNetwork(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

data "unifi_network" "lookup" {
  site_id = data.unifi_site.target.id
  id      = unifi_network.test.id
}
`, networkName, vlanID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", networkName),
					resource.TestCheckResourceAttr(dataSourceName, "management", "UNMANAGED"),
					resource.TestCheckResourceAttr(dataSourceName, "vlan_id", strconv.FormatInt(vlanID, 10)),
				),
			},
		},
	})
}

func TestAccLiveResourceTrafficMatchingList(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_traffic_matching_list.test"
	listName := liveAcceptanceName(config, "ports")
	updatedName := liveAcceptanceName(config, "portsu")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyTrafficMatchingList(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.target.id
  type    = "PORTS"
  name    = %q
  ports   = ["80", "443", "8443-8444"]
}
`, listName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "PORTS"),
					resource.TestCheckResourceAttr(resourceName, "name", listName),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "80"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "8443-8444"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.target.id
  type    = "PORTS"
  name    = %q
  ports   = ["53", "443", "10000-10010"]
}
`, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "53"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "10000-10010"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLiveDataSourceTrafficMatchingList(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_traffic_matching_list.test"
	dataSourceName := "data.unifi_traffic_matching_list.lookup"
	listName := liveAcceptanceName(config, "portsds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyTrafficMatchingList(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.target.id
  type    = "PORTS"
  name    = %q
  ports   = ["80", "443", "8443-8444"]
}

data "unifi_traffic_matching_list" "lookup" {
  site_id = data.unifi_site.target.id
  id      = unifi_traffic_matching_list.test.id
}
`, listName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", listName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "PORTS"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "ports.*", "80"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "ports.*", "8443-8444"),
				),
			},
		},
	})
}

func TestAccLiveResourceFirewallZone(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	requireZoneFirewallConfigured(t, config)
	resourceName := "unifi_firewall_zone.test"
	networkName := liveAcceptanceName(config, "zone-net")
	zoneName := liveAcceptanceName(config, "zone")
	updatedZoneName := liveAcceptanceName(config, "zoneu")
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyFirewallZone(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.test.id]
}
`, networkName, vlanID, zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", zoneName),
					resource.TestCheckResourceAttr(resourceName, "network_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "network_ids.*", "unifi_network.test", "id"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.test.id]
}
`, networkName, vlanID, updatedZoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedZoneName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLiveDataSourceFirewallZone(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	requireZoneFirewallConfigured(t, config)
	resourceName := "unifi_firewall_zone.test"
	dataSourceName := "data.unifi_firewall_zone.lookup"
	networkName := liveAcceptanceName(config, "zone-ds-net")
	zoneName := liveAcceptanceName(config, "zone-ds")
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyFirewallZone(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.test.id]
}

data "unifi_firewall_zone" "lookup" {
  site_id = data.unifi_site.target.id
  id      = unifi_firewall_zone.test.id
}
`, networkName, vlanID, zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", zoneName),
					resource.TestCheckResourceAttr(dataSourceName, "network_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(dataSourceName, "network_ids.*", "unifi_network.test", "id"),
				),
			},
		},
	})
}

func TestAccLiveResourceFirewallPolicy(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	requireZoneFirewallConfigured(t, config)
	resourceName := "unifi_firewall_policy.test"
	sourceNetworkName := liveAcceptanceName(config, "policy-src-net")
	destinationNetworkName := liveAcceptanceName(config, "policy-dst-net")
	sourceZoneName := liveAcceptanceName(config, "policy-src-zone")
	destinationZoneName := liveAcceptanceName(config, "policy-dst-zone")
	policyName := liveAcceptanceName(config, "policy")
	updatedPolicyName := liveAcceptanceName(config, "policyu")
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyFirewallPolicy(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "source" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_network" "destination" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "source" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.source.id]
}

resource "unifi_firewall_zone" "destination" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.destination.id]
}

resource "unifi_firewall_policy" "test" {
  site_id             = data.unifi_site.target.id
  enabled             = true
  name                = %q
  action              = "ALLOW"
  source_zone_id      = unifi_firewall_zone.source.id
  destination_zone_id = unifi_firewall_zone.destination.id
  ip_version          = "IPV4"
  logging_enabled     = false
}
`, sourceNetworkName, vlanID, destinationNetworkName, vlanID+1, sourceZoneName, destinationZoneName, policyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", policyName),
					resource.TestCheckResourceAttr(resourceName, "action", "ALLOW"),
					resource.TestCheckResourceAttr(resourceName, "ip_version", "IPV4"),
					resource.TestCheckResourceAttr(resourceName, "logging_enabled", "false"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "source" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_network" "destination" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "source" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.source.id]
}

resource "unifi_firewall_zone" "destination" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.destination.id]
}

resource "unifi_firewall_policy" "test" {
  site_id             = data.unifi_site.target.id
  enabled             = true
  name                = %q
  description         = "updated by live acceptance"
  action              = "BLOCK"
  source_zone_id      = unifi_firewall_zone.source.id
  destination_zone_id = unifi_firewall_zone.destination.id
  ip_version          = "IPV4"
  logging_enabled     = true
}
`, sourceNetworkName, vlanID, destinationNetworkName, vlanID+1, sourceZoneName, destinationZoneName, updatedPolicyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedPolicyName),
					resource.TestCheckResourceAttr(resourceName, "description", "updated by live acceptance"),
					resource.TestCheckResourceAttr(resourceName, "action", "BLOCK"),
					resource.TestCheckResourceAttr(resourceName, "logging_enabled", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLiveResourceWifiBroadcast(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	passphrase := requireWifiPassphrase(t)
	deviceTags := requireDeviceTags(t, config, 2)
	resourceName := "unifi_wifi_broadcast.test"
	broadcastName := liveAcceptanceName(config, "wifi")
	updatedName := liveAcceptanceName(config, "wifiu")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyWifiBroadcast(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_wifi_broadcast" "test" {
  site_id                              = data.unifi_site.target.id
  type                                 = "STANDARD"
  name                                 = %q
  enabled                              = true
  client_isolation_enabled             = false
  hide_name                            = false
  uapsd_enabled                        = true
  multicast_to_unicast_conversion_enabled = true
  broadcasting_frequencies_ghz         = [2.4, 5]
  advertise_device_name                = true
  arp_proxy_enabled                    = false
  bss_transition_enabled               = true

  network = {
    type = "NATIVE"
  }

  security_configuration = {
    type       = "WPA2_PERSONAL"
    passphrase = %q
  }

  broadcasting_device_filter = {
    type           = "DEVICE_TAGS"
    device_tag_ids = [%q]
  }
}
`, broadcastName, passphrase, deviceTags[0].ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", broadcastName),
					resource.TestCheckResourceAttr(resourceName, "type", "STANDARD"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "hide_name", "false"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.type", "DEVICE_TAGS"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.device_tag_ids.#", "1"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_wifi_broadcast" "test" {
  site_id                              = data.unifi_site.target.id
  type                                 = "STANDARD"
  name                                 = %q
  enabled                              = true
  client_isolation_enabled             = false
  hide_name                            = true
  uapsd_enabled                        = true
  multicast_to_unicast_conversion_enabled = true
  broadcasting_frequencies_ghz         = [2.4, 5]
  advertise_device_name                = true
  arp_proxy_enabled                    = false
  band_steering_enabled                = true
  bss_transition_enabled               = true

  network = {
    type = "NATIVE"
  }

  security_configuration = {
    type       = "WPA2_PERSONAL"
    passphrase = %q
  }

  broadcasting_device_filter = {
    type           = "DEVICE_TAGS"
    device_tag_ids = [%q, %q]
  }
}
`, updatedName, passphrase, deviceTags[0].ID, deviceTags[1].ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "hide_name", "true"),
					resource.TestCheckResourceAttr(resourceName, "band_steering_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "broadcasting_device_filter.device_tag_ids.#", "2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       liveImportCompositeID(resourceName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"security_configuration"},
			},
		},
	})
}

func TestAccLiveDataSourceWifiBroadcast(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	passphrase := requireWifiPassphrase(t)
	deviceTags := requireDeviceTags(t, config, 1)
	resourceName := "unifi_wifi_broadcast.test"
	dataSourceName := "data.unifi_wifi_broadcast.lookup"
	broadcastName := liveAcceptanceName(config, "wifids")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyWifiBroadcast(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_wifi_broadcast" "test" {
  site_id                              = data.unifi_site.target.id
  type                                 = "STANDARD"
  name                                 = %q
  enabled                              = true
  client_isolation_enabled             = false
  hide_name                            = false
  uapsd_enabled                        = true
  multicast_to_unicast_conversion_enabled = true
  broadcasting_frequencies_ghz         = [2.4, 5]
  advertise_device_name                = true
  arp_proxy_enabled                    = false
  bss_transition_enabled               = true

  network = {
    type = "NATIVE"
  }

  security_configuration = {
    type       = "WPA2_PERSONAL"
    passphrase = %q
  }

  broadcasting_device_filter = {
    type           = "DEVICE_TAGS"
    device_tag_ids = [%q]
  }
}

data "unifi_wifi_broadcast" "lookup" {
  site_id = data.unifi_site.target.id
  id      = unifi_wifi_broadcast.test.id
}
`, broadcastName, passphrase, deviceTags[0].ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", broadcastName),
					resource.TestCheckResourceAttr(dataSourceName, "type", "STANDARD"),
					resource.TestCheckResourceAttr(dataSourceName, "network.type", "NATIVE"),
					resource.TestCheckResourceAttr(dataSourceName, "broadcasting_device_filter.type", "DEVICE_TAGS"),
					resource.TestCheckResourceAttr(dataSourceName, "broadcasting_device_filter.device_tag_ids.#", "1"),
				),
			},
		},
	})
}

func TestAccLiveResourceDNSPolicy(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_dns_policy.test"
	recordName := liveAcceptanceName(config, "dns")
	updatedText := "live-acceptance-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyDNSPolicy(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_dns_policy" "test" {
  site_id  = data.unifi_site.target.id
  type     = "TXT_RECORD"
  enabled  = true
  domain   = "%s.example.test"
  text     = "live-acceptance"
}
`, recordName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "TXT_RECORD"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "domain", recordName+".example.test"),
					resource.TestCheckResourceAttr(resourceName, "text", "live-acceptance"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_dns_policy" "test" {
  site_id  = data.unifi_site.target.id
  type     = "TXT_RECORD"
  enabled  = true
  domain   = "%s.example.test"
  text     = %q
}
`, recordName, updatedText),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "text", updatedText),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLiveResourceACLRule(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_acl_rule.test"
	ruleName := liveAcceptanceName(config, "acl")
	updatedName := liveAcceptanceName(config, "aclu")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyACLRule(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_acl_rule" "test" {
  site_id  = data.unifi_site.target.id
  type     = "IPV4"
  enabled  = true
  name     = %q
  action   = "BLOCK"
  protocol_filter = ["TCP"]

  source_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["10.0.0.0/8"]
  }

  destination_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["192.168.0.0/16"]
  }
}
`, ruleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", ruleName),
					resource.TestCheckResourceAttr(resourceName, "type", "IPV4"),
					resource.TestCheckResourceAttr(resourceName, "action", "BLOCK"),
					resource.TestCheckTypeSetElemAttr(resourceName, "protocol_filter.*", "TCP"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_acl_rule" "test" {
  site_id      = data.unifi_site.target.id
  type         = "IPV4"
  enabled      = true
  name         = %q
  description  = "updated by live acceptance"
  action       = "ALLOW"
  protocol_filter = ["TCP", "UDP"]

  source_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["10.0.0.0/8"]
  }

  destination_ip_filter = {
    type                    = "IP_ADDRESSES_OR_SUBNETS"
    ip_addresses_or_subnets = ["192.168.0.0/16"]
  }
}
`, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "description", "updated by live acceptance"),
					resource.TestCheckResourceAttr(resourceName, "action", "ALLOW"),
					resource.TestCheckTypeSetElemAttr(resourceName, "protocol_filter.*", "UDP"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLiveDataSourceRadiusProfile(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	profile := requireRadiusProfile(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_radius_profile" "lookup" {
  site_id = data.unifi_site.target.id
  id      = %q
}
`, profile.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_radius_profile.lookup", "id", profile.ID),
					resource.TestCheckResourceAttr("data.unifi_radius_profile.lookup", "name", profile.Name),
				),
			},
		},
	})
}

func TestAccLiveDataSourceDeviceTag(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	tag := requireDeviceTag(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_device_tag" "lookup" {
  site_id = data.unifi_site.target.id
  id      = %q
}
`, tag.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_device_tag.lookup", "id", tag.ID),
					resource.TestCheckResourceAttr("data.unifi_device_tag.lookup", "name", tag.Name),
				),
			},
		},
	})
}

func TestAccLiveDataSourceSwitchDevice(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	device := requireDeviceWithFeature(t, config, "switching")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_device" "lookup" {
  site_id          = data.unifi_site.target.id
  id               = %q
  required_feature = "switching"
}
`, device.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_device.lookup", "id", device.ID),
					resource.TestCheckResourceAttr("data.unifi_device.lookup", "name", device.Name),
					resource.TestCheckResourceAttr("data.unifi_device.lookup", "model", device.Model),
					resource.TestCheckResourceAttr("data.unifi_device.lookup", "mac_address", device.MacAddress),
					resource.TestCheckResourceAttr("data.unifi_device.lookup", "ip_address", device.IPAddress),
					resource.TestCheckResourceAttr("data.unifi_device.lookup", "state", device.State),
					resource.TestCheckTypeSetElemAttr("data.unifi_device.lookup", "features.*", "switching"),
				),
			},
		},
	})
}

func TestAccLiveDataSourceWAN(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	wan := requireWAN(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_wan" "lookup" {
  site_id = data.unifi_site.target.id
  id      = %q
}
`, wan.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_wan.lookup", "id", wan.ID),
					resource.TestCheckResourceAttr("data.unifi_wan.lookup", "name", wan.Name),
				),
			},
		},
	})
}

func TestAccLiveDataSourceSwitchStack(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	stack := requireSwitchStack(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_switch_stack" "lookup" {
  site_id = data.unifi_site.target.id
  id      = %q
}
`, stack.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_switch_stack.lookup", "id", stack.ID),
					resource.TestCheckResourceAttr("data.unifi_switch_stack.lookup", "name", stack.Name),
				),
			},
		},
	})
}

func TestAccLiveDataSourceMcLagDomain(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	domain := requireMcLagDomain(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_mc_lag_domain" "lookup" {
  site_id = data.unifi_site.target.id
  id      = %q
}
`, domain.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_mc_lag_domain.lookup", "id", domain.ID),
					resource.TestCheckResourceAttr("data.unifi_mc_lag_domain.lookup", "name", domain.Name),
				),
			},
		},
	})
}

func TestAccLiveDataSourceLag(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	lag := requireLag(t, config)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
data "unifi_lag" "lookup" {
  site_id = data.unifi_site.target.id
  id      = %q
}
`, lag.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_lag.lookup", "id", lag.ID),
					resource.TestCheckResourceAttr("data.unifi_lag.lookup", "type", lag.Type),
				),
			},
		},
	})
}

func requireLiveAcceptanceConfig(t *testing.T) liveAcceptanceConfig {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run live acceptance tests")
	}

	config := liveAcceptanceConfig{
		APIURL:        strings.TrimSpace(os.Getenv("UNIFI_API_URL")),
		APIKey:        strings.TrimSpace(os.Getenv("UNIFI_API_KEY")),
		SiteID:        strings.TrimSpace(os.Getenv("UNIFI_TEST_SITE_ID")),
		SiteName:      strings.TrimSpace(os.Getenv("UNIFI_TEST_SITE_NAME")),
		NamePrefix:    strings.TrimSpace(os.Getenv("UNIFI_TEST_NAME_PREFIX")),
		AllowInsecure: parseEnvBool("UNIFI_ALLOW_INSECURE"),
	}

	if config.APIURL == "" {
		t.Fatal("UNIFI_API_URL must be set for live acceptance tests")
	}
	if config.APIKey == "" {
		t.Fatal("UNIFI_API_KEY must be set for live acceptance tests")
	}

	selectorCount := 0
	if config.SiteID != "" {
		selectorCount++
	}
	if config.SiteName != "" {
		selectorCount++
	}
	if selectorCount != 1 {
		t.Fatal("set exactly one of UNIFI_TEST_SITE_ID or UNIFI_TEST_SITE_NAME for live acceptance tests")
	}

	if config.NamePrefix == "" {
		config.NamePrefix = "acctest-"
	}

	return config
}

func parseEnvBool(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func requireZoneFirewallConfigured(t *testing.T, config liveAcceptanceConfig) {
	t.Helper()

	if !parseEnvBool("UNIFI_TEST_ENABLE_ZONE_FIREWALL") {
		t.Skip("set UNIFI_TEST_ENABLE_ZONE_FIREWALL=1 to run zone firewall live acceptance tests")
	}

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatalf("build client for zone firewall capability check: %v", err)
	}

	_, err = apiClient.ListFirewallZones(context.Background(), resolveLiveSiteID(t, config))
	if err == nil {
		return
	}

	if isZoneFirewallNotConfigured(err) {
		t.Skip("Zone Based Firewall is not configured on the target site")
	}

	t.Fatalf("check zone firewall capability: %v", err)
}

func requireWifiPassphrase(t *testing.T) string {
	t.Helper()

	passphrase := strings.TrimSpace(os.Getenv("UNIFI_TEST_WIFI_PASSPHRASE"))
	if passphrase == "" {
		t.Skip("set UNIFI_TEST_WIFI_PASSPHRASE to run live WiFi broadcast acceptance tests")
	}

	return passphrase
}

func liveProviderConfig(config liveAcceptanceConfig) string {
	return fmt.Sprintf(`
provider "unifi" {
  api_url        = %q
  api_key        = %q
  allow_insecure = %t
}
`, config.APIURL, config.APIKey, config.AllowInsecure)
}

func liveSiteLookupDataSource(config liveAcceptanceConfig) string {
	switch {
	case config.SiteID != "":
		return fmt.Sprintf(`
data "unifi_site" "target" {
  id = %q
}
`, config.SiteID)
	default:
		return fmt.Sprintf(`
data "unifi_site" "target" {
  name = %q
}
`, config.SiteName)
	}
}

func liveNetworkResourceConfig(config liveAcceptanceConfig, name string, vlanID int64) string {
	return liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}
`, name, vlanID)
}

func liveAcceptanceName(config liveAcceptanceConfig, suffix string) string {
	token := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 36)
	name := fmt.Sprintf("%s%s-%s", config.NamePrefix, suffix, token)
	if len(name) <= 32 {
		return name
	}

	excess := len(name) - 32
	if excess >= len(config.NamePrefix) {
		return name[len(name)-32:]
	}

	return fmt.Sprintf("%s%s-%s", config.NamePrefix[:len(config.NamePrefix)-excess], suffix, token)
}

func liveAcceptanceVLAN() int64 {
	return 2000 + time.Now().UnixNano()%1000
}

func liveImportCompositeID(resourceName string) resource.ImportStateIdFunc {
	return func(state *tfstate.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		siteID := state.RootModule().Resources["data.unifi_site.target"].Primary.ID
		if siteID == "" {
			return "", fmt.Errorf("site lookup data source missing from state")
		}

		return fmt.Sprintf("%s/%s", siteID, resourceState.Primary.ID), nil
	}
}

func liveCheckDestroyNetwork(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := client.New(client.Config{
			BaseURL:       config.APIURL,
			APIKey:        config.APIKey,
			AllowInsecure: config.AllowInsecure,
			UserAgent:     "terraform-provider-unifi/testacc",
		})
		if err != nil {
			return fmt.Errorf("build client for destroy check: %w", err)
		}

		_, err = apiClient.GetNetwork(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify network destroy: %w", err)
		}

		return fmt.Errorf("network %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyFirewallZone(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetFirewallZone(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify firewall zone destroy: %w", err)
		}

		return fmt.Errorf("firewall zone %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyTrafficMatchingList(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetTrafficMatchingList(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify traffic matching list destroy: %w", err)
		}

		return fmt.Errorf("traffic matching list %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyWifiBroadcast(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetWifiBroadcast(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify wifi broadcast destroy: %w", err)
		}

		return fmt.Errorf("wifi broadcast %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyDNSPolicy(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetDNSPolicy(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify dns policy destroy: %w", err)
		}

		return fmt.Errorf("dns policy %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyACLRule(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetACLRule(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify acl rule destroy: %w", err)
		}

		return fmt.Errorf("acl rule %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyFirewallPolicy(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetFirewallPolicy(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify firewall policy destroy: %w", err)
		}

		return fmt.Errorf("firewall policy %s still exists", resourceState.Primary.ID)
	}
}

func newLiveDestroyCheckClient(config liveAcceptanceConfig) (*client.Client, error) {
	apiClient, err := client.New(client.Config{
		BaseURL:       config.APIURL,
		APIKey:        config.APIKey,
		AllowInsecure: config.AllowInsecure,
		UserAgent:     "terraform-provider-unifi/testacc",
	})
	if err != nil {
		return nil, fmt.Errorf("build client for destroy check: %w", err)
	}

	return apiClient, nil
}

func isZoneFirewallNotConfigured(err error) bool {
	var clientErr *client.Error
	return errors.As(err, &clientErr) && clientErr.Code == "api.firewall.zone-based-firewall-not-configured"
}

func requireRadiusProfile(t *testing.T, config liveAcceptanceConfig) client.RadiusProfile {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	profiles, err := apiClient.ListRadiusProfiles(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list radius profiles: %v", err)
	}
	if len(profiles) == 0 {
		t.Skip("no RADIUS profiles found in the target site")
	}

	return profiles[0]
}

func requireDeviceTag(t *testing.T, config liveAcceptanceConfig) client.DeviceTag {
	return requireDeviceTags(t, config, 1)[0]
}

func requireDeviceTags(t *testing.T, config liveAcceptanceConfig, minimum int) []client.DeviceTag {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	tags, err := apiClient.ListDeviceTags(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list device tags: %v", err)
	}
	if len(tags) == 0 {
		t.Skip("no device tags found in the target site")
	}
	if len(tags) < minimum {
		t.Skipf("need at least %d device tags in the target site", minimum)
	}

	return tags
}

func requireWAN(t *testing.T, config liveAcceptanceConfig) client.WAN {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	wans, err := apiClient.ListWANs(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list wans: %v", err)
	}
	if len(wans) == 0 {
		t.Skip("no WANs found in the target site")
	}

	return wans[0]
}

func requireDeviceWithFeature(t *testing.T, config liveAcceptanceConfig, feature string) client.Device {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := apiClient.ListDevices(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}

	for _, device := range devices {
		if deviceHasFeature(device, feature) {
			return device
		}
	}

	t.Skipf("no devices with feature %q found in the target site", feature)
	return client.Device{}
}

func requireSwitchStack(t *testing.T, config liveAcceptanceConfig) client.SwitchStack {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	stacks, err := apiClient.ListSwitchStacks(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list switch stacks: %v", err)
	}
	if len(stacks) == 0 {
		t.Skip("no switch stacks found in the target site")
	}

	return stacks[0]
}

func requireMcLagDomain(t *testing.T, config liveAcceptanceConfig) client.McLagDomain {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	domains, err := apiClient.ListMcLagDomains(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list mc-lag domains: %v", err)
	}
	if len(domains) == 0 {
		t.Skip("no MC-LAG domains found in the target site")
	}

	return domains[0]
}

func requireLag(t *testing.T, config liveAcceptanceConfig) client.Lag {
	t.Helper()

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	lags, err := apiClient.ListLags(context.Background(), resolveLiveSiteID(t, config))
	if err != nil {
		t.Fatalf("list lags: %v", err)
	}
	if len(lags) == 0 {
		t.Skip("no LAGs found in the target site")
	}

	return lags[0]
}

func resolveLiveSiteID(t *testing.T, config liveAcceptanceConfig) string {
	t.Helper()

	if config.SiteID != "" {
		return config.SiteID
	}

	apiClient, err := newLiveDestroyCheckClient(config)
	if err != nil {
		t.Fatal(err)
	}

	sites, err := apiClient.ListSites(context.Background())
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}

	for _, site := range sites {
		if site.Name == config.SiteName {
			return site.ID
		}
	}

	t.Fatalf("site %q not found during live discovery", config.SiteName)
	return ""
}
