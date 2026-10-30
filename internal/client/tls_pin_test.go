package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A TLS test server has a self-signed certificate for 127.0.0.1/example.com,
// like a console's for unifi.local: standard verification refuses it.
func newPinTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	sum := sha256.Sum256(server.Certificate().Raw)
	return server, hex.EncodeToString(sum[:])
}

func pinTestGet(t *testing.T, server *httptest.Server, config Config) error {
	t.Helper()
	config.BaseURL, config.APIKey = server.URL, "fake-api-key"
	c, err := New(config)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	response, err := c.httpClient.Get(server.URL)
	if err == nil {
		_ = response.Body.Close()
	}
	return err
}

func TestTLSCertificatePinAcceptsOnlyThatCertificate(t *testing.T) {
	server, pin := newPinTestServer(t)
	colons := strings.ToUpper(pin[0:2])
	for i := 2; i < len(pin); i += 2 {
		colons += ":" + strings.ToUpper(pin[i:i+2])
	}
	for name, value := range map[string]string{"hex": pin, "openssl form": colons} {
		t.Run(name, func(t *testing.T) {
			if err := pinTestGet(t, server, Config{TLSCertificateSHA256: value}); err != nil {
				t.Fatalf("pinned certificate refused: %v", err)
			}
		})
	}

	// Every httptest TLS server shares one built-in certificate: the other
	// server needs one of its own.
	other := newServerWithOwnCertificate(t)
	err := pinTestGet(t, other, Config{TLSCertificateSHA256: pin})
	if err == nil || !strings.Contains(err.Error(), "does not match tls_certificate_sha256") {
		t.Fatalf("another certificate must be refused, got %v", err)
	}
}

func TestWithoutPinSelfSignedIsRefused(t *testing.T) {
	server, _ := newPinTestServer(t)
	if err := pinTestGet(t, server, Config{}); err == nil {
		t.Fatal("a self-signed certificate must fail standard verification")
	}
}

func TestTLSCertificatePinConfigErrors(t *testing.T) {
	for name, config := range map[string]Config{
		"too short":           {TLSCertificateSHA256: "abcd"},
		"not hex":             {TLSCertificateSHA256: strings.Repeat("zz", 32)},
		"with allow_insecure": {TLSCertificateSHA256: strings.Repeat("ab", 32), AllowInsecure: true},
	} {
		t.Run(name, func(t *testing.T) {
			config.BaseURL, config.APIKey = "https://192.0.2.1", "fake-api-key"
			if _, err := New(config); err == nil || !strings.Contains(err.Error(), "tls_certificate_sha256") {
				t.Fatalf("want a tls_certificate_sha256 error, got %v", err)
			}
		})
	}
}

func TestPinnedTLSConfigNeedsACertificate(t *testing.T) {
	config, err := pinnedTLSConfig(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	if err := config.VerifyPeerCertificate(nil, nil); err == nil {
		t.Fatal("a handshake without a certificate must fail")
	}
}

func newServerWithOwnCertificate(t *testing.T) *httptest.Server {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "unifi.local"},
		DNSNames:     []string{"unifi.local"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func TestTLSCertificatePinRequiresHTTPS(t *testing.T) {
	for _, scheme := range []string{"http", "ftp"} {
		t.Run(scheme, func(t *testing.T) {
			_, err := New(Config{BaseURL: scheme + "://192.0.2.1", APIKey: "fake-api-key", TLSCertificateSHA256: strings.Repeat("ab", 32)})
			if err == nil || !strings.Contains(err.Error(), "tls_certificate_sha256 requires an https api_url") {
				t.Fatalf("want HTTPS configuration error, got %v", err)
			}
		})
	}
}

func TestTLSCertificatePinRedirects(t *testing.T) {
	for _, api := range []string{"integration", "legacy"} {
		for _, scenario := range []string{"http downgrade", "https same certificate", "https different certificate", "https loop"} {
			t.Run(api+"/"+scenario, func(t *testing.T) {
				var plaintextRequests, finalRequests, redirects atomic.Int32
				plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					plaintextRequests.Add(1)
					w.WriteHeader(http.StatusNoContent)
				}))
				defer plain.Close()
				other := newServerWithOwnCertificate(t)
				var server *httptest.Server
				server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-API-KEY") != "fake-api-key" {
						t.Error("missing API key")
					}
					if r.URL.Path == "/final" {
						finalRequests.Add(1)
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"data":[],"totalCount":0}`))
						return
					}
					redirects.Add(1)
					target := server.URL + "/final"
					switch scenario {
					case "http downgrade":
						target = plain.URL
					case "https different certificate":
						target = other.URL
					case "https loop":
						target = server.URL + "/loop"
					}
					http.Redirect(w, r, target, http.StatusFound)
				}))
				defer server.Close()
				sum := sha256.Sum256(server.Certificate().Raw)
				c, err := New(Config{BaseURL: server.URL, APIKey: "fake-api-key", TLSCertificateSHA256: hex.EncodeToString(sum[:])})
				if err != nil {
					t.Fatal(err)
				}
				if api == "integration" {
					_, err = c.ListSites(context.Background())
				} else {
					err = c.doLegacyRequest(context.Background(), http.MethodGet, []string{"s", "default", "rest", "user"}, nil, nil)
				}
				switch scenario {
				case "http downgrade":
					if err == nil || !strings.Contains(err.Error(), "tls_certificate_sha256 requires HTTPS redirects") {
						t.Fatalf("want downgrade rejection, got %v", err)
					}
					if got := plaintextRequests.Load(); got != 0 {
						t.Fatalf("sent %d plaintext requests, exposing the API key", got)
					}
				case "https same certificate":
					if err != nil {
						t.Fatalf("pinned HTTPS redirect failed: %v", err)
					}
					if got := finalRequests.Load(); got != 1 {
						t.Fatalf("want one final request, got %d", got)
					}
				case "https different certificate":
					if err == nil || !strings.Contains(err.Error(), "does not match tls_certificate_sha256") {
						t.Fatalf("want pin mismatch, got %v", err)
					}
				case "https loop":
					if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
						t.Fatalf("want redirect limit, got %v", err)
					}
					if got := redirects.Load(); got != 10 {
						t.Fatalf("want 10 requests, got %d", got)
					}
				}
			})
		}
	}
}
