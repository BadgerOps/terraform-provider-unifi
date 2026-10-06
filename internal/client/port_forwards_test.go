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
			for _, key := range []string{"destination_ip", "destination_ips", "src_firewall_group_id"} {
				if _, ok := body[key]; ok {
					t.Errorf("update overwrites %s", key)
				}
			}
			if body["src_limiting_enabled"] != true {
				t.Error("source limiting must be enabled")
			}
			_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[]}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"meta":{"rc":"ok"},"data":[{"_id":"rule-id","dst_port":"80,443","fwd_port":"8080,8443"}]}`)
	})
	rule, err := c.UpdatePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id", PortForward{Source: "198.51.100.1"})
	if err != nil {
		t.Fatal(err)
	}
	if rule.DestinationPort != "80,443" || rule.ForwardPort != "8080,8443" {
		t.Fatalf("update did not read controller state: %#v", rule)
	}
}
