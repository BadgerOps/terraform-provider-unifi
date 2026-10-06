package provider

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

// This opt-in test creates disabled rules on the configured disposable site.
// The explicit target address prevents guessing a suitable internal host.
func TestAccLiveResourcePortForward(t *testing.T) {
	config := requireLiveAcceptanceConfig(t)
	forwardIP := os.Getenv("UNIFI_TEST_PORT_FORWARD_IP")
	if forwardIP == "" {
		t.Skip("set UNIFI_TEST_PORT_FORWARD_IP to an IPv4 address on the disposable test site")
	}
	if ip, err := netip.ParseAddr(forwardIP); err != nil || !ip.Is4() {
		t.Fatal("UNIFI_TEST_PORT_FORWARD_IP must be an IPv4 address")
	}
	name := liveAcceptanceName(config, "port-forward")
	resourceName := "unifi_port_forward.test"
	testConfig := func(ports, ruleName string) string {
		return liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_port_forward" "test" {
  site_id = data.unifi_site.target.id
  name = %q
  enabled = false
  protocol = "udp"
  destination_port = %q
  forward_ip = %q
  forward_port = %q
}
data "unifi_port_forward" "test" {
  site_id = data.unifi_site.target.id
  id = unifi_port_forward.test.id
}
`, ruleName, ports, forwardIP, ports)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(state *tfstate.State) error {
			rule, ok := state.RootModule().Resources[resourceName]
			if !ok || rule.Primary.ID == "" {
				return nil
			}
			c, err := newLiveDestroyCheckClient(config)
			if err != nil {
				return err
			}
			_, err = c.GetPortForward(context.Background(), rule.Primary.Attributes["site_id"], rule.Primary.ID)
			if client.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return fmt.Errorf("port forward %s survived destroy", rule.Primary.ID)
		},
		Steps: []resource.TestStep{
			{Config: testConfig("53074", name), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "destination_port", "53074"),
				resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				resource.TestCheckResourceAttrPair("data.unifi_port_forward.test", "id", resourceName, "id"),
			)},
			{Config: testConfig("53074-53076", name+"-updated"), Check: resource.TestCheckResourceAttr(resourceName, "destination_port", "53074-53076")},
			{ResourceName: resourceName, ImportState: true, ImportStateVerify: true, ImportStateIdFunc: liveImportCompositeID(resourceName)},
			{Config: testConfig("53074,53076", name+"-updated"), Check: resource.TestCheckResourceAttr(resourceName, "forward_port", "53074,53076")},
		},
	})
}
