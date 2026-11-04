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

func TestGetSiteSetting(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/proxy/network/api/s/default/get/setting/ntp" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"rsyslogd","enabled":true},{"_id":"s1","key":"ntp","setting_preference":"auto","ntp_server_1":"0.pool.ntp.org","ntp_server_2":null}]}`)
	})
	setting, err := c.GetSiteSetting(context.Background(), ipsSuppressionTestSite, "ntp")
	if err != nil {
		t.Fatal(err)
	}
	var preference, server, missing string
	if found, err := setting.Decode("setting_preference", &preference); err != nil || !found || preference != "auto" {
		t.Fatalf("setting_preference = %q, %v, %v", preference, found, err)
	}
	if found, _ := setting.Decode("ntp_server_1", &server); !found || server != "0.pool.ntp.org" {
		t.Fatalf("ntp_server_1 = %q", server)
	}
	if found, err := setting.Decode("ntp_server_2", &missing); found || err != nil {
		t.Fatalf("a null field must read as absent, got found=%v err=%v", found, err)
	}
	if found, _ := setting.Decode("ntp_server_3", &missing); found {
		t.Fatal("a missing field must read as absent")
	}
	var wrongType int
	if _, err := setting.Decode("setting_preference", &wrongType); err == nil {
		t.Fatal("decoding a string into an int must fail")
	}
}

func TestGetSiteSettingMissing(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"rsyslogd"}]}`)
	})
	_, err := c.GetSiteSetting(context.Background(), ipsSuppressionTestSite, "snmp")
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetSiteSettingUnknownSite(t *testing.T) {
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no setting request expected, got %s", r.URL.Path)
	})
	if _, err := c.GetSiteSetting(context.Background(), "no-such-site", "ntp"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected an unknown site error, got %v", err)
	}
	if _, err := c.UpdateSiteSetting(context.Background(), "no-such-site", "ntp", nil); err == nil {
		t.Fatal("expected an unknown site error")
	}
}

// The update keeps every field it doesn't change, including credentials, and
// returns what the controller stored, not what was sent.
func TestUpdateSiteSetting(t *testing.T) {
	stored := `{"_id":"s1","key":"snmp","site_id":"legacy","enabled":false,"community":"fake-community","x_password":"fake-password"}`
	var put map[string]any
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/proxy/network/api/s/default/get/setting/snmp":
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[`+stored+`]}`)
		case r.Method == http.MethodPut && r.URL.Path == "/proxy/network/api/s/default/set/setting/snmp":
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &put); err != nil {
				t.Fatal(err)
			}
			stored = string(body)
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	setting, err := c.UpdateSiteSetting(context.Background(), ipsSuppressionTestSite, "snmp", map[string]any{"enabled": true, "enabledV3": false})
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]any{"_id": "s1", "key": "snmp", "site_id": "legacy", "enabled": true, "enabledV3": false, "community": "fake-community", "x_password": "fake-password"} {
		if put[field] != want {
			t.Errorf("PUT %s = %#v, want %#v", field, put[field], want)
		}
	}
	var enabled bool
	if _, err := setting.Decode("enabled", &enabled); err != nil || !enabled {
		t.Fatalf("read back enabled = %v, %v", enabled, err)
	}
}

// A fresh site answers get/setting with an empty document, without a key.
func TestUpdateSiteSettingCreatesMissing(t *testing.T) {
	var put map[string]any
	c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &put); err != nil {
				t.Fatal(err)
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
		case put != nil:
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"s1","key":"snmp","enabled":false}]}`)
		default:
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{}]}`)
		}
	})
	setting, err := c.UpdateSiteSetting(context.Background(), ipsSuppressionTestSite, "snmp", map[string]any{"enabled": false})
	if err != nil {
		t.Fatal(err)
	}
	if len(put) != 2 || put["key"] != "snmp" || put["enabled"] != false {
		t.Fatalf("unexpected create body: %#v", put)
	}
	if string(setting["_id"]) != `"s1"` {
		t.Fatalf("expected the stored document, got %#v", setting)
	}
}

func TestUpdateSiteSettingErrors(t *testing.T) {
	t.Run("refused create", func(t *testing.T) {
		c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"meta":{"rc":"error","msg":"api.err.Invalid"},"data":[]}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{}]}`)
		})
		if _, err := c.UpdateSiteSetting(context.Background(), ipsSuppressionTestSite, "ntp", map[string]any{"setting_preference": "auto"}); err == nil || !strings.Contains(err.Error(), "refused to create") {
			t.Fatalf("expected the create to be refused, got %v", err)
		}
	})
	t.Run("broken read", func(t *testing.T) {
		c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"error","msg":"api.err.Internal"},"data":[]}`)
		})
		if _, err := c.UpdateSiteSetting(context.Background(), ipsSuppressionTestSite, "ntp", map[string]any{"setting_preference": "auto"}); err == nil || IsNotFound(err) {
			t.Fatalf("expected the read error, got %v", err)
		}
	})
	t.Run("unencodable value", func(t *testing.T) {
		c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ntp"}]}`)
		})
		if _, err := c.UpdateSiteSetting(context.Background(), ipsSuppressionTestSite, "ntp", map[string]any{"bad": make(chan int)}); err == nil || !strings.Contains(err.Error(), "encode setting field bad") {
			t.Fatalf("expected an encode error, got %v", err)
		}
	})
	t.Run("rejected write", func(t *testing.T) {
		c := ipsSuppressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"meta":{"rc":"error","msg":"api.err.Invalid"},"data":[]}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"key":"ntp"}]}`)
		})
		if _, err := c.UpdateSiteSetting(context.Background(), ipsSuppressionTestSite, "ntp", map[string]any{"setting_preference": "auto"}); err == nil || !strings.Contains(err.Error(), "update setting ntp") {
			t.Fatalf("expected a write error, got %v", err)
		}
	})
}
