package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"
)

func managementNetworkConfig(api *mockUniFiAPI, networkID string) string {
	return portOverrideTestConfig(api, fmt.Sprintf(`
resource "unifi_device_management_network" "mini" {
  site_id    = data.unifi_site.main.id
  device_mac = "0c:00:00:00:00:01"
  network_id = %q
}
`, networkID))
}

func deviceField(api *mockUniFiAPI, key string) any {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.legacyDevices[1][key]
}

func TestAccResourceDeviceManagementNetwork(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	// As on the real Flex Mini: it lives in Family.
	api.legacyDevices[1]["mgmt_network_id"] = "legacy-family"
	portsBefore := fmt.Sprint(deviceField(api, "port_overrides"))

	mini := "unifi_device_management_network.mini"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ResourceName:       mini,
				Config:             managementNetworkConfig(api, portOverrideFamilyID),
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(state *tfstate.State) (string, error) {
					return api.siteID + "/" + portOverrideSwitchMAC, nil
				},
				ImportStateCheck: func(states []*tfstate.InstanceState) error {
					if got := states[0].Attributes["network_id"]; got != portOverrideFamilyID {
						return fmt.Errorf("imported network_id %q", got)
					}
					return nil
				},
			},
			{
				// The move to Default is an in-place update of one field.
				Config: managementNetworkConfig(api, portOverrideDefaultID),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(mini, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mini, "network_id", portOverrideDefaultID),
					resource.TestCheckResourceAttr(mini, "id", api.siteID+"/"+portOverrideSwitchMAC),
					func(_ *tfstate.State) error {
						if got := deviceField(api, "mgmt_network_id"); got != "legacy-default" {
							return fmt.Errorf("mgmt_network_id %v", got)
						}
						if got := fmt.Sprint(deviceField(api, "port_overrides")); got != portsBefore {
							return fmt.Errorf("ports changed")
						}
						return nil
					},
				),
			},
			{
				// Cleared in the UI: the plan shows the network again.
				PreConfig: func() {
					api.mu.Lock()
					defer api.mu.Unlock()
					delete(api.legacyDevices[1], "mgmt_network_id")
				},
				Config:             managementNetworkConfig(api, portOverrideDefaultID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
		CheckDestroy: func(_ *tfstate.State) error {
			if api.devicePuts != 1 {
				return fmt.Errorf("want exactly one write, got %d", api.devicePuts)
			}
			return nil
		},
	})
}

func TestResourceDeviceManagementNetworkErrors(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      managementNetworkConfig(api, "00000000-0000-4000-8000-000000000000"),
				ExpectError: regexp.MustCompile("No network with ID"),
			},
			{
				Config: portOverrideTestConfig(api, fmt.Sprintf(`
resource "unifi_device_management_network" "missing" {
  site_id    = data.unifi_site.main.id
  device_mac = "0c:00:00:00:00:42"
  network_id = %q
}
`, portOverrideDefaultID)),
				ExpectError: regexp.MustCompile("device 0c:00:00:00:00:42 not found"),
			},
			{
				Config:        managementNetworkConfig(api, portOverrideDefaultID),
				ResourceName:  "unifi_device_management_network.mini",
				ImportState:   true,
				ImportStateId: "only-site",
				ExpectError:   regexp.MustCompile("Invalid import identifier"),
			},
		},
	})
	if api.devicePuts != 0 {
		t.Fatalf("errors must stop before any write, got %d PUTs", api.devicePuts)
	}
}

func TestResourceDeviceManagementNetworkWithoutIntegrationID(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	api.legacyDevices[1]["mgmt_network_id"] = "legacy-wan"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			ResourceName:      "unifi_device_management_network.mini",
			Config:            managementNetworkConfig(api, portOverrideDefaultID),
			ImportState:       true,
			ImportStateIdFunc: func(*tfstate.State) (string, error) { return api.siteID + "/" + portOverrideSwitchMAC, nil },
			ExpectError:       regexp.MustCompile("Network without an integration ID"),
		}},
	})
}

func TestResourceDeviceMACMustBeLowerCase(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	for name, body := range map[string]string{
		"port override": `resource "unifi_device_port_override" "x" {
  site_id    = data.unifi_site.main.id
  device_mac = "0C:00:00:00:00:01"
  port_idx   = 2
}`,
		"management network": fmt.Sprintf(`resource "unifi_device_management_network" "x" {
  site_id    = data.unifi_site.main.id
  device_mac = "0c-00-00-00-00-01"
  network_id = %q
}`, portOverrideDefaultID),
	} {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      portOverrideTestConfig(api, body),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile("lower case with colons"),
				}},
			})
		})
	}
}
