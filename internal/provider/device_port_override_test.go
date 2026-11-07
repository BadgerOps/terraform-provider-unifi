package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	portOverrideSwitchMAC = "0c:00:00:00:00:01"
	portOverrideFamilyID  = "8374e1da-0000-4000-8000-000000000020"
	portOverrideDefaultID = "675db688-0000-4000-8000-000000000001"
	portOverrideLabID     = "c24ccd69-0000-4000-8000-000000000040"
)

// The mock stores what the client PUTs as the device's whole port_overrides
// list, as the controller does, so a client that drops another port's entry or
// an entry's unmodelled fields fails the checks.
func (api *mockUniFiAPI) handleLegacyDevices(w http.ResponseWriter, r *http.Request, segments []string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if segments[1] != "default" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch {
	case segments[2] == "rest" && segments[3] == "networkconf" && r.Method == http.MethodGet:
		api.writeLegacyJSON(w, http.StatusOK, api.legacyNetworkconfs)
	case segments[2] == "stat" && len(segments) == 4 && r.Method == http.MethodGet:
		api.writeLegacyJSON(w, http.StatusOK, api.legacyDevices)
	case segments[2] == "rest" && len(segments) == 5 && r.Method == http.MethodPut:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// The client sends exactly one field: the whole port list, or the
		// management network. The controller merges top-level fields.
		if len(payload) != 1 {
			http.Error(w, "expected exactly one field", http.StatusBadRequest)
			return
		}
		if overrides, ok := payload["port_overrides"]; ok {
			list, ok := overrides.([]any)
			if !ok {
				http.Error(w, "port_overrides must be a list", http.StatusBadRequest)
				return
			}
			for _, raw := range list {
				if _, ok := raw.(map[string]any)["port_idx"].(float64); !ok {
					http.Error(w, "entry without port_idx", http.StatusBadRequest)
					return
				}
			}
		} else if _, ok := payload["mgmt_network_id"].(string); !ok {
			http.Error(w, "expected port_overrides or mgmt_network_id", http.StatusBadRequest)
			return
		}
		for _, device := range api.legacyDevices {
			if device["_id"] == segments[4] {
				for key, value := range payload {
					device[key] = value
				}
				api.devicePuts++
				api.writeLegacyJSON(w, http.StatusOK, []any{})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// seedPortOverrides is the shape of a USW Flex Mini from a real controller
// (2026-10-21): port 2 and 4 pinned to one network, no entry for port 3.
func seedPortOverrides(api *mockUniFiAPI) {
	api.legacyNetworkconfs = []map[string]any{
		{"_id": "legacy-family", "external_id": portOverrideFamilyID, "name": "Family"},
		{"_id": "legacy-default", "external_id": portOverrideDefaultID, "name": "Default"},
		{"_id": "legacy-lab", "external_id": portOverrideLabID, "name": "Lab"},
		{"_id": "legacy-wan", "name": "WAN"},
	}
	entry := func(port int, name string) map[string]any {
		return map[string]any{
			"port_idx": port, "name": name, "forward": "native", "tagged_vlan_mgmt": "block_all",
			"native_networkconf_id": "legacy-family", "setting_preference": "auto", "autoneg": true,
			"port_security_mac_address": []any{}, "stp_port_mode": true, "voice_networkconf_id": "",
		}
	}
	api.legacyDevices = []map[string]any{
		{"_id": "dev-ap", "mac": "0c:00:00:00:00:99", "type": "uap"},
		{"_id": "dev-mini", "mac": portOverrideSwitchMAC, "type": "usw", "name": "USW Flex Mini",
			"port_overrides": []any{entry(2, "Port 2"), entry(4, "Port 4")}},
	}
}

func portOverrideTestConfig(api *mockUniFiAPI, body string) string {
	return siteLookupConfig(api.URL()) + body
}

func devicePortOverrides(api *mockUniFiAPI) []any {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.legacyDevices[1]["port_overrides"].([]any)
}

func portEntry(api *mockUniFiAPI, port int) map[string]any {
	for _, raw := range devicePortOverrides(api) {
		entry := raw.(map[string]any)
		if entry["port_idx"] == float64(port) || entry["port_idx"] == port {
			return entry
		}
	}
	return nil
}

func TestAccResourceDevicePortOverride(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	port4Before := fmt.Sprint(portEntry(api, 4))

	two := "unifi_device_port_override.two"
	three := "unifi_device_port_override.three"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Port 2 has an entry: only the native network changes. Port 3
				// has none: it is appended. Both on one switch, in parallel.
				Config: portOverrideTestConfig(api, fmt.Sprintf(`
resource "unifi_device_port_override" "two" {
  site_id           = data.unifi_site.main.id
  device_mac        = "0c:00:00:00:00:01"
  port_idx          = 2
  native_network_id = %q
}

resource "unifi_device_port_override" "three" {
  site_id              = data.unifi_site.main.id
  device_mac           = %q
  port_idx             = 3
  name                 = "Lab spare"
  native_network_id    = %q
  tagged_vlan_mgmt     = "custom"
  excluded_network_ids = [%q]
}
`, portOverrideLabID, portOverrideSwitchMAC, portOverrideLabID, portOverrideFamilyID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(two, "device_mac", portOverrideSwitchMAC),
					resource.TestCheckResourceAttr(two, "id", api.siteID+"/"+portOverrideSwitchMAC+"/2"),
					resource.TestCheckResourceAttr(two, "name", "Port 2"),
					resource.TestCheckResourceAttr(two, "native_network_id", portOverrideLabID),
					resource.TestCheckResourceAttr(two, "tagged_vlan_mgmt", "block_all"),
					resource.TestCheckResourceAttr(two, "excluded_network_ids.#", "0"),
					resource.TestCheckResourceAttrPair(two, "site_id", "data.unifi_site.main", "id"),
					resource.TestCheckResourceAttr(three, "excluded_network_ids.#", "1"),
					resource.TestCheckResourceAttr(three, "tagged_vlan_mgmt", "custom"),
					func(_ *tfstate.State) error {
						entry := portEntry(api, 2)
						if entry["native_networkconf_id"] != "legacy-lab" {
							return fmt.Errorf("port 2 native network: %v", entry["native_networkconf_id"])
						}
						for _, key := range []string{"forward", "setting_preference", "autoneg", "stp_port_mode", "voice_networkconf_id", "port_security_mac_address"} {
							if _, ok := entry[key]; !ok {
								return fmt.Errorf("port 2 lost unmodelled field %s", key)
							}
						}
						if _, ok := entry["excluded_networkconf_ids"]; ok {
							return fmt.Errorf("port 2 got an exclusion list it didn't have")
						}
						if got := fmt.Sprint(portEntry(api, 4)); got != port4Before {
							return fmt.Errorf("port 4 changed:\n%s\n%s", port4Before, got)
						}
						three := portEntry(api, 3)
						if three == nil || three["name"] != "Lab spare" || !reflect.DeepEqual(three["excluded_networkconf_ids"], []any{"legacy-family"}) {
							return fmt.Errorf("port 3 entry: %#v", three)
						}
						if n := len(devicePortOverrides(api)); n != 3 {
							return fmt.Errorf("want 3 entries, got %d", n)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      two,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *tfstate.State) (string, error) {
					return state.RootModule().Resources[two].Primary.Attributes["id"], nil
				},
			},
			{
				// A rename, in place, keeps the native network the config no longer sets.
				Config: portOverrideRenameConfig(api),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(two, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(two, "name", "Brother printer"),
					resource.TestCheckResourceAttr(two, "native_network_id", portOverrideLabID),
					func(_ *tfstate.State) error {
						if entry := portEntry(api, 2); entry["name"] != "Brother printer" || entry["native_networkconf_id"] != "legacy-lab" {
							return fmt.Errorf("port 2 entry: %#v", entry)
						}
						return nil
					},
				),
			},
			{
				// The entry was removed in the UI: the resource is gone from
				// state and the next apply writes it again.
				PreConfig: func() {
					api.mu.Lock()
					defer api.mu.Unlock()
					kept := []any{}
					for _, raw := range api.legacyDevices[1]["port_overrides"].([]any) {
						if raw.(map[string]any)["port_idx"] != float64(2) {
							kept = append(kept, raw)
						}
					}
					api.legacyDevices[1]["port_overrides"] = kept
				},
				Config: portOverrideRenameConfig(api),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(two, plancheck.ResourceActionCreate),
				}},
				Check: func(_ *tfstate.State) error {
					if entry := portEntry(api, 2); entry == nil || entry["name"] != "Brother printer" {
						return fmt.Errorf("port 2 not written again: %#v", entry)
					}
					return nil
				},
			},
		},
		// Destroy only forgets the ports; resetting them would move the
		// devices plugged in to the default network.
		CheckDestroy: func(_ *tfstate.State) error {
			if n := len(devicePortOverrides(api)); n != 3 {
				return fmt.Errorf("destroy changed the device: %d entries", n)
			}
			return nil
		},
	})
}

func portOverrideRenameConfig(api *mockUniFiAPI) string {
	return portOverrideTestConfig(api, `
resource "unifi_device_port_override" "two" {
  site_id    = data.unifi_site.main.id
  device_mac = "0c:00:00:00:00:01"
  port_idx   = 2
  name       = "Brother printer"
}
`)
}

func TestResourceDevicePortOverrideErrors(t *testing.T) {
	cases := map[string]struct{ body, err string }{
		"unknown native network": {`native_network_id = "00000000-0000-4000-8000-000000000000"`, "No network with ID"},
		"unknown excluded network": {`
  tagged_vlan_mgmt     = "custom"
  excluded_network_ids = ["00000000-0000-4000-8000-000000000000"]`, "No network with ID"},
		"bad tagged mode": {`tagged_vlan_mgmt = "some"`, "Must be one of: auto, block_all, custom"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			api := newMockUniFiAPI(t)
			defer api.Close()
			seedPortOverrides(api)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: portOverrideTestConfig(api, fmt.Sprintf(`
resource "unifi_device_port_override" "test" {
  site_id    = data.unifi_site.main.id
  device_mac = %q
  port_idx   = 2
  %s
}
`, portOverrideSwitchMAC, testCase.body)),
					ExpectError: regexp.MustCompile(testCase.err),
				}},
			})
			if api.devicePuts != 0 {
				t.Fatalf("errors must stop before any write, got %d PUTs", api.devicePuts)
			}
		})
	}
}

func TestResourceDevicePortOverrideMissingDevice(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: portOverrideTestConfig(api, `
resource "unifi_device_port_override" "test" {
  site_id    = data.unifi_site.main.id
  device_mac = "0c:00:00:00:00:42"
  port_idx   = 1
}
`),
			ExpectError: regexp.MustCompile("device 0c:00:00:00:00:42 not found"),
		}},
	})
}

func TestResourceDevicePortOverrideImportID(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	seedPortOverrides(api)
	for _, id := range []string{"only-site", "site/" + portOverrideSwitchMAC + "/zero", "site/" + portOverrideSwitchMAC + "/0", "/mac/1"} {
		t.Run(id, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: portOverrideTestConfig(api, `resource "unifi_device_port_override" "test" {
  site_id    = data.unifi_site.main.id
  device_mac = "0c:00:00:00:00:01"
  port_idx   = 2
}
`),
					ResourceName:  "unifi_device_port_override.test",
					ImportState:   true,
					ImportStateId: id,
					ExpectError:   regexp.MustCompile("Invalid import identifier"),
				}},
			})
		})
	}
}

func TestResourceDevicePortOverrideOptionalValues(t *testing.T) {
	for _, taggedMode := range []string{"custom", "block_all"} {
		t.Run(taggedMode, func(t *testing.T) {
			api := newMockUniFiAPI(t)
			defer api.Close()
			seedPortOverrides(api)
			entry := portEntry(api, 2)
			entry["tagged_vlan_mgmt"] = taggedMode
			entry["excluded_networkconf_ids"] = []any{"legacy-lab"}
			config := func(attributes string) string {
				return portOverrideTestConfig(api, fmt.Sprintf(`
resource "unifi_device_port_override" "test" {
 site_id = data.unifi_site.main.id
 device_mac = %q
 port_idx = 2
 tagged_vlan_mgmt = %q
 %s
}
`, portOverrideSwitchMAC, taggedMode, attributes))
			}
			check := func(name string, excluded []any) resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_port_override.test", "name", name),
					resource.TestCheckResourceAttr("unifi_device_port_override.test", "excluded_network_ids.#", fmt.Sprint(len(excluded))),
					func(_ *tfstate.State) error {
						entry := portEntry(api, 2)
						if entry["name"] != name || !reflect.DeepEqual(entry["excluded_networkconf_ids"], excluded) {
							return fmt.Errorf("unexpected port settings: %#v", entry)
						}
						return nil
					},
				)
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{ // Taking over an existing port preserves omitted values.
						Config: config(""),
						Check:  check("Port 2", []any{"legacy-lab"}),
					},
					{ // An explicit empty name must clear the controller's name.
						Config: config(`name = ""`),
						Check:  check("", []any{"legacy-lab"}),
					},
					{ // An explicit empty set clears exclusions in either tagged mode.
						Config: config(`name = ""
excluded_network_ids = []`),
						Check: check("", []any{}),
					},
				},
			})
		})
	}
}
