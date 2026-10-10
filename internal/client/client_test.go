package client

import (
	"testing"
	"time"
)

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "root", in: "https://controller.example.com", want: "https://controller.example.com/integration"},
		{name: "with path", in: "https://controller.example.com/proxy/network", want: "https://controller.example.com/proxy/network/integration"},
		{name: "already integration", in: "https://controller.example.com/proxy/network/integration", want: "https://controller.example.com/proxy/network/integration"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBaseURL(testCase.in)
			if err != nil {
				t.Fatalf("normalizeBaseURL() error = %v", err)
			}

			if got.String() != testCase.want {
				t.Fatalf("normalizeBaseURL() = %q, want %q", got.String(), testCase.want)
			}
		})
	}
}

func TestNormalizeLegacyBaseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "root", in: "https://controller.example.com", want: "https://controller.example.com/proxy/network/api"},
		{name: "with proxy network path", in: "https://controller.example.com/proxy/network", want: "https://controller.example.com/proxy/network/api"},
		{name: "with proxy network api path", in: "https://controller.example.com/proxy/network/api", want: "https://controller.example.com/proxy/network/api"},
		{name: "from integration path", in: "https://controller.example.com/integration", want: "https://controller.example.com/proxy/network/api"},
		{name: "from proxy network integration path", in: "https://controller.example.com/proxy/network/integration", want: "https://controller.example.com/proxy/network/api"},
		{name: "with reverse proxy prefix", in: "https://controller.example.com/unifi/proxy/network", want: "https://controller.example.com/unifi/proxy/network/api"},
		{name: "with reverse proxy prefix from integration path", in: "https://controller.example.com/unifi/proxy/network/integration", want: "https://controller.example.com/unifi/proxy/network/api"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeLegacyBaseURL(testCase.in)
			if err != nil {
				t.Fatalf("normalizeLegacyBaseURL() error = %v", err)
			}

			if got.String() != testCase.want {
				t.Fatalf("normalizeLegacyBaseURL() = %q, want %q", got.String(), testCase.want)
			}
		})
	}
}

func TestNewRequestTimeout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
		wantErr  bool
	}{
		{name: "zero selects default", timeout: 0, expected: DefaultRequestTimeout},
		{name: "explicit value is applied", timeout: 5 * time.Minute, expected: 5 * time.Minute},
		{name: "negative is rejected", timeout: -1 * time.Second, wantErr: true},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			apiClient, err := New(Config{
				BaseURL:        "https://unifi.example.com",
				APIKey:         "key",
				RequestTimeout: testCase.timeout,
			})
			if testCase.wantErr {
				if err == nil {
					t.Fatal("New() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if apiClient.httpClient.Timeout != testCase.expected {
				t.Fatalf("httpClient.Timeout = %v, want %v", apiClient.httpClient.Timeout, testCase.expected)
			}
		})
	}
}
