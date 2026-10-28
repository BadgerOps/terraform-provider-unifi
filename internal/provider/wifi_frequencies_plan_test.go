package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResourceWifiBroadcastFrequenciesPlanStable reproduces #45: a STANDARD
// broadcast with 2.4 GHz in broadcasting_frequencies_ghz must not plan an
// in-place update immediately after apply.
func TestAccResourceWifiBroadcastFrequenciesPlanStable(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	config := siteLookupConfig(api.URL()) + `
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
  name                                    = "freq-stable"
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
    type       = "OPEN"
    encryption = "ENHANCED_OPEN"
  }
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
			{
				ResourceName:      "unifi_wifi_broadcast.test",
				ImportState:       true,
				ImportStateIdFunc: testImportCompositeID("unifi_wifi_broadcast.test", api.siteID),
				ImportStateVerify: true,
			},
			{Config: config, PlanOnly: true},
		},
	})
}
