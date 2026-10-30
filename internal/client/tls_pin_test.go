package client

import (
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
