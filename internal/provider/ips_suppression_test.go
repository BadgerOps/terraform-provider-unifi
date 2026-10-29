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

// The mock replaces the whole setting on PUT, as the controller does, so a
// client that forgets to send back the whitelist or unmodelled fields fails.
func (api *mockUniFiAPI) handleIPSSuppression(w http.ResponseWriter, r *http.Request, segments []string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if segments[1] != "default" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch {
	case segments[2] == "get" && r.Method == http.MethodGet:
		api.writeLegacyJSON(w, http.StatusOK, []map[string]any{api.ipsSuppression})
	case segments[2] == "set" && r.Method == http.MethodPut:
		var setting map[string]any
		if err := json.NewDecoder(r.Body).Decode(&setting); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, key := range []string{"_id", "key", "site_id", "whitelist"} {
			if _, ok := setting[key]; !ok {
				http.Error(w, "missing field: "+key, http.StatusBadRequest)
				return
			}
		}
		alerts, ok := setting["alerts"].([]any)
		if !ok {
			http.Error(w, "alerts must be a list", http.StatusBadRequest)
			return
		}
		for _, raw := range alerts {
			alert := raw.(map[string]any)
			for _, key := range []string{"category", "signature", "type"} {
				if _, ok := alert[key].(string); !ok {
					http.Error(w, "missing alert string field: "+key, http.StatusBadRequest)
					return
				}
			}
			for _, key := range []string{"gid", "id"} {
				if _, ok := alert[key].(float64); !ok {
					http.Error(w, "missing alert number field: "+key, http.StatusBadRequest)
					return
				}
			}
			tracking, present := alert["tracking"]
			if alert["type"] == "track" {
				if list, ok := tracking.([]any); !ok || len(list) == 0 {
					http.Error(w, "track needs a tracking list", http.StatusBadRequest)
					return
				}
			} else if present {
				http.Error(w, "type all is stored without tracking", http.StatusBadRequest)
				return
			}
		}
		api.ipsSuppression = setting
		api.ipsSuppressionPuts++
		// Acknowledge without the document, which the client must read back.
		api.writeLegacyJSON(w, http.StatusOK, []any{})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func ipsSuppressionTestConfig(api *mockUniFiAPI, alerts string) string {
	return siteLookupConfig(api.URL()) + fmt.Sprintf(`
resource "unifi_setting_ips_suppression" "test" {
  site_id = data.unifi_site.main.id
  alerts  = %s
}
`, alerts)
}

const ipsSuppressionTwoAlerts = `[
    {
      signature_id = 2016642
      category     = "emerging-webserver"
      signature    = "ET WEB_SERVER Possible Perl Shell in HTTP POST"
      type         = "track"
      tracking     = [{ mode = "ip", value = "192.0.2.30", direction = "dest" }]
    },
    {
      signature_id = 2053457
      gid          = 1
      category     = "emerging-webserver"
      type         = "track"
      tracking = [
        { mode = "subnet", value = "198.51.100.0/24", direction = "src" },
        { mode = "ip", value = "192.0.2.30", direction = "dest" },
      ]
    },
  ]`

func TestAccResourceIPSSuppression(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	// An alert made in the UI: its description is kept when the config omits it.
	api.ipsSuppression["alerts"] = []any{map[string]any{"gid": 1, "id": 2053457, "category": "emerging-webserver", "signature": "ET WEB_SERVER Possible SQL Injection DELETE FROM", "type": "all"}}

	resourceName := "unifi_setting_ips_suppression.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: ipsSuppressionTestConfig(api, ipsSuppressionTwoAlerts),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "id", "data.unifi_site.main", "id"),
					resource.TestCheckResourceAttr(resourceName, "alerts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "alerts.0.gid", "1"),
					resource.TestCheckResourceAttr(resourceName, "alerts.0.tracking.0.direction", "dest"),
					resource.TestCheckResourceAttr(resourceName, "alerts.1.signature", "ET WEB_SERVER Possible SQL Injection DELETE FROM"),
					resource.TestCheckResourceAttr(resourceName, "alerts.1.tracking.#", "2"),

					checkIPSSuppressionWhitelistKept(api),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *tfstate.State) (string, error) {
					return state.RootModule().Resources[resourceName].Primary.Attributes["site_id"], nil
				},
			},
			{
				// One alert suppressed everywhere; the other is removed on apply.
				Config: ipsSuppressionTestConfig(api, `[{ signature_id = 2016642, category = "emerging-webserver", type = "all" }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "alerts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "alerts.0.type", "all"),
					resource.TestCheckNoResourceAttr(resourceName, "alerts.0.tracking.#"),
					resource.TestCheckResourceAttr(resourceName, "alerts.0.signature", "ET WEB_SERVER Possible Perl Shell in HTTP POST"),
					checkIPSSuppressionWhitelistKept(api),
				),
			},
			{
				Config: ipsSuppressionTestConfig(api, `[]`),
				Check:  resource.TestCheckResourceAttr(resourceName, "alerts.#", "0"),
			},
		},
		// Destroy leaves the controller untouched.
		CheckDestroy: func(_ *tfstate.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			if api.ipsSuppression["_id"] != "setting-ips-suppression" {
				return fmt.Errorf("setting replaced on destroy")
			}
			return nil
		},
	})
}

func checkIPSSuppressionWhitelistKept(api *mockUniFiAPI) resource.TestCheckFunc {
	return func(_ *tfstate.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		whitelist, ok := api.ipsSuppression["whitelist"].([]any)
		if !ok || len(whitelist) != 1 {
			return fmt.Errorf("whitelist not preserved: %#v", api.ipsSuppression["whitelist"])
		}
		if api.ipsSuppression["site_id"] != "legacy-site" {
			return fmt.Errorf("unmodelled field site_id not preserved")
		}
		return nil
	}
}

func TestResourceIPSSuppressionValidation(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	cases := map[string]struct{ alerts, err string }{
		"track without tracking": {`[{ signature_id = 1, category = "c", type = "track" }]`, "needs at least one tracking entry"},
		"all with tracking":      {`[{ signature_id = 1, category = "c", type = "all", tracking = [{ mode = "ip", value = "192.0.2.1", direction = "src" }] }]`, "remove tracking"},
		"bad type":               {`[{ signature_id = 1, category = "c", type = "never" }]`, "Must be one of: all, track"},
		"ip mode with prefix":    {`[{ signature_id = 1, category = "c", type = "track", tracking = [{ mode = "ip", value = "192.0.2.0/24", direction = "src" }] }]`, "an IPv4 address"},
		"subnet with host bits":  {`[{ signature_id = 1, category = "c", type = "track", tracking = [{ mode = "subnet", value = "192.0.2.1/24", direction = "src" }] }]`, "no host bits set"},
		"ipv6 address":           {`[{ signature_id = 1, category = "c", type = "track", tracking = [{ mode = "ip", value = "2001:db8::1", direction = "src" }] }]`, "an IPv4 address"},
		"bad direction":          {`[{ signature_id = 1, category = "c", type = "track", tracking = [{ mode = "ip", value = "192.0.2.1", direction = "in" }] }]`, "Must be one of: src, dest, both"},
		"duplicate signature":    {`[{ signature_id = 7, category = "c", type = "all" }, { signature_id = 7, gid = 1, category = "c", type = "all" }]`, "already suppressed by alerts\\[0\\]"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      ipsSuppressionTestConfig(api, testCase.alerts),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(testCase.err),
				}},
			})
		})
	}
	if api.ipsSuppressionPuts != 0 {
		t.Fatalf("validation errors must stop before any write, got %d PUTs", api.ipsSuppressionPuts)
	}
}
