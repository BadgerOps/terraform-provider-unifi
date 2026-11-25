package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type legacyResponse[T any] struct {
	Data T `json:"data"`
}

func (c *Client) resolveLegacySiteReference(ctx context.Context, siteID string) (string, error) {
	sites, err := c.ListSites(ctx)
	if err != nil {
		return "", err
	}

	for _, site := range sites {
		if site.ID == siteID {
			return site.InternalReference, nil
		}
	}

	return "", fmt.Errorf("site %s not found", siteID)
}

func (c *Client) doLegacyRequest(ctx context.Context, method string, pathElements []string, payload any, target any) error {
	return c.doLegacyRequestWithExpectedStatus(ctx, method, pathElements, payload, target, http.StatusOK)
}

func (c *Client) doLegacyRequestWithExpectedStatus(
	ctx context.Context,
	method string,
	pathElements []string,
	payload any,
	target any,
	expectedStatusCodes ...int,
) error {
	endpoint := *c.legacyBaseURL
	endpoint.Path = joinURLPath(c.legacyBaseURL.Path, pathElements...)
	response, body, err := c.sendAPIRequest(ctx, method, endpoint, payload, "legacy")
	if err != nil {
		return err
	}

	// Legacy failures can arrive with HTTP 200. Check meta even when callers
	// ignore the data (PUT/DELETE), including error envelopes returned with 400.
	var envelope struct {
		Meta struct {
			RC      string `json:"rc"`
			Message string `json:"msg"`
		} `json:"meta"`
		Data json.RawMessage `json:"data"`
	}
	decodeErr := json.Unmarshal(body, &envelope)
	if decodeErr == nil && envelope.Meta.RC != "" && envelope.Meta.RC != "ok" {
		status := response.StatusCode
		if status == http.StatusOK || status == http.StatusBadRequest {
			switch envelope.Meta.Message {
			case "api.err.IdInvalid", "api.err.NotFound":
				status = http.StatusNotFound
			}
		}
		if status >= 200 && status < 300 {
			status = http.StatusBadRequest
		}
		return &Error{StatusCode: status, Code: envelope.Meta.RC, Message: envelope.Meta.Message, Body: string(body)}
	}
	if err := requireStatus(response.StatusCode, body, expectedStatusCodes...); err != nil {
		return err
	}
	// Callers that ignore the response (PUT/DELETE acknowledgements) must not
	// fail on an empty or non-JSON 2xx body; the status and envelope checks above
	// already caught reported failures.
	if target == nil {
		return nil
	}
	if len(body) == 0 {
		return fmt.Errorf("empty legacy response body")
	}
	if decodeErr != nil {
		return fmt.Errorf("decode legacy response envelope: %w", decodeErr)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode legacy response body: %w", err)
	}

	return nil
}

// unmarshalWithRawFields decodes data into typed and also returns every field
// as raw JSON, so a read-modify-write can send back the fields the provider
// does not model. RawMessage keeps numbers exactly as the controller sent them.
func unmarshalWithRawFields(data []byte, typed any) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, typed); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// sendAPIRequest sends one request to a controller API (legacy or v2) with the
// provider's authentication and headers, and returns the response with its
// whole body. The caller decides what the status and body mean.
func (c *Client) sendAPIRequest(ctx context.Context, method string, endpoint url.URL, payload any, api string) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		var err error
		bodyReader, err = jsonBodyReader(payload)
		if err != nil {
			return nil, nil, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("build %s request: %w", api, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-API-KEY", c.apiKey)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s response body: %w", api, err)
	}
	return response, body, nil
}
