package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

const defaultPageLimit = 100

type Config struct {
	BaseURL       string
	APIKey        string
	AllowInsecure bool
	UserAgent     string
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

	integrationBaseURL, err := normalizeBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	legacyBaseURL, err := normalizeLegacyBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.AllowInsecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
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
	switch {
	case strings.HasSuffix(trimmedPath, "/proxy/network/api"):
		baseURL.Path = trimmedPath
	case strings.HasSuffix(trimmedPath, "/proxy/network"):
		baseURL.Path = joinURLPath(trimmedPath, "api")
	case strings.HasSuffix(trimmedPath, "/integration"):
		baseURL.Path = joinURLPath(strings.TrimSuffix(trimmedPath, "/integration"), "proxy", "network", "api")
	default:
		baseURL.Path = joinURLPath(trimmedPath, "proxy", "network", "api")
	}

	return baseURL, nil
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
