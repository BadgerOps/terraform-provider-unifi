package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

func TestValidateWifiRadiusConfigurationNASID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		nasType string
		source  attr.Value
		value   attr.Value
		valid   bool
	}{
		{name: "derived source", nasType: "DERIVED", source: types.StringValue("DEVICE_NAME"), value: types.StringNull(), valid: true},
		{name: "derived value", nasType: "DERIVED", source: types.StringNull(), value: types.StringValue("wifi"), valid: false},
		{name: "derived missing source", nasType: "DERIVED", source: types.StringNull(), value: types.StringNull(), valid: false},
		{name: "user-defined value", nasType: "USER_DEFINED", source: types.StringNull(), value: types.StringValue("wifi"), valid: true},
		{name: "user-defined source", nasType: "USER_DEFINED", source: types.StringValue("SITE_NAME"), value: types.StringNull(), valid: false},
		{name: "user-defined missing value", nasType: "USER_DEFINED", source: types.StringNull(), value: types.StringNull(), valid: false},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			nasID, diagnostics := types.ObjectValue(wifiNASIDConfigurationAttrTypes(), map[string]attr.Value{
				"type": types.StringValue(testCase.nasType), "source": testCase.source, "value": testCase.value,
			})
			if diagnostics.HasError() {
				t.Fatalf("construct NAS-ID: %v", diagnostics)
			}
			radius, diagnostics := types.ObjectValue(wifiRadiusConfigurationAttrTypes(), map[string]attr.Value{
				"profile_id":                       types.StringValue("00000000-0000-0000-0000-000000000101"),
				"nas_id":                           nasID,
				"mac_authentication_configuration": types.ObjectNull(wifiRadiusMACAuthenticationConfigurationAttrTypes()),
			})
			if diagnostics.HasError() {
				t.Fatalf("construct RADIUS configuration: %v", diagnostics)
			}
			err := validateWifiRadiusConfiguration(context.Background(), radius)
			if testCase.valid && err != nil {
				t.Fatalf("validateWifiRadiusConfiguration() error = %v", err)
			}
			if !testCase.valid && err == nil {
				t.Fatal("validateWifiRadiusConfiguration() error = nil, want error")
			}
		})
	}
}

func TestValidateWifiRadiusConfigurationEnums(t *testing.T) {
	t.Parallel()

	for _, source := range []string{"DEVICE_MAC_ADDRESS", "DEVICE_NAME", "SITE_NAME", "BSSID"} {
		nasID, _ := types.ObjectValue(wifiNASIDConfigurationAttrTypes(), map[string]attr.Value{
			"type": types.StringValue("DERIVED"), "source": types.StringValue(source), "value": types.StringNull(),
		})
		macAuth, _ := types.ObjectValue(wifiRadiusMACAuthenticationConfigurationAttrTypes(), map[string]attr.Value{
			"mac_address_format": types.StringValue("LOWERCASE_COLON_SEPARATED"),
		})
		radius, _ := types.ObjectValue(wifiRadiusConfigurationAttrTypes(), map[string]attr.Value{
			"profile_id": types.StringValue("00000000-0000-0000-0000-000000000101"), "nas_id": nasID, "mac_authentication_configuration": macAuth,
		})
		if err := validateWifiRadiusConfiguration(context.Background(), radius); err != nil {
			t.Fatalf("source %s rejected: %v", source, err)
		}
	}
}

func TestValidateWifiSecurityMode(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"DEFAULT", "HIGH_SECURITY_192_BIT"} {
		if err := validateWifiSecurityMode(types.StringValue(value)); err != nil {
			t.Fatalf("validateWifiSecurityMode(%q) error = %v", value, err)
		}
	}
	if err := validateWifiSecurityMode(types.StringValue("UNKNOWN")); err == nil {
		t.Fatal("validateWifiSecurityMode(UNKNOWN) error = nil, want error")
	}
}

func TestPreserveWifiPresharedKeySecrets(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	networkA, _ := types.ObjectValue(wifiNetworkAttrTypes(), map[string]attr.Value{
		"type": types.StringValue("SPECIFIC"), "network_id": types.StringValue("network-a"),
	})
	networkB, _ := types.ObjectValue(wifiNetworkAttrTypes(), map[string]attr.Value{
		"type": types.StringValue("SPECIFIC"), "network_id": types.StringValue("network-b"),
	})
	priorKeys, diagnostics := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}, []wifiPresharedKeyModel{
		{Passphrase: types.StringValue("secret-a"), Network: networkA},
		{Passphrase: types.StringValue("secret-b"), Network: networkB},
	})
	if diagnostics.HasError() {
		t.Fatalf("construct prior PPSKs: %v", diagnostics)
	}
	priorSecurity, diagnostics := types.ObjectValueFrom(ctx, wifiSecurityConfigurationAttrTypes(), wifiSecurityConfigurationModel{
		Type: types.StringValue("WPA2_PERSONAL"), Passphrase: types.StringNull(), Encryption: types.StringNull(),
		PMFMode: types.StringNull(), FastRoamingEnabled: types.BoolNull(), GroupRekeyIntervalSeconds: types.Int64Null(),
		WPA3FastRoamingEnabled: types.BoolNull(), SAEConfiguration: types.ObjectNull(wifiSAEConfigurationAttrTypes()),
		RadiusConfiguration: types.ObjectNull(wifiRadiusConfigurationAttrTypes()), CoAEnabled: types.BoolNull(),
		SecurityMode: types.StringNull(), PresharedKeys: priorKeys,
	})
	if diagnostics.HasError() {
		t.Fatalf("construct prior security configuration: %v", diagnostics)
	}
	security := &client.WifiSecurityConfiguration{
		Type: "WPA2_PERSONAL",
		PresharedKeys: []client.WifiPresharedKey{
			{Network: client.WifiNetworkReference{Type: "SPECIFIC", NetworkID: "network-b"}},
			{Network: client.WifiNetworkReference{Type: "SPECIFIC", NetworkID: "network-a"}},
		},
	}
	preserveWifiSecuritySecrets(ctx, security, priorSecurity, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("preserveWifiSecuritySecrets() diagnostics = %v", diagnostics)
	}
	if security.PresharedKeys[0].Passphrase == nil || *security.PresharedKeys[0].Passphrase != "secret-b" {
		t.Fatalf("network-b passphrase = %#v, want secret-b", security.PresharedKeys[0].Passphrase)
	}
	if security.PresharedKeys[1].Passphrase == nil || *security.PresharedKeys[1].Passphrase != "secret-a" {
		t.Fatalf("network-a passphrase = %#v, want secret-a", security.PresharedKeys[1].Passphrase)
	}
}
