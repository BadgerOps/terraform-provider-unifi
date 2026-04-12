package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func parseUUID(raw string) (openapi_types.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return openapi_types.UUID{}, fmt.Errorf("parse uuid %q: %w", raw, err)
	}

	return parsed, nil
}

func pageParam(value int) *int32 {
	result := int32(value)
	return &result
}

func requireStatus(status int, body []byte, expected ...int) error {
	for _, candidate := range expected {
		if status == candidate {
			return nil
		}
	}

	if status >= http.StatusBadRequest {
		var apiErr apiError
		if err := json.Unmarshal(body, &apiErr); err == nil && (apiErr.Code != "" || apiErr.Message != "") {
			return &Error{
				StatusCode: status,
				Code:       apiErr.Code,
				Message:    apiErr.Message,
				Body:       string(body),
			}
		}

		return &Error{
			StatusCode: status,
			Body:       string(body),
		}
	}

	return fmt.Errorf("unexpected response status=%d body=%s", status, string(body))
}

func requireJSON[T any](status int, body []byte, value *T, expected ...int) (*T, error) {
	if err := requireStatus(status, body, expected...); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("unexpected empty response body for status=%d", status)
	}

	return value, nil
}

func transcode[T any](input any) (T, error) {
	var output T
	if isNilValue(input) {
		return output, nil
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return output, fmt.Errorf("marshal value: %w", err)
	}
	if string(payload) == "null" {
		return output, nil
	}

	if err := json.Unmarshal(payload, &output); err != nil {
		return output, fmt.Errorf("unmarshal value: %w", err)
	}

	return output, nil
}

func jsonBodyReader(input any) (io.Reader, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	return bytes.NewReader(payload), nil
}

func decodeBody[T any](body []byte) (*T, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("unexpected empty response body")
	}

	var output T
	if err := json.Unmarshal(body, &output); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}

	return &output, nil
}

func isNilValue(input any) bool {
	if input == nil {
		return true
	}

	value := reflect.ValueOf(input)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
