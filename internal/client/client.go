package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultPageLimit = 100

type Config struct {
	BaseURL       string
	APIKey        string
	AllowInsecure bool
	UserAgent     string
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	userAgent  string
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

	baseURL, err := normalizeBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.AllowInsecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		userAgent: config.UserAgent,
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
		parsed.Path = path.Join(trimmedPath, "integration")
		if !strings.HasPrefix(parsed.Path, "/") {
			parsed.Path = "/" + parsed.Path
		}
	}

	return parsed, nil
}

func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	var sites []Site
	offset := 0

	for {
		var response page[Site]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, "/v1/sites", query, nil, &response); err != nil {
			return nil, err
		}

		sites = append(sites, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return sites, nil
}

func (c *Client) do(ctx context.Context, method, requestPath string, query url.Values, body any, out any) error {
	requestURL := *c.baseURL
	requestURL.Path = path.Join(c.baseURL.Path, requestPath)
	requestURL.RawQuery = query.Encode()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-API-KEY", c.apiKey)
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, requestURL.String(), err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		var apiErr apiError
		if err := json.Unmarshal(payload, &apiErr); err == nil && (apiErr.Code != "" || apiErr.Message != "") {
			return &Error{
				StatusCode: response.StatusCode,
				Code:       apiErr.Code,
				Message:    apiErr.Message,
				Body:       string(payload),
			}
		}

		return &Error{
			StatusCode: response.StatusCode,
			Body:       string(payload),
		}
	}

	if out == nil || len(payload) == 0 {
		return nil
	}

	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}
