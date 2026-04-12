package generated

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeneratedClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/sites":
			if request.Header.Get("X-API-KEY") != "test-key" {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			writeGeneratedJSON(t, writer, map[string]any{
				"offset":     0,
				"limit":      100,
				"count":      1,
				"totalCount": 1,
				"data": []map[string]any{
					{
						"id":                "11111111-1111-1111-1111-111111111111",
						"internalReference": "default",
						"name":              "Default",
					},
				},
			})
		case "/v1/sites/11111111-1111-1111-1111-111111111111/networks":
			if request.Header.Get("X-API-KEY") != "test-key" {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			writeGeneratedJSON(t, writer, map[string]any{
				"offset":     0,
				"limit":      100,
				"count":      1,
				"totalCount": 1,
				"data": []map[string]any{
					{
						"id":         "22222222-2222-2222-2222-222222222222",
						"management": "GATEWAY",
						"name":       "trusted",
						"enabled":    true,
						"default":    false,
						"vlanId":     20,
						"metadata": map[string]any{
							"origin": "USER",
						},
					},
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	apiClient, err := NewClientWithResponses(server.URL, WithRequestEditorFn(func(_ context.Context, request *http.Request) error {
		request.Header.Set("X-API-KEY", "test-key")
		return nil
	}))
	if err != nil {
		t.Fatalf("new generated client: %v", err)
	}

	siteResponse, err := apiClient.GetSiteOverviewPageWithResponse(context.Background(), nil)
	if err != nil {
		t.Fatalf("get sites with generated client: %v", err)
	}
	if siteResponse.JSON200 == nil || len(siteResponse.JSON200.Data) != 1 {
		t.Fatalf("expected one site in generated response")
	}

	networkResponse, err := apiClient.GetNetworksOverviewPageWithResponse(context.Background(), siteResponse.JSON200.Data[0].Id, nil)
	if err != nil {
		t.Fatalf("get networks with generated client: %v", err)
	}
	if networkResponse.JSON200 == nil || len(networkResponse.JSON200.Data) != 1 {
		t.Fatalf("expected one network in generated response")
	}
	if networkResponse.JSON200.Data[0].Name != "trusted" {
		t.Fatalf("unexpected generated network name: %s", networkResponse.JSON200.Data[0].Name)
	}
}

func writeGeneratedJSON(t *testing.T, writer http.ResponseWriter, payload any) {
	t.Helper()

	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		t.Fatalf("encode generated test payload: %v", err)
	}
}
