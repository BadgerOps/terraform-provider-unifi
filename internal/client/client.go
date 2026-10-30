package client

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

const defaultPageLimit = 100

const DefaultRequestTimeout = 30 * time.Second

type Config struct {
	BaseURL       string
	APIKey        string
	AllowInsecure bool
	// TLSCertificateSHA256 pins the server certificate by the SHA-256 of its
	// DER encoding (hex, colons optional). Set, it replaces CA and host name
	// verification: only that exact certificate is accepted. For a console's
	// self-signed certificate, which names unifi.local, not the address used.
	TLSCertificateSHA256 string
	UserAgent            string
	// RequestTimeout bounds each HTTP request to the controller. Zero selects
	// DefaultRequestTimeout.
	RequestTimeout time.Duration
}

type Client struct {
	apiClient          *generated.ClientWithResponses
	httpClient         *http.Client
	integrationBaseURL *url.URL
	legacyBaseURL      *url.URL
	apiKey             string
	userAgent          string
}

type apiError struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	HTTPStatusCode int    `json:"httpStatusCode"`
}

type page[T any] struct {
	Offset     int64 `json:"offset"`
	Limit      int   `json:"limit"`
	Count      int   `json:"count"`
	TotalCount int64 `json:"totalCount"`
	Data       []T   `json:"data"`
}

type Site struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"`
	Name              string `json:"name"`
}

func New(config Config) (*Client, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("api_url must not be empty")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("api_key must not be empty")
	}
	if config.RequestTimeout < 0 {
		return nil, fmt.Errorf("request_timeout_seconds must not be negative")
	}

	integrationBaseURL, err := normalizeBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	legacyBaseURL, err := normalizeLegacyBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	pin := strings.TrimSpace(config.TLSCertificateSHA256)
	switch {
	case pin != "" && config.AllowInsecure:
		return nil, fmt.Errorf("tls_certificate_sha256 and allow_insecure exclude each other")
	case pin != "":
		if integrationBaseURL.Scheme != "https" {
			return nil, fmt.Errorf("tls_certificate_sha256 requires an https api_url")
		}
		tlsConfig, err := pinnedTLSConfig(pin)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = tlsConfig
	case config.AllowInsecure:
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	requestTimeout := config.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = DefaultRequestTimeout
	}

	httpClient := &http.Client{
		Timeout:   requestTimeout,
		Transport: transport,
	}
	if pin != "" {
		httpClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
			if request.URL.Scheme != "https" {
				return fmt.Errorf("tls_certificate_sha256 requires HTTPS redirects")
			}
			// Preserve net/http's default limit when installing a redirect policy.
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		}
	}

	apiClient, err := generated.NewClientWithResponses(
		integrationBaseURL.String(),
		generated.WithHTTPClient(httpClient),
		generated.WithRequestEditorFn(func(_ context.Context, request *http.Request) error {
			request.Header.Set("Accept", "application/json")
			request.Header.Set("X-API-KEY", config.APIKey)
			if config.UserAgent != "" {
				request.Header.Set("User-Agent", config.UserAgent)
			}

			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create generated api client: %w", err)
	}

	return &Client{
		apiClient:          apiClient,
		httpClient:         httpClient,
		integrationBaseURL: integrationBaseURL,
		legacyBaseURL:      legacyBaseURL,
		apiKey:             config.APIKey,
		userAgent:          config.UserAgent,
	}, nil
}

// pinnedTLSConfig accepts exactly one server certificate. The standard
// verification is switched off only to be replaced by VerifyPeerCertificate,
// which runs on every handshake and fails closed on any other certificate.
func pinnedTLSConfig(pin string) (*tls.Config, error) {
	want, err := hex.DecodeString(strings.ToLower(strings.ReplaceAll(pin, ":", "")))
	if err != nil || len(want) != sha256.Size {
		return nil, fmt.Errorf("tls_certificate_sha256 must be the 64 hex digits of a SHA-256 fingerprint")
	}
	return &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // replaced by the pin check in VerifyPeerCertificate
		MinVersion:         tls.VersionTLS12,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("server sent no certificate")
			}
			got := sha256.Sum256(rawCerts[0])
			if subtle.ConstantTimeCompare(got[:], want) != 1 {
				return fmt.Errorf("server certificate sha256 %s does not match tls_certificate_sha256", hex.EncodeToString(got[:]))
			}
			return nil
		},
	}, nil
}

func normalizeBaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse api_url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("api_url must include scheme and host")
	}

	trimmedPath := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(trimmedPath, "/integration") {
		parsed.Path = joinURLPath(trimmedPath, "integration")
	}

	return parsed, nil
}

func normalizeLegacyBaseURL(raw string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse api_url: %w", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("api_url must include scheme and host")
	}

	trimmedPath := strings.TrimRight(baseURL.Path, "/")
	if connectorPath, ok := siteManagerConnectorPath(trimmedPath); ok {
		// The Site Manager cloud connector forwards to the console's Network application itself,
		// so the legacy API sits directly below it; /proxy/network only exists on the console.
		baseURL.Path = joinURLPath(connectorPath, "api")
		return baseURL, nil
	}

	switch {
	case strings.HasSuffix(trimmedPath, "/proxy/network/api"):
		baseURL.Path = trimmedPath
	case strings.HasSuffix(trimmedPath, "/proxy/network"):
		baseURL.Path = joinURLPath(trimmedPath, "api")
	case strings.HasSuffix(trimmedPath, "/integration"):
		basePath := strings.TrimSuffix(trimmedPath, "/integration")
		if strings.HasSuffix(basePath, "/proxy/network") {
			baseURL.Path = joinURLPath(basePath, "api")
		} else {
			baseURL.Path = joinURLPath(basePath, "proxy", "network", "api")
		}
	default:
		baseURL.Path = joinURLPath(trimmedPath, "proxy", "network", "api")
	}

	return baseURL, nil
}

// siteManagerConnectorRE matches a UniFi Site Manager cloud connector path for the Network
// application, e.g. /v1/connector/consoles/<console-id>/network, optionally followed by /integration
// (the integration API's base below the connector).
var siteManagerConnectorRE = regexp.MustCompile(`^(.*/v1/connector/consoles/[^/]+/network)(?:/integration)?$`)

// siteManagerConnectorPath returns the connector's Network application path if path is one.
func siteManagerConnectorPath(path string) (string, bool) {
	match := siteManagerConnectorRE.FindStringSubmatch(path)
	if match == nil {
		return "", false
	}
	return match[1], true
}

func joinURLPath(basePath string, pathElements ...string) string {
	parts := make([]string, 0, len(pathElements)+1)
	trimmedBasePath := strings.TrimRight(basePath, "/")
	if trimmedBasePath != "" {
		parts = append(parts, trimmedBasePath)
	}
	parts = append(parts, pathElements...)

	joined := path.Join(parts...)
	if joined == "." {
		joined = ""
	}
	if joined == "" {
		return ""
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}

	return joined
}

func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	var sites []Site
	offset := 0

	for {
		response, err := c.apiClient.GetSiteOverviewPageWithResponse(ctx, &generated.GetSiteOverviewPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list sites: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]Site](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate site page: %w", err)
		}

		sites = append(sites, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return sites, nil
}
