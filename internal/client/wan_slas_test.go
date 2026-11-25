package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const wanSLAListJSON = `[{"_id":"sla-1","name":"ping 1.1.1.1","monitor_health_type":"ALL","monitor_down_threshold":2,"monitors":[{"monitor_id":1,"type":"ICMP","target":"1.1.1.1","interval":1,"time_period":60,"alert":{"loss_threshold":25,"latency_threshold":500,"threshold_policy":"any"}}]}]`

func TestV2BaseURL(t *testing.T) {
	cases := map[string]string{
		"https://192.0.2.1/proxy/network/integration":                      "https://192.0.2.1/proxy/network/v2/api",
		"https://192.0.2.1":                                                "https://192.0.2.1/proxy/network/v2/api",
		"https://example.com/unifi/proxy/network/api":                      "https://example.com/unifi/proxy/network/v2/api",
		"https://api.ui.com/v1/connector/consoles/abc/network":             "https://api.ui.com/v1/connector/consoles/abc/network/v2/api",
		"https://api.ui.com/v1/connector/consoles/abc/network/integration": "https://api.ui.com/v1/connector/consoles/abc/network/v2/api",
	}
	for in, want := range cases {
		c, err := New(Config{BaseURL: in, APIKey: "fake-api-key"})
		if err != nil {
			t.Fatal(err)
		}
		if got := c.v2BaseURL().String(); got != want {
			t.Errorf("%s: v2 base %s, want %s", in, got, want)
		}
	}
}

func TestListAndGetWANSLA(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/proxy/network/v2/api/site/default/wan-slas" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-API-KEY") != "fake-api-key" {
			t.Error("missing API key")
		}
		_, _ = fmt.Fprint(w, wanSLAListJSON)
	})
	slas, err := c.ListWANSLAs(context.Background(), ipsSuppressionTestSite)
	if err != nil || len(slas) != 1 {
		t.Fatalf("list = %v, %v", slas, err)
	}
	sla, err := c.GetWANSLA(context.Background(), ipsSuppressionTestSite, "sla-1")
	if err != nil {
		t.Fatal(err)
	}
	m := sla.Monitors[0]
	if sla.Name != "ping 1.1.1.1" || m.Target != "1.1.1.1" || *m.Alert.LossThreshold != 25 || m.Alert.JitterThreshold != nil || m.Alert.ThresholdPolicy != "any" {
		t.Fatalf("unexpected SLA %+v", sla)
	}
	if _, err := c.GetWANSLA(context.Background(), ipsSuppressionTestSite, "sla-2"); !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestWANSLAErrors(t *testing.T) {
	status, body := http.StatusOK, "not json"
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	})
	ctx := context.Background()
	if _, err := c.ListWANSLAs(ctx, ipsSuppressionTestSite); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected a decode error, got %v", err)
	}
	body = `{"not":"a list"}`
	if _, err := c.ListWANSLAs(ctx, ipsSuppressionTestSite); err == nil {
		t.Fatal("an object instead of a list must fail")
	}
	status, body = http.StatusNotFound, `{"code":"api.err.NotFound","message":"not found"}`
	if err := c.DeleteWANSLA(ctx, ipsSuppressionTestSite, "x"); !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	status, body = http.StatusBadRequest, `{"code":"api.err.Invalid","message":"bad threshold"}`
	if _, err := c.CreateWANSLA(ctx, ipsSuppressionTestSite, WANSLA{Name: "x"}); err == nil || !strings.Contains(err.Error(), "bad threshold") {
		t.Fatalf("expected the controller's message, got %v", err)
	}
	if _, err := c.UpdateWANSLA(ctx, ipsSuppressionTestSite, WANSLA{ID: "x"}); err == nil {
		t.Fatal("a failed read must fail the update")
	}
	for name, call := range map[string]func() error{
		"list":   func() error { _, err := c.ListWANSLAs(ctx, "no-such-site"); return err },
		"get":    func() error { _, err := c.GetWANSLA(ctx, "no-such-site", "x"); return err },
		"create": func() error { _, err := c.CreateWANSLA(ctx, "no-such-site", WANSLA{}); return err },
		"update": func() error { _, err := c.UpdateWANSLA(ctx, "no-such-site", WANSLA{}); return err },
		"delete": func() error { return c.DeleteWANSLA(ctx, "no-such-site", "x") },
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("%s: expected an unknown site error, got %v", name, err)
		}
	}
}

// Update sends back the fields it doesn't model and replaces the others.
func TestUpdateWANSLAKeepsUnmodelledFields(t *testing.T) {
	var put map[string]any
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = fmt.Fprint(w, wanSLAListJSON)
		case r.Method == http.MethodPut && r.URL.Path == "/proxy/network/v2/api/site/default/wan-slas/sla-1":
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &put); err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	loss := int64(30)
	_, err := c.UpdateWANSLA(context.Background(), ipsSuppressionTestSite, WANSLA{ID: "sla-1", Name: "renamed", Monitors: []WANSLAMonitor{{MonitorID: 1, Type: "ICMP", Target: "1.1.1.1", Interval: 1, TimePeriod: 60, Alert: WANSLAAlert{LossThreshold: &loss, ThresholdPolicy: "any"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if put["monitor_down_threshold"] != float64(2) || put["name"] != "renamed" || put["monitor_health_type"] != "ALL" {
		t.Fatalf("unexpected PUT body %v", put)
	}
	alert := put["monitors"].([]any)[0].(map[string]any)["alert"].(map[string]any)
	if alert["loss_threshold"] != float64(30) {
		t.Fatalf("loss_threshold not replaced: %v", alert)
	}
	if _, has := alert["latency_threshold"]; has {
		t.Fatalf("an unset threshold must be left out: %v", alert)
	}
	if _, err := c.UpdateWANSLA(context.Background(), ipsSuppressionTestSite, WANSLA{ID: "sla-9"}); !IsNotFound(err) {
		t.Fatalf("expected not found for an unknown ID, got %v", err)
	}
}

// Without an ID in the answer, the new SLA is the one that wasn't there
// before the POST, even when an older SLA has the same name; with no such SLA,
// or more than one, create fails instead of guessing.
func TestCreateWANSLAWithoutIDInAnswer(t *testing.T) {
	const older = `{"_id":"sla-1","name":"ping 1.1.1.1","monitors":[]}`
	appended := []string{}
	var list []string
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			raw, _ := io.ReadAll(r.Body)
			if strings.Contains(string(raw), `"_id"`) {
				t.Errorf("create must not send an ID: %s", raw)
			}
			list = append(list, appended...)
			_, _ = fmt.Fprint(w, `{}`)
			return
		}
		_, _ = fmt.Fprint(w, "["+strings.Join(list, ",")+"]")
	})
	create := func() (*WANSLA, error) {
		return c.CreateWANSLA(context.Background(), ipsSuppressionTestSite, WANSLA{ID: "stale", Name: "ping 1.1.1.1"})
	}

	list, appended = []string{older}, []string{`{"_id":"sla-2","name":"ping 1.1.1.1","monitors":[]}`}
	if sla, err := create(); err != nil || sla.ID != "sla-2" {
		t.Fatalf("expected the new sla-2, not the older same-name sla-1: %v, %v", sla, err)
	}

	list, appended = []string{older}, nil
	if _, err := create(); err == nil || !strings.Contains(err.Error(), "0 new SLAs") {
		t.Fatalf("expected an error when no new SLA appears, got %v", err)
	}

	list, appended = []string{older}, []string{`{"_id":"sla-2","name":"ping 1.1.1.1"}`, `{"_id":"sla-3","name":"ping 1.1.1.1"}`}
	if _, err := create(); err == nil || !strings.Contains(err.Error(), "2 new SLAs") {
		t.Fatalf("expected an error for two new same-name SLAs, got %v", err)
	}
}

// Fields the client doesn't model survive an update inside each monitor and
// its alert too, matched by monitor_id; a threshold that is now unset is
// removed, not kept from the stored copy.
func TestUpdateWANSLAKeepsUnmodelledMonitorFields(t *testing.T) {
	const stored = `[{"_id":"sla-1","name":"x","monitors":[
		{"monitor_id":1,"type":"ICMP","target":"1.1.1.1","interval":1,"time_period":60,"extra_monitor":"keep-1","alert":{"loss_threshold":25,"latency_threshold":500,"threshold_policy":"any","extra_alert":"keep-a"}},
		{"monitor_id":2,"type":"ICMP","target":"8.8.8.8","interval":1,"time_period":60,"extra_monitor":"keep-2","alert":{"loss_threshold":25}}]}]`
	var put struct {
		Monitors []map[string]any `json:"monitors"`
	}
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &put); err != nil {
				t.Error(err)
			}
			return
		}
		_, _ = fmt.Fprint(w, stored)
	})
	loss := int64(30)
	_, err := c.UpdateWANSLA(context.Background(), ipsSuppressionTestSite, WANSLA{ID: "sla-1", Name: "x", Monitors: []WANSLAMonitor{
		{MonitorID: 1, Type: "ICMP", Target: "1.0.0.1", Interval: 2, TimePeriod: 60, Alert: WANSLAAlert{LossThreshold: &loss}},
		{MonitorID: 3, Type: "DNS", Domain: "example.com", Interval: 5, TimePeriod: 60, Alert: WANSLAAlert{LossThreshold: &loss}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(put.Monitors) != 2 {
		t.Fatalf("expected the two configured monitors, got %v", put.Monitors)
	}
	first, second := put.Monitors[0], put.Monitors[1]
	alert := first["alert"].(map[string]any)
	if first["extra_monitor"] != "keep-1" || first["target"] != "1.0.0.1" || first["interval"] != float64(2) {
		t.Fatalf("monitor 1 not merged: %v", first)
	}
	if alert["extra_alert"] != "keep-a" || alert["loss_threshold"] != float64(30) {
		t.Fatalf("alert not merged: %v", alert)
	}
	for _, gone := range []string{"latency_threshold", "threshold_policy"} {
		if _, has := alert[gone]; has {
			t.Fatalf("unset %s must be removed: %v", gone, alert)
		}
	}
	if _, has := second["extra_monitor"]; has || second["domain"] != "example.com" {
		t.Fatalf("a new monitor must not inherit another's fields: %v", second)
	}
	if _, has := second["target"]; has {
		t.Fatalf("an unset target must be left out: %v", second)
	}
}

func TestCreateAndDeleteWANSLA(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"_id":"sla-1"}`)
		case http.MethodDelete:
			if r.URL.Path != "/proxy/network/v2/api/site/default/wan-slas/sla-1" {
				t.Errorf("unexpected delete %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = fmt.Fprint(w, wanSLAListJSON)
		}
	})
	sla, err := c.CreateWANSLA(context.Background(), ipsSuppressionTestSite, WANSLA{Name: "ping 1.1.1.1"})
	if err != nil || sla.ID != "sla-1" || sla.MonitorHealthType != "ALL" {
		t.Fatalf("create = %+v, %v", sla, err)
	}
	if err := c.DeleteWANSLA(context.Background(), ipsSuppressionTestSite, "sla-1"); err != nil {
		t.Fatal(err)
	}
}

// A 2xx answer that isn't JSON fails a call that reads it, and a connection
// that fails is reported.
func TestDoV2RequestBadAnswers(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = fmt.Fprint(w, "<html>")
			return
		}
		_, _ = fmt.Fprint(w, "[]")
	})
	if _, err := c.CreateWANSLA(context.Background(), ipsSuppressionTestSite, WANSLA{Name: "x"}); err == nil || !strings.Contains(err.Error(), "decode v2 response") {
		t.Fatalf("expected a decode error, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := c.ListWANSLAs(ctx, ipsSuppressionTestSite); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, _, err := c.listWANSLAs(ctx, "default"); err == nil {
		t.Fatal("a cancelled request must fail")
	}
}
