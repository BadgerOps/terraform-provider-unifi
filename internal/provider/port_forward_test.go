package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Use wire-shaped objects so the mock does not silently share serialization bugs
// with the client model. The routes and fields come from issue #34's GET capture;
// write behavior still needs verification against a live controller.
func (api *mockUniFiAPI) handlePortForwards(w http.ResponseWriter, r *http.Request, segments []string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if segments[1] != "default" || len(segments) > 5 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if len(segments) == 4 && r.Method == http.MethodGet {
		rules := []map[string]any{}
		for _, rule := range api.portForwards {
			rules = append(rules, rule)
		}
		api.writeLegacyJSON(w, http.StatusOK, rules)
		return
	}
	if len(segments) == 5 {
		if _, ok := api.portForwards[segments[4]]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodDelete {
			delete(api.portForwards, segments[4])
			api.writeLegacyJSON(w, http.StatusOK, []any{})
			return
		}
	}
	if (len(segments) == 4 && r.Method == http.MethodPost) || (len(segments) == 5 && r.Method == http.MethodPut) {
		var rule map[string]any
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		for _, key := range []string{"name", "pfwd_interface", "proto", "src", "dst_port", "fwd", "fwd_port"} {
			if _, ok := rule[key].(string); !ok {
				http.Error(w, "missing string field: "+key, 400)
				return
			}
		}
		for _, key := range []string{"enabled", "log", "src_limiting_enabled"} {
			if _, ok := rule[key].(bool); !ok {
				http.Error(w, "missing bool field: "+key, 400)
				return
			}
		}
		if rule["src_firewall_group_id"] == "" && rule["src_limiting_enabled"] != (rule["src"] != "any") {
			http.Error(w, "incorrect source limiting", 400)
			return
		}
		status := http.StatusOK
		if r.Method == http.MethodPost {
			if rule["destination_ip"] != "any" || rule["src_firewall_group_id"] != "" {
				http.Error(w, "missing create defaults", 400)
				return
			}
			if ips, ok := rule["destination_ips"].([]any); !ok || len(ips) != 0 {
				http.Error(w, "destination_ips must be []", 400)
				return
			}
			rule["_id"] = api.newID()
			status = http.StatusCreated
		} else {
			// Replace the whole object: missing fields must not be rescued by the mock.
			rule["_id"] = segments[4]
		}
		api.portForwards[rule["_id"].(string)] = rule
		api.writeLegacyJSON(w, status, []map[string]any{rule})
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func portForwardTestConfig(api *mockUniFiAPI, ports, extra string) string {
	return siteLookupConfig(api.URL()) + fmt.Sprintf(`
resource "unifi_port_forward" "test" {
  site_id = data.unifi_site.main.id
  name = "test-forward"
  destination_port = %q
  forward_ip = "192.0.2.90"
  forward_port = %q
  %s
}
`, ports, ports, extra)
}

func TestAccResourcePortForward(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()

	resourceName := "unifi_port_forward.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *tfstate.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			if len(api.portForwards) != 0 {
				return fmt.Errorf("port forward survived destroy")
			}
			return nil
		},
		Steps: []resource.TestStep{{
			Config: portForwardTestConfig(api, "3074", ""),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "destination_port", "3074"),
				resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				resource.TestCheckResourceAttr(resourceName, "wan_interface", "wan"),
				resource.TestCheckResourceAttr(resourceName, "protocol", "tcp_udp"),
				resource.TestCheckResourceAttr(resourceName, "source", "any"),
				resource.TestCheckResourceAttr(resourceName, "logging_enabled", "false"),
			),
		}, {
			Config: portForwardTestConfig(api, "3000-3010", "enabled = false\nprotocol = \"udp\"\nsource = \"198.51.100.0/24\"\nwan_interface = \"wan2\"\nlogging_enabled = true"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "destination_port", "3000-3010"),
				resource.TestCheckResourceAttr(resourceName, "forward_port", "3000-3010"),
				resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				resource.TestCheckResourceAttr(resourceName, "source", "198.51.100.0/24"),
			),
		}, {
			ResourceName: resourceName, ImportState: true, ImportStateVerify: true,
			ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
		}, {
			Config: portForwardTestConfig(api, "80,443,8080", "protocol = \"tcp\"\nwan_interface = \"both\""),
			Check:  resource.TestCheckResourceAttr(resourceName, "destination_port", "80,443,8080"),
		}},
	})
}

func TestAccResourcePortForwardRecreatesDeletedRule(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	config := portForwardTestConfig(api, "3074", "")
	var originalID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config, ExpectNonEmptyPlan: true,
			Check: func(s *tfstate.State) error {
				originalID = s.RootModule().Resources["unifi_port_forward.test"].Primary.ID
				api.mu.Lock()
				defer api.mu.Unlock()
				delete(api.portForwards, originalID)
				return nil
			},
		}, {
			Config: config,
			Check: func(s *tfstate.State) error {
				if s.RootModule().Resources["unifi_port_forward.test"].Primary.ID == originalID {
					return fmt.Errorf("deleted rule was not recreated")
				}
				return nil
			},
		}},
	})
}

func TestAccDataSourcePortForward(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: portForwardTestConfig(api, "3074", "") + `
data "unifi_port_forward" "by_id" {
  site_id = data.unifi_site.main.id
  id = unifi_port_forward.test.id
}
data "unifi_port_forward" "by_name" {
  site_id = data.unifi_site.main.id
  name = unifi_port_forward.test.name
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.unifi_port_forward.by_id", "id", "unifi_port_forward.test", "id"),
				resource.TestCheckResourceAttrPair("data.unifi_port_forward.by_name", "id", "unifi_port_forward.test", "id"),
				resource.TestCheckResourceAttr("data.unifi_port_forward.by_name", "forward_ip", "192.0.2.90"),
			),
		}},
	})
}

func TestAccDataSourcePortForwardLookupErrors(t *testing.T) {
	for _, tc := range []struct{ name, selector, message string }{
		{"missing", `name = "absent"`, "Port forward not found"},
		{"neither", "", "Exactly one of"},
		{"both", "id = \"id\"\nname = \"name\"", "Exactly one of"},
		{"duplicate", `name = "duplicate"`, "Multiple port forwards matched"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := newMockUniFiAPI(t)
			defer api.Close()
			for _, id := range []string{"first", "second"} {
				api.portForwards[id] = map[string]any{"_id": id, "name": "duplicate"}
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    []resource.TestStep{{Config: siteLookupConfig(api.URL()) + "\ndata \"unifi_port_forward\" \"test\" {\nsite_id = data.unifi_site.main.id\n" + tc.selector + "\n}", ExpectError: regexp.MustCompile(tc.message)}},
			})
		})
	}
}

func TestAccResourcePortForwardRejectsInvalidEnums(t *testing.T) {
	for _, attribute := range []string{"protocol", "wan_interface"} {
		t.Run(attribute, func(t *testing.T) {
			api := newMockUniFiAPI(t)
			defer api.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      portForwardTestConfig(api, "3074", attribute+" = \"invalid\""),
					ExpectError: regexp.MustCompile("Invalid string value"),
				}},
			})
		})
	}
}

func TestAccResourcePortForwardRejectsInvalidSource(t *testing.T) {
	for _, source := range []string{"Any", "", "not-an-ip", "192.0.2.1/99", " any ", "2001:db8::1", "2001:db8::/64", "198.51.100.7/24"} {
		t.Run(source, func(t *testing.T) {
			api := newMockUniFiAPI(t)
			defer api.Close()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    []resource.TestStep{{Config: portForwardTestConfig(api, "3074", fmt.Sprintf("source = %q", source)), ExpectError: regexp.MustCompile("Invalid source")}},
			})
		})
	}
}

func TestAccResourcePortForwardPreservesControllerRestrictions(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	config := portForwardTestConfig(api, "3074", "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: func(s *tfstate.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				rule := api.portForwards[s.RootModule().Resources["unifi_port_forward.test"].Primary.ID]
				rule["destination_ip"] = "203.0.113.5"
				rule["destination_ips"] = []any{"203.0.113.5"}
				rule["src_firewall_group_id"] = "group-id"
				rule["src_limiting_enabled"] = true
				return nil
			},
		}, {
			ResourceName: "unifi_port_forward.test", ImportState: true, ImportStateVerify: true, ImportStateIdFunc: testImportCompositeID("unifi_port_forward.test", api.siteID),
		}, {
			Config: portForwardTestConfig(api, "3074", "enabled = false"),
			Check: func(s *tfstate.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				rule := api.portForwards[s.RootModule().Resources["unifi_port_forward.test"].Primary.ID]
				if rule["enabled"] != false || rule["destination_ip"] != "203.0.113.5" || rule["src_firewall_group_id"] != "group-id" || rule["src_limiting_enabled"] != true {
					return fmt.Errorf("lost controller restrictions: %#v", rule)
				}
				return nil
			},
		}},
	})
}

func TestAccResourcePortForwardRejectsInvalidPortsAndForwardIP(t *testing.T) {
	for name, values := range map[string][3]string{
		"destination_port": {"80;443", "192.0.2.90", "3074"},
		"forward_ip":       {"3074", "example.lan", "3074"},
		"forward_port":     {"3074", "192.0.2.90", "3010-3000"},
	} {
		t.Run(name, func(t *testing.T) {
			api := newMockUniFiAPI(t)
			defer api.Close()
			config := siteLookupConfig(api.URL()) + fmt.Sprintf(`
resource "unifi_port_forward" "test" {
  site_id = data.unifi_site.main.id
  name = "test-forward"
  destination_port = %q
  forward_ip = %q
  forward_port = %q
}
`, values[0], values[1], values[2])
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Invalid " + name)}},
			})
		})
	}
}

func TestAccResourcePortForwardRepairsDisabledSourceLimiting(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	config := portForwardTestConfig(api, "3074", `source = "198.51.100.0/24"`)
	ruleOf := func(s *tfstate.State) map[string]any {
		return api.portForwards[s.RootModule().Resources["unifi_port_forward.test"].Primary.ID]
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: func(s *tfstate.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				// Simulate an admin switching source limiting off in the UI.
				ruleOf(s)["src_limiting_enabled"] = false
				return nil
			},
			ExpectNonEmptyPlan: true,
		}, {
			// The refresh must report the rule as open and the re-apply must restore the restriction.
			RefreshState: true, ExpectNonEmptyPlan: true,
			Check: resource.TestCheckResourceAttr("unifi_port_forward.test", "source", "any"),
		}, {
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("unifi_port_forward.test", "source", "198.51.100.0/24"),
				func(s *tfstate.State) error {
					api.mu.Lock()
					defer api.mu.Unlock()
					if rule := ruleOf(s); rule["src_limiting_enabled"] != true || rule["src"] != "198.51.100.0/24" {
						return fmt.Errorf("source limiting was not repaired: %#v", rule)
					}
					return nil
				},
			),
		}},
	})
}

func TestAccResourcePortForwardRejectsSourceConflictingWithFirewallGroup(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: portForwardTestConfig(api, "3074", ""),
			Check: func(s *tfstate.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				rule := api.portForwards[s.RootModule().Resources["unifi_port_forward.test"].Primary.ID]
				rule["src_firewall_group_id"] = "group-id"
				rule["src_limiting_enabled"] = true
				return nil
			},
		}, {
			Config:      portForwardTestConfig(api, "3074", `source = "203.0.113.9"`),
			ExpectError: regexp.MustCompile(`(?s)source\s+firewall\s+group`),
		}},
	})
}

func TestAccResourceDHCPReservationBootstrapsNullClientTable(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	// The existing legacy mock serializes its nil result slice as data:null.
	clear(api.dhcpReservations["default"])
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: siteLookupConfig(api.URL()) + fmt.Sprintf(`
resource "unifi_dhcp_reservation" "test" {
  site_id = data.unifi_site.main.id
  mac_address = %q
  fixed_ip = "10.200.0.25"
}
`, api.existingAdoptedDeviceMAC),
			Check: resource.TestCheckResourceAttr("unifi_dhcp_reservation.test", "fixed_ip", "10.200.0.25"),
		}},
	})
}
