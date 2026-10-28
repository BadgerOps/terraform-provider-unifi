package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLiveResourceWifiBroadcastFrequenciesPlanStable(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	passphrase := requireWifiPassphrase(t)
	deviceTags := requireDeviceTags(t, config, 1)
	name := liveAcceptanceName(config, "freq")

	testConfig := liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_wifi_broadcast" "test" {
  site_id                                 = data.unifi_site.target.id
  type                                    = "STANDARD"
  name                                    = %q
  enabled                                 = true
  client_isolation_enabled                = false
  hide_name                               = false
  uapsd_enabled                           = true
  multicast_to_unicast_conversion_enabled = true
  broadcasting_frequencies_ghz            = [2.4, 5]
  advertise_device_name                   = true
  arp_proxy_enabled                       = false
  bss_transition_enabled                  = true
  network                                 = { type = "NATIVE" }
  security_configuration = {
    type                 = "WPA2_PERSONAL"
    passphrase           = %q
    fast_roaming_enabled = true
  }
  broadcasting_device_filter = {
    type           = "DEVICE_TAGS"
    device_tag_ids = [%q]
  }
}
`, name, passphrase, deviceTags[0].ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyWifiBroadcast(config, "unifi_wifi_broadcast.test"),
		Steps: []resource.TestStep{
			{Config: testConfig},
			{Config: testConfig, PlanOnly: true},
		},
	})
}
