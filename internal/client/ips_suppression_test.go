package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const ipsSuppressionTestSite = "11111111-1111-1111-1111-111111111111"

func ipsSuppressionTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/integration/v1/sites" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"data":[{"id":"`+ipsSuppressionTestSite+`","internalReference":"default","name":"Default"}],"totalCount":1}`)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	c, err := New(Config{BaseURL: server.URL, APIKey: "fake-api-key"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestGetIPSSuppression(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/proxy/network/api/s/default/get/setting/ips_suppression" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ips","alerts":[{"id":1}]},{"_id":"s1","key":"ips_suppression","alerts":[{"gid":1,"id":2100498,"category":"emerging-scan","signature":"ET SCAN x","type":"track","tracking":[{"mode":"ip","value":"192.0.2.5","direction":"both"}]}],"whitelist":[]}]}`)
	})
	setting, err := c.GetIPSSuppression(context.Background(), ipsSuppressionTestSite)
	if err != nil {
		t.Fatal(err)
	}
	if len(setting.Alerts) != 1 || setting.Alerts[0].ID != 2100498 || setting.Alerts[0].Tracking[0].Value != "192.0.2.5" {
		t.Fatalf("wrong setting picked or decoded: %#v", setting)
	}
}

func TestGetIPSSuppressionMissingAndNullAlerts(t *testing.T) {
	missing := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ips"}]}`)
	})
	if _, err := missing.GetIPSSuppression(context.Background(), ipsSuppressionTestSite); !IsNotFound(err) {
		t.Fatalf("a site without the setting must be not found, got %v", err)
	}
	null := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ips_suppression","alerts":null}]}`)
	})
	setting, err := null.GetIPSSuppression(context.Background(), ipsSuppressionTestSite)
	if err != nil || setting.Alerts == nil || len(setting.Alerts) != 0 {
		t.Fatalf("null alerts must read as an empty list: %#v %v", setting, err)
	}
	failing := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := failing.GetIPSSuppression(context.Background(), ipsSuppressionTestSite); err == nil || !strings.Contains(err.Error(), "read IPS suppression") {
		t.Fatalf("expected a read error, got %v", err)
	}
	if _, err := failing.GetIPSSuppression(context.Background(), "22222222-2222-2222-2222-222222222222"); err == nil || !strings.Contains(err.Error(), "site") {
		t.Fatalf("expected a site error, got %v", err)
	}
}

func TestSetIPSSuppressionOverlaysOnlyAlerts(t *testing.T) {
	stored := `{"_id":"s1","key":"ips_suppression","site_id":"legacy","whitelist":[{"mode":"ip","value":"192.0.2.9","direction":"src"}],"future_field":{"n":12345678901234567},"alerts":[]}`
	var put map[string]json.RawMessage
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[`+stored+`]}`)
		case r.Method == http.MethodPut && r.URL.Path == "/proxy/network/api/s/default/set/setting/ips_suppression":
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &put); err != nil {
				t.Fatal(err)
			}
			stored = string(body)
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	setting, err := c.SetIPSSuppression(context.Background(), ipsSuppressionTestSite, []IPSSuppressionAlert{{GID: 1, ID: 2016642, Category: "c", Signature: "s", Type: "all"}})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"_id": `"s1"`, "key": `"ips_suppression"`, "site_id": `"legacy"`, "future_field": `{"n":12345678901234567}`, "whitelist": `[{"mode":"ip","value":"192.0.2.9","direction":"src"}]`} {
		if string(put[key]) != want {
			t.Errorf("%s not preserved: got %s, want %s", key, put[key], want)
		}
	}
	if strings.Contains(string(put["alerts"]), `"tracking"`) {
		t.Errorf("an alert without tracking must omit the field like the controller does, got %s", put["alerts"])
	}
	if len(setting.Alerts) != 1 || setting.Alerts[0].ID != 2016642 {
		t.Fatalf("state must be read back after the PUT: %#v", setting)
	}

	if _, err := c.SetIPSSuppression(context.Background(), ipsSuppressionTestSite, nil); err != nil {
		t.Fatal(err)
	}
	if string(put["alerts"]) != "[]" {
		t.Fatalf("nil alerts must be sent as [], got %s", put["alerts"])
	}
}

func TestSetIPSSuppressionErrors(t *testing.T) {
	rejecting := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ips_suppression","alerts":[]}]}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"error","msg":"api.err.InvalidPayload"},"data":[]}`)
	})
	_, err := rejecting.SetIPSSuppression(context.Background(), ipsSuppressionTestSite, nil)
	if err == nil || !strings.Contains(err.Error(), "api.err.InvalidPayload") {
		t.Fatalf("a rejected PUT reported with HTTP 200 must fail, got %v", err)
	}
	if _, err := rejecting.SetIPSSuppression(context.Background(), "22222222-2222-2222-2222-222222222222", nil); err == nil || !strings.Contains(err.Error(), "site") {
		t.Fatalf("expected a site error, got %v", err)
	}
}

// The legacy set/setting endpoint creates a missing document, so a site that
// has never had the setting is written from key and alerts alone.
func TestSetIPSSuppressionCreatesMissingSetting(t *testing.T) {
	stored := ""
	var put map[string]json.RawMessage
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if stored == "" {
				_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[`+stored+`]}`)
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &put); err != nil {
				t.Fatal(err)
			}
			stored = string(body)
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
		}
	})
	setting, err := c.SetIPSSuppression(context.Background(), ipsSuppressionTestSite, []IPSSuppressionAlert{{GID: 1, ID: 2016642, Category: "c", Signature: "s", Type: "all"}})
	if err != nil {
		t.Fatalf("creating the missing setting must succeed, got %v", err)
	}
	if len(put) != 2 || string(put["key"]) != `"ips_suppression"` || put["alerts"] == nil {
		t.Fatalf("a missing setting is created from key and alerts only, got %s", put)
	}
	if len(setting.Alerts) != 1 || setting.Alerts[0].ID != 2016642 {
		t.Fatalf("state must be read back after the create: %#v", setting)
	}
}

// An alert without a description inherits the one the controller already
// holds for the same gid and signature.
func TestSetIPSSuppressionKeepsDescriptions(t *testing.T) {
	var put struct {
		Alerts []IPSSuppressionAlert `json:"alerts"`
	}
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ips_suppression","alerts":[{"gid":1,"id":2016642,"category":"c","signature":"ET known","type":"all"},{"gid":1,"id":9,"category":"c","signature":"ET replaced","type":"all"}]}]}`)
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &put); err != nil {
				t.Fatal(err)
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
		}
	})
	_, err := c.SetIPSSuppression(context.Background(), ipsSuppressionTestSite, []IPSSuppressionAlert{
		{GID: 1, ID: 2016642, Category: "c", Type: "all"},
		{GID: 1, ID: 9, Category: "c", Signature: "mine", Type: "all"},
		{GID: 1, ID: 7, Category: "c", Type: "all"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{put.Alerts[0].Signature, put.Alerts[1].Signature, put.Alerts[2].Signature}
	if got[0] != "ET known" || got[1] != "mine" || got[2] != "" {
		t.Fatalf("descriptions: got %q, want [ET known mine \"\"]", got)
	}
}
