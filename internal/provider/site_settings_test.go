package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

// The settings as a current Network application stores them, including the
// fields the resources don't model (the SNMP credentials among them).
func newMockSiteSettings() map[string]map[string]any {
	return map[string]map[string]any{
		"rsyslogd": {
			"_id": "setting-rsyslogd", "key": "rsyslogd", "site_id": "legacy-site",
			"enabled": false, "ip": "", "port": "514", "contents": []any{"device"},
			"log_all_contents": false, "this_controller": false, "this_controller_encrypted_only": false,
			"netconsole_enabled": false, "debug": false,
		},
		"ntp": {
			"_id": "setting-ntp", "key": "ntp", "site_id": "legacy-site", "setting_preference": "auto",
			"ntp_server_1": "0.ubnt.pool.ntp.org", "ntp_server_2": "1.ubnt.pool.ntp.org", "ntp_server_3": "", "ntp_server_4": "",
		},
		"snmp": {
			"_id": "setting-snmp", "key": "snmp", "site_id": "legacy-site", "enabled": false, "enabledV3": false,
			"community": "fake-community", "username": "fake-user", "x_password": "fake-password",
		},
		"mgmt": {
			"_id": "setting-mgmt", "key": "mgmt", "site_id": "legacy-site",
			"x_ssh_enabled": true, "x_ssh_auth_password_enabled": true, "x_ssh_bind_wildcard": false,
			"auto_upgrade": true, "auto_upgrade_hour": 3, "advanced_feature_enabled": true,
			"x_ssh_username": "fake-user", "x_ssh_password": "fake-password", "x_ssh_sha512passwd": "fake-hash",
			"x_ssh_keys":  []any{map[string]any{"name": "fake-key", "key": "fake-public-key"}},
			"x_api_token": "fake-api-token", "x_mgmt_key": "fake-mgmt-key",
		},
	}
}

// The mock replaces the whole document on PUT, stricter than Network 10.6,
// which merges, so a client that drops unmodelled fields fails the checks below.
// A setting the site never saved reads as one empty document, without a key.
func (api *mockUniFiAPI) handleSiteSetting(w http.ResponseWriter, r *http.Request, segments []string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.siteSettingsBroken {
		http.Error(w, `{"meta":{"rc":"error","msg":"api.err.Internal"},"data":[]}`, http.StatusInternalServerError)
		return
	}
	if segments[1] != "default" {
		api.writeLegacyJSON(w, http.StatusOK, []any{})
		return
	}
	setting, exists := api.siteSettings[segments[4]]
	switch {
	case segments[2] == "get" && r.Method == http.MethodGet:
		if !exists {
			setting = map[string]any{}
		}
		api.writeLegacyJSON(w, http.StatusOK, []map[string]any{setting})
	case segments[2] == "set" && r.Method == http.MethodPut:
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !exists {
			if body["key"] != segments[4] {
				http.Error(w, "missing or changed field: key", http.StatusBadRequest)
				return
			}
			body["_id"] = "created-" + segments[4]
			api.siteSettings[segments[4]] = body
			api.siteSettingPuts++
			api.writeLegacyJSON(w, http.StatusOK, []any{})
			return
		}
		for _, field := range []string{"_id", "key", "site_id"} {
			if body[field] != setting[field] {
				http.Error(w, "missing or changed field: "+field, http.StatusBadRequest)
				return
			}
		}
		api.siteSettings[segments[4]] = body
		api.siteSettingPuts++
		api.writeLegacyJSON(w, http.StatusOK, []any{})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func checkSiteSettingField(api *mockUniFiAPI, key, field string, want any) resource.TestCheckFunc {
	return func(_ *tfstate.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		got := api.siteSettings[key][field]
		if fmt.Sprint(got) != fmt.Sprint(want) {
			return fmt.Errorf("setting %s field %s = %#v, want %#v", key, field, got, want)
		}
		return nil
	}
}

func siteSettingImportStep(resourceName string) resource.TestStep {
	return resource.TestStep{
		ResourceName:      resourceName,
		ImportState:       true,
		ImportStateVerify: true,
		ImportStateIdFunc: func(state *tfstate.State) (string, error) {
			return state.RootModule().Resources[resourceName].Primary.Attributes["site_id"], nil
		},
	}
}

func siteSettingConfig(api *mockUniFiAPI, resourceType, body string) string {
	return siteLookupConfig(api.URL()) + fmt.Sprintf(`
resource "%s" "test" {
  site_id = data.unifi_site.main.id
%s
}
`, resourceType, body)
}

func TestAccResourceRemoteSyslog(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resourceName := "unifi_setting_remote_syslog.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteSettingConfig(api, "unifi_setting_remote_syslog", `
  enabled  = true
  ip       = "192.0.2.10"
  port     = 514
  contents = ["device", "client", "firewall_default_policy"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "id", "data.unifi_site.main", "id"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "port", "514"),
					resource.TestCheckResourceAttr(resourceName, "contents.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "log_all_contents", "false"),
					// The port stays a string, as the controller stores it.
					checkSiteSettingField(api, "rsyslogd", "port", "514"),
					checkSiteSettingField(api, "rsyslogd", "ip", "192.0.2.10"),
					// Unmodelled fields survive the whole-document PUT.
					checkSiteSettingField(api, "rsyslogd", "debug", false),
				),
			},
			siteSettingImportStep(resourceName),
			{
				// Only enabled: every other field keeps the controller's value.
				Config: siteSettingConfig(api, "unifi_setting_remote_syslog", `  enabled = false`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "ip", "192.0.2.10"),
					resource.TestCheckResourceAttr(resourceName, "contents.#", "3"),
					checkSiteSettingField(api, "rsyslogd", "enabled", false),
				),
			},
			{
				Config: siteSettingConfig(api, "unifi_setting_remote_syslog", `
  enabled            = true
  port               = 1514
  log_all_contents   = true
  netconsole_enabled = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "port", "1514"),
					resource.TestCheckResourceAttr(resourceName, "log_all_contents", "true"),
					checkSiteSettingField(api, "rsyslogd", "port", "1514"),
					checkSiteSettingField(api, "rsyslogd", "netconsole_enabled", true),
				),
			},
		},
		// Destroy leaves logging as it is.
		CheckDestroy: func(_ *tfstate.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			if api.siteSettings["rsyslogd"]["enabled"] != true {
				return fmt.Errorf("remote syslog changed on destroy")
			}
			return nil
		},
	})
}

// A controller that stores the port as a number gets a number back.
func TestAccResourceRemoteSyslogNumericPort(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	api.siteSettings["rsyslogd"]["port"] = 514
	api.siteSettings["rsyslogd"]["ip"] = "192.0.2.10"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: siteSettingConfig(api, "unifi_setting_remote_syslog", "  enabled = true\n  port = 5514"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("unifi_setting_remote_syslog.test", "port", "5514"),
				checkSiteSettingField(api, "rsyslogd", "port", float64(5514)),
			),
		}},
	})
}

func TestResourceRemoteSyslogValidation(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	cases := map[string]struct{ body, err string }{
		"ipv6 server":     {"  enabled = true\n  ip = \"2001:db8::1\"", "needs an IPv4 address"},
		"host name":       {"  enabled = true\n  ip = \"syslog.example\"", "needs an IPv4 address"},
		"port zero":       {"  enabled = true\n  ip = \"192.0.2.1\"\n  port = 0", "between 1 and 65535"},
		"port too high":   {"  enabled = true\n  ip = \"192.0.2.1\"\n  port = 70000", "between 1 and 65535"},
		"missing enabled": {"  ip = \"192.0.2.1\"", "enabled"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      siteSettingConfig(api, "unifi_setting_remote_syslog", testCase.body),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(testCase.err),
				}},
			})
		})
	}
	if api.siteSettingPuts != 0 {
		t.Fatalf("validation errors must stop before any write, got %d PUTs", api.siteSettingPuts)
	}
}

// Enabling without a server, when the controller has none, fails before the write.
func TestAccResourceRemoteSyslogNeedsServer(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      siteSettingConfig(api, "unifi_setting_remote_syslog", "  enabled = true"),
			ExpectError: regexp.MustCompile("the controller has no syslog server yet"),
		}},
	})
	if api.siteSettingPuts != 0 {
		t.Fatalf("no write expected, got %d PUTs", api.siteSettingPuts)
	}
}

func TestAccResourceNTP(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resourceName := "unifi_setting_ntp.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Taking over in auto mode keeps the controller's servers.
				Config: siteSettingConfig(api, "unifi_setting_ntp", `  mode = "auto"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mode", "auto"),
					resource.TestCheckResourceAttr(resourceName, "servers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "servers.1", "1.ubnt.pool.ntp.org"),
				),
			},
			siteSettingImportStep(resourceName),
			{
				Config: siteSettingConfig(api, "unifi_setting_ntp", `
  mode    = "manual"
  servers = ["192.0.2.1", "time.example"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "servers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "servers.0", "192.0.2.1"),
					checkSiteSettingField(api, "ntp", "setting_preference", "manual"),
					checkSiteSettingField(api, "ntp", "ntp_server_2", "time.example"),
					// Unused slots are cleared, not left with the old pool servers.
					checkSiteSettingField(api, "ntp", "ntp_server_3", ""),
				),
			},
			{
				Config: siteSettingConfig(api, "unifi_setting_ntp", `
  mode    = "manual"
  servers = ["192.0.2.1"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "servers.#", "1"),
					checkSiteSettingField(api, "ntp", "ntp_server_2", ""),
				),
			},
		},
	})
}

// manual without servers is fine while the controller already has some, and
// an error when it has none.
func TestAccResourceNTPManualNeedsServers(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	for slot := 1; slot <= 4; slot++ {
		api.siteSettings["ntp"][fmt.Sprintf("ntp_server_%d", slot)] = ""
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      siteSettingConfig(api, "unifi_setting_ntp", `  mode = "manual"`),
			ExpectError: regexp.MustCompile("the controller has none"),
		}},
	})
	if api.siteSettingPuts != 0 {
		t.Fatalf("no write expected, got %d PUTs", api.siteSettingPuts)
	}
}

func TestResourceNTPValidation(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	cases := map[string]struct{ body, err string }{
		"five servers":  {`  mode = "manual"` + "\n" + `  servers = ["a", "b", "c", "d", "e"]`, "at most 4 servers"},
		"empty server":  {`  mode = "manual"` + "\n" + `  servers = ["a", ""]`, "Remove the entry"},
		"manual, empty": {`  mode = "manual"` + "\n" + `  servers = []`, "needs at least one server"},
		"bad mode":      {`  mode = "ntp"`, "Must be one of: auto, manual"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      siteSettingConfig(api, "unifi_setting_ntp", testCase.body),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(testCase.err),
				}},
			})
		})
	}
	if api.siteSettingPuts != 0 {
		t.Fatalf("validation errors must stop before any write, got %d PUTs", api.siteSettingPuts)
	}
}

func TestAccResourceSNMP(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resourceName := "unifi_setting_snmp.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: siteSettingConfig(api, "unifi_setting_snmp", `  enabled = false`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "enabled_v3", "false"),
					// The credentials are neither in state nor touched.
					resource.TestCheckNoResourceAttr(resourceName, "community"),
					checkSiteSettingField(api, "snmp", "community", "fake-community"),
					checkSiteSettingField(api, "snmp", "x_password", "fake-password"),
				),
			},
			siteSettingImportStep(resourceName),
			{
				Config: siteSettingConfig(api, "unifi_setting_snmp", "  enabled = true\n  enabled_v3 = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled_v3", "true"),
					checkSiteSettingField(api, "snmp", "enabled", true),
					checkSiteSettingField(api, "snmp", "enabledV3", true),
					checkSiteSettingField(api, "snmp", "x_password", "fake-password"),
				),
			},
		},
	})
}

// A setting the controller doesn't have can't be taken over, and one that
// disappears is dropped from state.
func TestAccResourceSiteSettingMissing(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	delete(api.siteSettings, "snmp")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: siteSettingConfig(api, "unifi_setting_snmp", `  enabled = false`),
			Check:  checkSiteSettingField(api, "snmp", "enabled", false),
		}},
	})
	if api.siteSettingPuts != 1 {
		t.Fatalf("one create expected, got %d PUTs", api.siteSettingPuts)
	}
}

func TestAccResourceRemoteSyslogCreatesMissing(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	delete(api.siteSettings, "rsyslogd")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      siteSettingConfig(api, "unifi_setting_remote_syslog", "  enabled = true"),
				ExpectError: regexp.MustCompile("the controller has no syslog server yet"),
			},
			{
				Config: siteSettingConfig(api, "unifi_setting_remote_syslog", "  enabled = true\n  ip = \"192.0.2.10\"\n  port = 514"),
				Check:  checkSiteSettingField(api, "rsyslogd", "port", "514"),
			},
		},
	})
	if api.siteSettingPuts != 1 {
		t.Fatalf("one create expected, got %d PUTs", api.siteSettingPuts)
	}
}

func TestAccResourceNTPManualMissingNeedsServers(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	delete(api.siteSettings, "ntp")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      siteSettingConfig(api, "unifi_setting_ntp", `  mode = "manual"`),
			ExpectError: regexp.MustCompile("the controller has none"),
		}},
	})
	if api.siteSettingPuts != 0 {
		t.Fatalf("no write expected, got %d PUTs", api.siteSettingPuts)
	}
}

func TestAccResourceSiteSettingRemovedFromController(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: siteSettingConfig(api, "unifi_setting_ntp", `  mode = "auto"`)},
			{
				PreConfig: func() {
					api.mu.Lock()
					delete(api.siteSettings, "ntp")
					api.mu.Unlock()
				},
				Config:             siteSettingConfig(api, "unifi_setting_ntp", `  mode = "auto"`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Fields of an unexpected type are reported, not silently read as zero.
func TestFlattenSiteSettingsBadTypes(t *testing.T) {
	ctx := t.Context()
	siteID := types.StringValue("site")
	bad := client.SiteSetting{
		"enabled": json.RawMessage(`"yes"`), "ip": json.RawMessage(`5`), "port": json.RawMessage(`"x"`),
		"contents": json.RawMessage(`"device"`), "setting_preference": json.RawMessage(`1`), "enabledV3": json.RawMessage(`[]`),
	}
	var diags diag.Diagnostics
	flattenRemoteSyslog(ctx, siteID, bad, &diags)
	if diags.ErrorsCount() < 4 {
		t.Fatalf("expected an error per bad field, got %v", diags)
	}
	diags = nil
	flattenNTP(ctx, siteID, bad, &diags)
	flattenSNMP(siteID, bad, &diags)
	if diags.ErrorsCount() < 3 {
		t.Fatalf("expected errors for setting_preference, enabled and enabledV3, got %v", diags)
	}
	if _, err := parseRemoteSyslogPort(json.RawMessage(`true`)); err == nil {
		t.Fatal("a boolean port must fail")
	}
	// A missing boolean reads as false, a missing port as null.
	diags = nil
	model := flattenRemoteSyslog(ctx, siteID, client.SiteSetting{}, &diags)
	if diags.HasError() || model.Enabled.ValueBool() || !model.Port.IsNull() || len(model.Contents.Elements()) != 0 {
		t.Fatalf("unexpected model for an empty setting: %+v, %v", model, diags)
	}
}

func TestSiteSettingImportNeedsSiteID(t *testing.T) {
	for _, r := range []frameworkresource.ResourceWithImportState{&remoteSyslogResource{}, &ntpResource{}, &snmpResource{}, &deviceManagementResource{}} {
		resp := &frameworkresource.ImportStateResponse{}
		r.ImportState(t.Context(), frameworkresource.ImportStateRequest{ID: ""}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatalf("%T: an empty import ID must fail", r)
		}
	}
}

// Any controller error other than "not found" fails the read instead of
// dropping the resource from state.
func TestAccResourceSiteSettingReadError(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	for _, resourceType := range []string{"unifi_setting_remote_syslog", "unifi_setting_ntp", "unifi_setting_snmp", "unifi_setting_device_management"} {
		t.Run(resourceType, func(t *testing.T) {
			body := map[string]string{"unifi_setting_remote_syslog": "  enabled = false", "unifi_setting_ntp": `  mode = "auto"`, "unifi_setting_snmp": "  enabled = false", "unifi_setting_device_management": "  ssh_enabled = true"}[resourceType]
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: siteSettingConfig(api, resourceType, body)},
					{
						PreConfig: func() {
							api.mu.Lock()
							api.siteSettingsBroken = true
							api.mu.Unlock()
						},
						Config:      siteSettingConfig(api, resourceType, body),
						ExpectError: regexp.MustCompile("Unable to read setting"),
					},
				},
				CheckDestroy: func(_ *tfstate.State) error {
					api.mu.Lock()
					api.siteSettingsBroken = false
					api.mu.Unlock()
					return nil
				},
			})
		})
	}
}

func TestAccResourceDeviceManagement(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resourceName := "unifi_setting_device_management.test"
	secrets := map[string]any{"x_ssh_username": "fake-user", "x_ssh_password": "fake-password", "x_ssh_sha512passwd": "fake-hash", "x_api_token": "fake-api-token", "x_mgmt_key": "fake-mgmt-key"}
	checkSecretsKept := func(_ *tfstate.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		for field, want := range secrets {
			if api.siteSettings["mgmt"][field] != want {
				return fmt.Errorf("mgmt field %s = %#v, want %#v", field, api.siteSettings["mgmt"][field], want)
			}
		}
		if keys, ok := api.siteSettings["mgmt"]["x_ssh_keys"].([]any); !ok || len(keys) != 1 {
			return fmt.Errorf("SSH keys not preserved: %#v", api.siteSettings["mgmt"]["x_ssh_keys"])
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Taking over keeps every unset field as the controller has it.
				Config: siteSettingConfig(api, "unifi_setting_device_management", "  ssh_enabled = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ssh_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "ssh_password_auth_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(resourceName, "auto_upgrade_hour", "3"),
					// No credential reaches state.
					resource.TestCheckNoResourceAttr(resourceName, "x_ssh_password"),
					resource.TestCheckNoResourceAttr(resourceName, "ssh_keys.#"),
					checkSecretsKept,
				),
			},
			siteSettingImportStep(resourceName),
			{
				Config: siteSettingConfig(api, "unifi_setting_device_management", `
  ssh_enabled               = true
  ssh_password_auth_enabled = false
  auto_upgrade_hour         = 4
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ssh_password_auth_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "auto_upgrade_hour", "4"),
					checkSiteSettingField(api, "mgmt", "x_ssh_auth_password_enabled", false),
					checkSiteSettingField(api, "mgmt", "auto_upgrade_hour", float64(4)),
					checkSiteSettingField(api, "mgmt", "x_ssh_bind_wildcard", false),
					checkSecretsKept,
				),
			},
			{
				Config: siteSettingConfig(api, "unifi_setting_device_management", "  ssh_enabled = false\n  auto_upgrade = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkSiteSettingField(api, "mgmt", "x_ssh_enabled", false),
					checkSiteSettingField(api, "mgmt", "auto_upgrade", false),
					checkSecretsKept,
				),
			},
		},
		// Destroy leaves device access as it is.
		CheckDestroy: func(_ *tfstate.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			if api.siteSettings["mgmt"]["x_ssh_enabled"] != false {
				return fmt.Errorf("device access changed on destroy")
			}
			return nil
		},
	})
}

func TestResourceDeviceManagementValidation(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	for name, body := range map[string]string{"hour 24": "  ssh_enabled = true\n  auto_upgrade_hour = 24", "negative hour": "  ssh_enabled = true\n  auto_upgrade_hour = -1"} {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      siteSettingConfig(api, "unifi_setting_device_management", body),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile("between 0 and 23"),
				}},
			})
		})
	}
	if api.siteSettingPuts != 0 {
		t.Fatalf("validation errors must stop before any write, got %d PUTs", api.siteSettingPuts)
	}
}

func TestFlattenDeviceManagementBadTypes(t *testing.T) {
	var diags diag.Diagnostics
	flattenDeviceManagement(types.StringValue("site"), client.SiteSetting{"x_ssh_enabled": json.RawMessage(`1`), "auto_upgrade_hour": json.RawMessage(`"3"`)}, &diags)
	if diags.ErrorsCount() != 2 {
		t.Fatalf("expected two errors, got %v", diags)
	}
	diags = nil
	model := flattenDeviceManagement(types.StringValue("site"), client.SiteSetting{}, &diags)
	if diags.HasError() || !model.AutoUpgradeHour.IsNull() || model.SSHEnabled.ValueBool() {
		t.Fatalf("unexpected model for an empty setting: %+v, %v", model, diags)
	}
}

func TestAccResourceDeviceManagementMissing(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: siteSettingConfig(api, "unifi_setting_device_management", "  ssh_enabled = true")},
			{
				// Removed on the controller: dropped from state, so the next plan takes it over again.
				PreConfig: func() {
					api.mu.Lock()
					delete(api.siteSettings, "mgmt")
					api.mu.Unlock()
				},
				Config:             siteSettingConfig(api, "unifi_setting_device_management", "  ssh_enabled = true"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// The next apply creates it again.
				Config: siteSettingConfig(api, "unifi_setting_device_management", "  ssh_enabled = true"),
				Check:  checkSiteSettingField(api, "mgmt", "x_ssh_enabled", true),
			},
		},
	})
}
