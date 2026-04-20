package client

import "testing"

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
		{name: "with reverse proxy prefix", in: "https://controller.example.com/unifi/proxy/network", want: "https://controller.example.com/unifi/proxy/network/api"},
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
