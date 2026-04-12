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
