package client

import (
	"encoding/json"
	"testing"
)

func TestWifiSecurityConfigurationMarshalEnterprise(t *testing.T) {
	source := "DEVICE_NAME"
	configuration := WifiSecurityConfiguration{
		Type: "WPA2_WPA3_ENTERPRISE",
		RadiusConfiguration: &WifiRadiusConfiguration{
			ProfileID: "00000000-0000-0000-0000-000000000101",
			NASID:     WifiNASIDConfiguration{Type: "DERIVED", Source: &source},
			MACAuthenticationConfiguration: &WifiRadiusMACAuthenticationConfiguration{
				MACAddressFormat: "LOWERCASE_COLON_SEPARATED",
			},
		},
		CoAEnabled:             wifiBoolPtr(true),
		PMFMode:                stringPtr("OPTIONAL"),
		FastRoamingEnabled:     wifiBoolPtr(true),
		WPA3FastRoamingEnabled: wifiBoolPtr(true),
	}

	got, err := json.Marshal(configuration)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	want := []byte(`{"type":"WPA2_WPA3_ENTERPRISE","pmfMode":"OPTIONAL","fastRoamingEnabled":true,"wpa3FastRoamingEnabled":true,"radiusConfiguration":{"profileId":"00000000-0000-0000-0000-000000000101","nasId":{"type":"DERIVED","source":"DEVICE_NAME"},"macAuthenticationConfiguration":{"macAddressFormat":"LOWERCASE_COLON_SEPARATED"}},"coaEnabled":true}`)
	if !jsonEqual(got, want) {
		t.Fatalf("json.Marshal() = %s, want %s", got, want)
	}
}

func wifiBoolPtr(value bool) *bool {
	return &value
}

func TestWifiSecurityConfigurationMarshalPresharedKeys(t *testing.T) {
	configuration := WifiSecurityConfiguration{
		Type: "WPA2_PERSONAL",
		PresharedKeys: []WifiPresharedKey{
			{Passphrase: stringPtr("network-a-secret"), Network: WifiNetworkReference{Type: "SPECIFIC", NetworkID: "00000000-0000-0000-0000-000000000201"}},
			{Passphrase: stringPtr("network-b-secret"), Network: WifiNetworkReference{Type: "SPECIFIC", NetworkID: "00000000-0000-0000-0000-000000000202"}},
		},
	}

	got, err := json.Marshal(configuration)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	want := []byte(`{"type":"WPA2_PERSONAL","presharedKeys":[{"passphrase":"network-a-secret","network":{"type":"SPECIFIC","networkId":"00000000-0000-0000-0000-000000000201"}},{"passphrase":"network-b-secret","network":{"type":"SPECIFIC","networkId":"00000000-0000-0000-0000-000000000202"}}]}`)
	if !jsonEqual(got, want) {
		t.Fatalf("json.Marshal() = %s, want %s", got, want)
	}
}
