package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyErrorDiagnostics(t *testing.T) {
	for _, status := range []int{200, 400} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := `{"meta":{"rc":"error"},"detail":"controller diagnostic"}`
			c := portForwardTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status); _, _ = fmt.Fprint(w, body) })
			err := c.DeletePortForward(context.Background(), "11111111-1111-1111-1111-111111111111", "rule-id")
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.Body != body || apiErr.StatusCode < 400 || !strings.Contains(err.Error(), "controller diagnostic") {
				t.Fatalf("lost diagnostic context: %#v / %v", apiErr, err)
			}
		})
	}
}

func TestDHCPReservationEmptyLegacyList(t *testing.T) {
	for _, body := range []string{`{"meta":{"rc":"ok"},"data":[]}`, `{"meta":{"rc":"ok"},"data":null}`, `{"meta":{"rc":"ok"}}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/integration/v1/sites" {
					_, _ = fmt.Fprint(w, `{"data":[{"id":"11111111-1111-1111-1111-111111111111","internalReference":"default","name":"Default"}],"totalCount":1}`)
					return
				}
				if r.URL.Path != "/proxy/network/api/s/default/rest/user" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				_, _ = fmt.Fprint(w, body)
			}))
			defer server.Close()
			c, err := New(Config{BaseURL: server.URL, APIKey: "test-key"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.GetDHCPReservation(context.Background(), "11111111-1111-1111-1111-111111111111", "00:11:22:33:44:55")
			if !IsMissingClient(err) {
				t.Fatalf("empty list must retain DHCP bootstrap path, got %v", err)
			}
		})
	}
}
