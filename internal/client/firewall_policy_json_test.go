package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFirewallPolicyTrafficFilterMarshalJSONMACAddressVariants(t *testing.T) {
	t.Parallel()

	singleMAC := "AA:BB:CC:DD:EE:FF"
	testCases := []struct {
		name   string
		filter FirewallPolicyTrafficFilter
		want   string
	}{
		{
			name: "single mac address",
			filter: FirewallPolicyTrafficFilter{
				Type:       "NETWORK",
				MACAddress: &singleMAC,
				NetworkFilter: &FirewallPolicyNetworkFilter{
					NetworkIDs:    []string{"00000000-0000-0000-0000-000000000111"},
					MatchOpposite: true,
				},
			},
			want: `{"macAddressFilter":"AA:BB:CC:DD:EE:FF","networkFilter":{"networkIds":["00000000-0000-0000-0000-000000000111"],"matchOpposite":true},"type":"NETWORK"}`,
		},
		{
			name: "mac address list",
			filter: FirewallPolicyTrafficFilter{
				Type: "MAC_ADDRESS",
				MACAddressFilter: &FirewallPolicyMACAddressListFilter{
					MacAddresses: []string{"AA:BB:CC:DD:EE:FF", "11:22:33:44:55:66"},
				},
			},
			want: `{"macAddressFilter":{"macAddresses":["AA:BB:CC:DD:EE:FF","11:22:33:44:55:66"]},"type":"MAC_ADDRESS"}`,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(testCase.filter)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if !jsonEqual(got, []byte(testCase.want)) {
				t.Fatalf("json.Marshal() = %s, want %s", got, testCase.want)
			}
		})
	}
}

func TestFirewallPolicyTrafficFilterUnmarshalJSONMACAddressVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		payload        string
		wantType       string
		wantMACAddress *string
		wantMACList    []string
	}{
		{
			name:           "single mac address",
			payload:        `{"type":"IP_ADDRESS","macAddressFilter":"AA:BB:CC:DD:EE:FF","ipAddressFilter":{"type":"IP_ADDRESSES","matchOpposite":false,"items":[{"type":"IP_ADDRESS","value":"10.0.0.10"}]}}`,
			wantType:       "IP_ADDRESS",
			wantMACAddress: stringPtr("AA:BB:CC:DD:EE:FF"),
		},
		{
			name:        "mac address list",
			payload:     `{"type":"MAC_ADDRESS","macAddressFilter":{"macAddresses":["AA:BB:CC:DD:EE:FF","11:22:33:44:55:66"]}}`,
			wantType:    "MAC_ADDRESS",
			wantMACList: []string{"AA:BB:CC:DD:EE:FF", "11:22:33:44:55:66"},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got FirewallPolicyTrafficFilter
			if err := json.Unmarshal([]byte(testCase.payload), &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if got.Type != testCase.wantType {
				t.Fatalf("Type = %q, want %q", got.Type, testCase.wantType)
			}

			switch {
			case testCase.wantMACAddress != nil:
				if got.MACAddress == nil || *got.MACAddress != *testCase.wantMACAddress {
					t.Fatalf("MACAddress = %#v, want %#v", got.MACAddress, testCase.wantMACAddress)
				}
				if got.MACAddressFilter != nil {
					t.Fatalf("MACAddressFilter = %#v, want nil", got.MACAddressFilter)
				}
			default:
				if got.MACAddress != nil {
					t.Fatalf("MACAddress = %#v, want nil", got.MACAddress)
				}
				if got.MACAddressFilter == nil {
					t.Fatalf("MACAddressFilter = nil, want list")
				}
				if len(got.MACAddressFilter.MacAddresses) != len(testCase.wantMACList) {
					t.Fatalf("MACAddressFilter length = %d, want %d", len(got.MACAddressFilter.MacAddresses), len(testCase.wantMACList))
				}
				for index, want := range testCase.wantMACList {
					if got.MACAddressFilter.MacAddresses[index] != want {
						t.Fatalf("MACAddressFilter[%d] = %q, want %q", index, got.MACAddressFilter.MacAddresses[index], want)
					}
				}
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}

func jsonEqual(left, right []byte) bool {
	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}

	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}

	return reflect.DeepEqual(leftValue, rightValue)
}
