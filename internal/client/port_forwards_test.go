package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func portForwardTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			t.Error("missing API key")
		}
		if r.URL.Path == "/integration/v1/sites" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"data":[{"id":"11111111-1111-1111-1111-111111111111","internalReference":"default","name":"Default"}],"totalCount":1}`)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/proxy/network/api/s/default/rest/portforward") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	c, err := New(Config{BaseURL: server.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPortForwardJSONRoundTrip(t *testing.T) {
	for _, ports := range []string{"3074", "3000-3010", "80,443,8080"} {
		t.Run(ports, func(t *testing.T) {
			original := PortForward{ID: "legacy-id", Name: "forward", Enabled: true, WANInterface: "wan2", Protocol: "udp", Source: "198.51.100.0/24", SourceLimitingEnabled: true, DestinationPort: ports, ForwardIP: "192.0.2.90", ForwardPort: ports, LoggingEnabled: true}
			body, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			var decoded PortForward
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal(err)
			}
			decoded.rawFields = nil
			if !reflect.DeepEqual(original, decoded) {
				t.Fatalf("round trip = %#v, want %#v", decoded, original)
			}
		})
	}
}

func TestPortForwardReadErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		notFound   bool
	}{
		{"absent", `{"meta":{"rc":"ok"},"data":[]}`, 200, true},
		{"forbidden", `{"meta":{"rc":"error","msg":"denied"}}`, 403, false},
		{"legacy error", `{"meta":{"rc":"error","msg":"denied"},"data":[]}`, 200, false},
		{"missing data", `{"meta":{"rc":"ok"}}`, 200, false},
		{"null data", `{"meta":{"rc":"ok"},"data":null}`, 200, false},
		{"empty body", "", 200, false},
		{"malformed", "not-json", 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := portForwardTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status); _, _ = fmt.Fprint(w, tc.body) })
			_, err := c.GetPortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "absent")
			if err == nil || IsNotFound(err) != tc.notFound {
				t.Fatalf("error = %v, want not-found=%v", err, tc.notFound)
			}
		})
	}
}

func TestPortForwardDeleteResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"deleted", `{"meta":{"rc":"ok"},"data":[]}`, 200, false},
		{"no content", "", 204, false},
		{"already absent", "", 404, false},
		{"legacy invalid id 200", `{"meta":{"rc":"error","msg":"api.err.IdInvalid"}}`, 200, false},
		{"legacy invalid id 400", `{"meta":{"rc":"error","msg":"api.err.IdInvalid"}}`, 400, false},
		{"legacy not found 200", `{"meta":{"rc":"error","msg":"api.err.NotFound"}}`, 200, false},
		{"legacy not found 400", `{"meta":{"rc":"error","msg":"api.err.NotFound"}}`, 400, false},
		{"forbidden", "", 403, true},
		{"legacy error", `{"meta":{"rc":"error","msg":"denied"}}`, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || !strings.HasSuffix(r.URL.Path, "/rule-id") {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			})
			err := c.DeletePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error=%v", err, tc.wantError)
			}
		})
	}
}

func TestPortForwardCreateResponseRequiresID(t *testing.T) {
	for _, body := range []string{`{"data":[]}`, `{"data":[{"name":"missing-id"}]}`, `{"data":[{"_id":"one"},{"_id":"two"}]}`} {
		c := portForwardTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) })
		if _, err := c.CreatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", PortForward{}); err == nil {
			t.Fatalf("accepted response %s", body)
		}
	}
}

func TestPortForwardUpdatePreservesUnmanagedFields(t *testing.T) {
	c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["destination_ip"] != "203.0.113.5" || !reflect.DeepEqual(body["destination_ips"], []any{"203.0.113.5"}) || body["src_firewall_group_id"] != "" || body["future_option"] != "preserve-me" {
				t.Errorf("update lost controller fields: %#v", body)
			}
			if body["src_limiting_enabled"] != true {
				t.Error("source limiting must be enabled")
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"rule-id","dst_port":"80,443","fwd_port":"8080,8443","destination_ip":"203.0.113.5","destination_ips":["203.0.113.5"],"src_firewall_group_id":"","future_option":"preserve-me"}]}`)
	})
	rule, err := c.UpdatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id", PortForward{Source: "198.51.100.1"})
	if err != nil {
		t.Fatal(err)
	}
	if rule.DestinationPort != "80,443" || rule.ForwardPort != "8080,8443" {
		t.Fatalf("update did not read controller state: %#v", rule)
	}
}

func TestPortForwardUpdatePreservesSourceFirewallGroup(t *testing.T) {
	wrote := false
	c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			wrote = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["src_firewall_group_id"] != "group-id" || body["src_limiting_enabled"] != true || body["name"] != "renamed" || body["enabled"] != false {
				t.Errorf("unexpected update: %#v", body)
			}
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"rule-id","src":"any","src_limiting_enabled":true,"src_firewall_group_id":"group-id"}]}`)
	})
	_, err := c.UpdatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id", PortForward{Name: "renamed", Source: "any"})
	if err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("rule with a source group was not updated")
	}
}

type portForwardCountingTransport struct {
	http.RoundTripper
	sites, total int
}

func (c *portForwardCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.total++
	if req.URL.Path == "/integration/v1/sites" {
		c.sites++
	}
	return c.RoundTripper.RoundTrip(req)
}

func TestPortForwardUpdateRequestCount(t *testing.T) {
	for _, echo := range []bool{true, false} {
		t.Run(fmt.Sprint(echo), func(t *testing.T) {
			stored := map[string]any{"_id": "rule-id", "name": "before", "enabled": true, "pfwd_interface": "wan", "proto": "tcp_udp", "src": "any", "src_limiting_enabled": false, "src_firewall_group_id": "", "dst_port": "80", "fwd": "192.0.2.90", "fwd_port": "80", "log": false, "destination_ips": []any{"203.0.113.5"}, "future_config": map[string]any{"keep": true}}
			c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					// Full replacement, not merge.
					stored = nil
					if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
						t.Error(err)
					}
					if !echo {
						_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
						return
					}
				}
				if err := json.NewEncoder(w).Encode(map[string]any{"meta": map[string]string{"rc": "ok"}, "data": []any{stored}}); err != nil {
					t.Error(err)
				}
			})
			counter := &portForwardCountingTransport{RoundTripper: c.httpClient.Transport}
			c.httpClient.Transport = counter
			rule, err := c.UpdatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id", PortForward{Name: "after", Source: "any", DestinationPort: "80", ForwardPort: "80", ForwardIP: "192.0.2.90", WANInterface: "wan", Protocol: "tcp_udp"})
			if err != nil {
				t.Fatal(err)
			}
			if rule.Name != "after" || rule.Enabled {
				t.Fatalf("did not read updated state: %#v", rule)
			}
			if !reflect.DeepEqual(stored["destination_ips"], []any{"203.0.113.5"}) || !reflect.DeepEqual(stored["future_config"], map[string]any{"keep": true}) {
				t.Fatalf("lost unmodelled fields: %#v", stored)
			}
			want := 3
			if !echo {
				want = 4
			}
			if counter.sites != 1 || counter.total != want {
				t.Fatalf("requests: %d sites, %d total; want 1 site, %d total", counter.sites, counter.total, want)
			}
		})
	}
}

func TestPortForwardUpdateRepairsDisabledSourceLimiting(t *testing.T) {
	// An admin can switch source limiting off in the UI while leaving src in
	// place. Re-applying the same source must re-enable the restriction.
	wrote := false
	c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			wrote = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["src"] != "198.51.100.0/24" || body["src_limiting_enabled"] != true {
				t.Errorf("source limiting was not repaired: %#v", body)
			}
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"rule-id","src":"198.51.100.0/24","src_limiting_enabled":false,"src_firewall_group_id":""}]}`)
	})
	if _, err := c.UpdatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id", PortForward{Source: "198.51.100.0/24"}); err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("rule was not updated")
	}
}

func TestPortForwardUpdateRejectsSourceConflictingWithFirewallGroup(t *testing.T) {
	c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected write %s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"rule-id","src":"any","src_limiting_enabled":true,"src_firewall_group_id":"group-id"}]}`)
	})
	_, err := c.UpdatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id", PortForward{Source: "203.0.113.9"})
	if err == nil || !strings.Contains(err.Error(), "source firewall group") {
		t.Fatalf("expected a source-group conflict error, got %v", err)
	}
}

func TestPortForwardCreateSendsEmptySourceFirewallGroup(t *testing.T) {
	c := portForwardTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if value, ok := body["src_firewall_group_id"]; !ok || value != "" {
			t.Errorf("create must send an empty src_firewall_group_id: %#v", body)
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"rule-id"}]}`)
	})
	if _, err := c.CreatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", PortForward{Source: "any"}); err != nil {
		t.Fatal(err)
	}
}
