package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"
)

// A WAN SLA as Network 10 stores it, with a field the resource doesn't model
// (monitor_down_threshold), so a write that drops it fails the checks below.
func mockWANSLA() map[string]any {
	return map[string]any{
		"_id": "sla-meo", "name": "ping 1.1.1.1", "monitor_health_type": "ALL", "monitor_down_threshold": float64(2),
		"monitors": []any{map[string]any{
			"monitor_id": float64(1), "type": "ICMP", "target": "1.1.1.1", "interval": float64(1), "time_period": float64(60),
			"alert": map[string]any{"loss_threshold": float64(25), "latency_threshold": float64(500), "threshold_policy": "any"},
		}},
	}
}

// handleV2HTTP serves /proxy/network/v2/api/site/<site>/wan-slas[/<id>]. PUT
// replaces the whole document, so a client must send back what it doesn't model.
func (api *mockUniFiAPI) handleV2HTTP(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/proxy/network/v2/api"), "/"), "/")
	if len(segments) < 3 || segments[0] != "site" || segments[1] != "default" || segments[2] != "wan-slas" {
		writeV2Error(w, http.StatusNotFound, "api.err.NotFound")
		return
	}
	api.mu.Lock()
	defer api.mu.Unlock()
	if r.Method == api.wanSLAFailMethod {
		writeV2Error(w, http.StatusInternalServerError, "api.err.Internal")
		return
	}
	if len(segments) == 3 {
		switch r.Method {
		case http.MethodGet:
			writeV2JSON(w, http.StatusOK, api.wanSLAs)
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeV2Error(w, http.StatusBadRequest, err.Error())
				return
			}
			if _, has := body["_id"]; has {
				writeV2Error(w, http.StatusBadRequest, "api.err.IdNotAllowed")
				return
			}
			api.wanSLAWrites++
			body["_id"] = fmt.Sprintf("sla-%d", api.nextID)
			api.nextID++
			api.wanSLAs = append(api.wanSLAs, body)
			if api.wanSLACreateOmitsID {
				writeV2JSON(w, http.StatusOK, map[string]any{})
				return
			}
			writeV2JSON(w, http.StatusCreated, body)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	id := segments[3]
	for i, sla := range api.wanSLAs {
		if sla["_id"] != id {
			continue
		}
		switch r.Method {
		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeV2Error(w, http.StatusBadRequest, err.Error())
				return
			}
			api.wanSLAWrites++
			body["_id"] = id
			api.wanSLAs[i] = body
			writeV2JSON(w, http.StatusOK, body)
		case http.MethodDelete:
			api.wanSLAWrites++
			api.wanSLAs = append(api.wanSLAs[:i], api.wanSLAs[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	writeV2Error(w, http.StatusNotFound, "api.err.NotFound")
}

func writeV2JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeV2Error(w http.ResponseWriter, status int, code string) {
	writeV2JSON(w, status, map[string]any{"code": code, "message": code})
}

func (api *mockUniFiAPI) wanSLA(id string) map[string]any {
	api.mu.Lock()
	defer api.mu.Unlock()
	for _, sla := range api.wanSLAs {
		if sla["_id"] == id {
			return sla
		}
	}
	return nil
}

func checkWANSLA(api *mockUniFiAPI, id string, check func(map[string]any) error) resource.TestCheckFunc {
	return func(_ *tfstate.State) error {
		sla := api.wanSLA(id)
		if sla == nil {
			return fmt.Errorf("WAN SLA %s not on the controller", id)
		}
		return check(sla)
	}
}

func firstMonitorAlert(sla map[string]any) map[string]any {
	return sla["monitors"].([]any)[0].(map[string]any)["alert"].(map[string]any)
}

func wanSLAConfig(api *mockUniFiAPI, name, body string) string {
	return siteLookupConfig(api.URL()) + fmt.Sprintf(`
resource "unifi_wan_sla" %q {
  site_id = data.unifi_site.main.id
%s
}
`, name, body)
}

const meoSLABody = `  name                = "ping 1.1.1.1"
  monitor_health_type = "ALL"
  monitors = [{
    type              = "ICMP"
    target            = "1.1.1.1"
    interval          = 1
    time_period       = 60
    loss_threshold    = %d
    latency_threshold = 500
    threshold_policy  = "any"
  }]`

// Import an SLA the UI made, show no diff, change one threshold, keep what
// isn't modelled.
func TestAccResourceWANSLAImportAndUpdate(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	api.wanSLAs = []map[string]any{mockWANSLA()}
	resourceName := "unifi_wan_sla.meo"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 25)),
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      api.siteID + "/sla-meo",
				ImportStatePersist: true,
			},
			{
				Config:   wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 25)),
				PlanOnly: true,
			},
			{
				Config: wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 30)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "sla-meo"),
					resource.TestCheckResourceAttr(resourceName, "monitors.0.loss_threshold", "30"),
					resource.TestCheckNoResourceAttr(resourceName, "monitors.0.jitter_threshold"),
					checkWANSLA(api, "sla-meo", func(sla map[string]any) error {
						if got := firstMonitorAlert(sla)["loss_threshold"]; got != float64(30) {
							return fmt.Errorf("loss_threshold = %v, want 30", got)
						}
						if _, has := firstMonitorAlert(sla)["jitter_threshold"]; has {
							return fmt.Errorf("an unset jitter_threshold must stay absent, got %v", firstMonitorAlert(sla)["jitter_threshold"])
						}
						if sla["monitor_down_threshold"] != float64(2) {
							return fmt.Errorf("unmodelled monitor_down_threshold lost: %v", sla["monitor_down_threshold"])
						}
						return nil
					}),
				),
			},
		},
	})
}

// Create a DNS SLA without a health type (the controller's stays), then
// delete it on destroy.
func TestAccResourceWANSLACreate(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	resourceName := "unifi_wan_sla.dns"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*tfstate.State) error {
			if n := len(api.wanSLAs); n != 0 {
				return fmt.Errorf("destroy must delete the SLA, %d left", n)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: wanSLAConfig(api, "dns", `
  name = "DNS probes"
  monitors = [{
    type             = "DNS"
    target           = "192.0.2.53"
    domain           = "example.com"
    interval         = 5
    time_period      = 60
    jitter_threshold = 50
    threshold_policy = "all"
  }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "monitors.0.domain", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "monitors.0.jitter_threshold", "50"),
					resource.TestCheckNoResourceAttr(resourceName, "monitors.0.loss_threshold"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testImportCompositeID(resourceName, api.siteID),
			},
		},
	})
}

// An answer without the new ID: the SLA is found by name.
func TestAccResourceWANSLACreateFindsByName(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	api.wanSLAs = []map[string]any{mockWANSLA()}
	api.wanSLACreateOmitsID = true
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: wanSLAConfig(api, "lte", `
  name                = "LTE ping"
  monitor_health_type = "ANY"
  monitors = [{
    type             = "ICMP"
    target           = "1.1.1.1"
    interval         = 5
    time_period      = 60
    loss_threshold   = 30
    threshold_policy = "any"
  }]`),
			Check: resource.ComposeAggregateTestCheckFunc(
				func(state *tfstate.State) error {
					id := state.RootModule().Resources["unifi_wan_sla.lte"].Primary.Attributes["id"]
					if sla := api.wanSLA(id); sla == nil || sla["name"] != "LTE ping" {
						return fmt.Errorf("id %s isn't the created SLA: %v", id, sla)
					}
					return nil
				},
				resource.TestCheckResourceAttr("unifi_wan_sla.lte", "monitor_health_type", "ANY"),
			),
		}},
	})
}

// An SLA deleted in the UI leaves state and is planned again.
func TestAccResourceWANSLARemovedFromController(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	api.wanSLAs = []map[string]any{mockWANSLA()}
	config := wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 25))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, ResourceName: "unifi_wan_sla.meo", ImportState: true, ImportStateId: api.siteID + "/sla-meo", ImportStatePersist: true},
			{
				PreConfig:          func() { api.mu.Lock(); api.wanSLAs = nil; api.mu.Unlock() },
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestResourceWANSLAValidation(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	monitor := func(fields string) string {
		return "  name = \"x\"\n  monitors = [{\n    type = \"ICMP\"\n    threshold_policy = \"any\"\n" + fields + "\n  }]"
	}
	cases := map[string]struct{ body, err string }{
		"no monitors":        {"  name = \"x\"\n  monitors = []", "at least one monitor"},
		"majority of one":    {strings.Replace(monitor("    target = \"a\"\n    interval = 1\n    time_period = 60\n    loss_threshold = 1"), "  name = \"x\"", "  name = \"x\"\n  monitor_health_type = \"MAJORITY\"", 1), "three or more"},
		"bad health type":    {strings.Replace(monitor("    target = \"a\"\n    interval = 1\n    time_period = 60\n    loss_threshold = 1"), "  name = \"x\"", "  name = \"x\"\n  monitor_health_type = \"SOME\"", 1), "Must be one of: ALL"},
		"zero interval":      {monitor("    target = \"a\"\n    interval = 0\n    time_period = 60\n    loss_threshold = 1"), "Expected 1 or more"},
		"loss over 100":      {monitor("    target = \"a\"\n    interval = 1\n    time_period = 60\n    loss_threshold = 101"), "0 to 100"},
		"period < interval":  {monitor("    target = \"a\"\n    interval = 10\n    time_period = 5\n    loss_threshold = 1"), "at least one probe"},
		"icmp without host":  {monitor("    interval = 1\n    time_period = 60\n    loss_threshold = 1"), "host to ping"},
		"icmp with domain":   {monitor("    target = \"a\"\n    domain = \"d\"\n    interval = 1\n    time_period = 60\n    loss_threshold = 1"), "Only DNS monitors"},
		"dns without domain": {strings.Replace(monitor("    target = \"a\"\n    interval = 1\n    time_period = 60\n    loss_threshold = 1"), "ICMP", "DNS", 1), "name to resolve"},
		"no threshold":       {monitor("    target = \"a\"\n    interval = 1\n    time_period = 60"), "never fails"},
		"bad policy":         {strings.Replace(monitor("    target = \"a\"\n    interval = 1\n    time_period = 60\n    loss_threshold = 1"), "\"any\"", "\"some\"", 1), "Must be one of: any"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      wanSLAConfig(api, "x", testCase.body),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(testCase.err),
				}},
			})
		})
	}
	if api.wanSLAWrites != 0 {
		t.Fatalf("validation errors must stop before any write, got %d", api.wanSLAWrites)
	}
}

func TestResourceWANSLAImportNeedsCompositeID(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	api.wanSLAs = []map[string]any{mockWANSLA()}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:        wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 25)),
			ResourceName:  "unifi_wan_sla.meo",
			ImportState:   true,
			ImportStateId: "sla-meo",
			ExpectError:   regexp.MustCompile("<site_id>/<resource_id>"),
		}},
	})
}

// Every controller error reaches the diagnostics: create, read, update and
// delete each fail once.
func TestAccResourceWANSLAControllerErrors(t *testing.T) {
	api := newMockUniFiAPI(t)
	defer api.Close()
	api.wanSLAs = []map[string]any{mockWANSLA()}
	fail := func(method string) func() {
		return func() { api.mu.Lock(); api.wanSLAFailMethod = method; api.mu.Unlock() }
	}
	imported := wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 25))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{PreConfig: fail(http.MethodPost), Config: wanSLAConfig(api, "new", fmt.Sprintf(meoSLABody, 25)), ExpectError: regexp.MustCompile("Unable to create WAN SLA")},
			{PreConfig: fail(""), Config: imported, ResourceName: "unifi_wan_sla.meo", ImportState: true, ImportStateId: api.siteID + "/sla-meo", ImportStatePersist: true},
			{PreConfig: fail(http.MethodPut), Config: wanSLAConfig(api, "meo", fmt.Sprintf(meoSLABody, 40)), ExpectError: regexp.MustCompile("Unable to update WAN SLA")},
			{PreConfig: fail(http.MethodGet), Config: imported, PlanOnly: true, ExpectError: regexp.MustCompile("Unable to read WAN SLA")},
			{PreConfig: fail(http.MethodDelete), Config: siteLookupConfig(api.URL()), ExpectError: regexp.MustCompile("Unable to delete WAN SLA")},
			{PreConfig: fail(""), Config: siteLookupConfig(api.URL())},
		},
	})
}
