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

func wifiSecurityWithField(t *testing.T, field string, value attr.Value) types.Object {
	t.Helper()

	attributes := map[string]attr.Value{
		"type":                         types.StringValue("WPA2_PERSONAL"),
		"passphrase":                   types.StringValue("acceptance-passphrase"),
		"encryption":                   types.StringNull(),
		"pmf_mode":                     types.StringNull(),
		"fast_roaming_enabled":         types.BoolNull(),
		"group_rekey_interval_seconds": types.Int64Null(),
		"wpa3_fast_roaming_enabled":    types.BoolNull(),
		"sae_configuration":            types.ObjectNull(wifiSAEConfigurationAttrTypes()),
		"radius_configuration":         types.ObjectNull(wifiRadiusConfigurationAttrTypes()),
		"coa_enabled":                  types.BoolNull(),
		"security_mode":                types.StringNull(),
		"preshared_keys":               types.ListNull(types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}),
	}
	if field != "" {
		attributes[field] = value
	}

	security, diagnostics := types.ObjectValue(wifiSecurityConfigurationAttrTypes(), attributes)
	if diagnostics.HasError() {
		t.Fatalf("construct security configuration: %v", diagnostics)
	}
	return security
}

func wifiNativeNetwork(t *testing.T) types.Object {
	t.Helper()

	network, diagnostics := types.ObjectValue(wifiNetworkAttrTypes(), map[string]attr.Value{
		"type": types.StringValue("NATIVE"), "network_id": types.StringNull(),
	})
	if diagnostics.HasError() {
		t.Fatalf("construct network: %v", diagnostics)
	}
	return network
}

func TestValidateWifiBroadcastModelIoTUnsupportedSecurityFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		field string
		value attr.Value
	}{
		{name: "pmf_mode", field: "pmf_mode", value: types.StringValue("OPTIONAL")},
		{name: "fast_roaming_enabled", field: "fast_roaming_enabled", value: types.BoolValue(true)},
		{name: "group_rekey_interval_seconds", field: "group_rekey_interval_seconds", value: types.Int64Value(3600)},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			plan := wifiBroadcastResourceModel{
				Type:                  types.StringValue("IOT_OPTIMIZED"),
				Network:               wifiNativeNetwork(t),
				SecurityConfiguration: wifiSecurityWithField(t, testCase.field, testCase.value),
			}

			err := validateWifiBroadcastModel(context.Background(), plan)
			if err == nil {
				t.Fatalf("validateWifiBroadcastModel() error = nil, want error for %s", testCase.field)
			}
			expected := "security_configuration." + testCase.field + " is not valid for IOT_OPTIMIZED broadcasts"
			if err.Error() != expected {
				t.Fatalf("validateWifiBroadcastModel() error = %q, want %q", err.Error(), expected)
			}
		})
	}
}

func TestValidateWifiBroadcastModelIoTAllowsSupportedSecurity(t *testing.T) {
	t.Parallel()

	plan := wifiBroadcastResourceModel{
		Type:                  types.StringValue("IOT_OPTIMIZED"),
		Network:               wifiNativeNetwork(t),
		SecurityConfiguration: wifiSecurityWithField(t, "", nil),
	}

	if err := validateWifiBroadcastModel(context.Background(), plan); err != nil {
		t.Fatalf("validateWifiBroadcastModel() error = %v", err)
	}
}

func TestValidateWifiBroadcastModelStandardAllowsIoTUnsupportedSecurityFields(t *testing.T) {
	t.Parallel()

	frequencies, diagnostics := types.SetValue(types.Float64Type, []attr.Value{types.Float64Value(2.4)})
	if diagnostics.HasError() {
		t.Fatalf("construct broadcasting frequencies: %v", diagnostics)
	}

	for _, testCase := range []struct {
		field string
		value attr.Value
	}{
		{field: "pmf_mode", value: types.StringValue("OPTIONAL")},
		{field: "fast_roaming_enabled", value: types.BoolValue(true)},
		{field: "group_rekey_interval_seconds", value: types.Int64Value(3600)},
	} {
		plan := wifiBroadcastResourceModel{
			Type:                       types.StringValue("STANDARD"),
			Network:                    wifiNativeNetwork(t),
			SecurityConfiguration:      wifiSecurityWithField(t, testCase.field, testCase.value),
			BroadcastingFrequenciesGHz: frequencies,
			AdvertiseDeviceName:        types.BoolValue(true),
			ARPProxyEnabled:            types.BoolValue(false),
			BSSTransitionEnabled:       types.BoolValue(true),
		}

		if err := validateWifiBroadcastModel(context.Background(), plan); err != nil {
			t.Fatalf("validateWifiBroadcastModel() error = %v for %s", err, testCase.field)
		}
	}
}

func wifiPresharedKeyList(t *testing.T) types.List {
	t.Helper()

	network, diagnostics := types.ObjectValue(wifiNetworkAttrTypes(), map[string]attr.Value{
		"type": types.StringValue("SPECIFIC"), "network_id": types.StringValue("00000000-0000-0000-0000-000000000201"),
	})
	if diagnostics.HasError() {
		t.Fatalf("construct preshared key network: %v", diagnostics)
	}
	key, diagnostics := types.ObjectValue(wifiPresharedKeyAttrTypes(), map[string]attr.Value{
		"passphrase": types.StringValue("preshared-passphrase"), "network": network,
	})
	if diagnostics.HasError() {
		t.Fatalf("construct preshared key: %v", diagnostics)
	}
	list, diagnostics := types.ListValue(types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}, []attr.Value{key})
	if diagnostics.HasError() {
		t.Fatalf("construct preshared keys: %v", diagnostics)
	}
	return list
}

func wifiPresharedKeySecurity(t *testing.T, withPassphrase bool) types.Object {
	t.Helper()

	attributes := map[string]attr.Value{
		"type":                         types.StringValue("WPA2_PERSONAL"),
		"passphrase":                   types.StringNull(),
		"encryption":                   types.StringNull(),
		"pmf_mode":                     types.StringNull(),
		"fast_roaming_enabled":         types.BoolValue(true),
		"group_rekey_interval_seconds": types.Int64Null(),
		"wpa3_fast_roaming_enabled":    types.BoolNull(),
		"sae_configuration":            types.ObjectNull(wifiSAEConfigurationAttrTypes()),
		"radius_configuration":         types.ObjectNull(wifiRadiusConfigurationAttrTypes()),
		"coa_enabled":                  types.BoolNull(),
		"security_mode":                types.StringNull(),
		"preshared_keys":               wifiPresharedKeyList(t),
	}
	if withPassphrase {
		attributes["passphrase"] = types.StringValue("acceptance-passphrase")
	}

	security, diagnostics := types.ObjectValue(wifiSecurityConfigurationAttrTypes(), attributes)
	if diagnostics.HasError() {
		t.Fatalf("construct PPSK security configuration: %v", diagnostics)
	}
	return security
}

func wifiStandardPlan(network types.Object, security types.Object) wifiBroadcastResourceModel {
	frequencies := types.SetValueMust(types.Float64Type, []attr.Value{types.Float64Value(2.4)})
	return wifiBroadcastResourceModel{
		Type:                       types.StringValue("STANDARD"),
		Network:                    network,
		SecurityConfiguration:      security,
		BroadcastingFrequenciesGHz: frequencies,
		AdvertiseDeviceName:        types.BoolValue(true),
		ARPProxyEnabled:            types.BoolValue(false),
		BSSTransitionEnabled:       types.BoolValue(true),
	}
}

func TestValidateWifiBroadcastModelPresharedKeysForbidNetwork(t *testing.T) {
	t.Parallel()

	plan := wifiStandardPlan(wifiNativeNetwork(t), wifiPresharedKeySecurity(t, false))

	err := validateWifiBroadcastModel(context.Background(), plan)
	if err == nil {
		t.Fatal("validateWifiBroadcastModel() error = nil, want error")
	}
	expected := "network must not be set when security_configuration.preshared_keys is set, because each preshared key carries its own network"
	if err.Error() != expected {
		t.Fatalf("validateWifiBroadcastModel() error = %q, want %q", err.Error(), expected)
	}
}

func TestValidateWifiBroadcastModelPresharedKeysWithoutNetwork(t *testing.T) {
	t.Parallel()

	plan := wifiStandardPlan(types.ObjectNull(wifiNetworkAttrTypes()), wifiPresharedKeySecurity(t, false))

	if err := validateWifiBroadcastModel(context.Background(), plan); err != nil {
		t.Fatalf("validateWifiBroadcastModel() error = %v", err)
	}
}

func TestValidateWifiBroadcastModelPresharedKeysForbidPassphrase(t *testing.T) {
	t.Parallel()

	plan := wifiStandardPlan(types.ObjectNull(wifiNetworkAttrTypes()), wifiPresharedKeySecurity(t, true))

	err := validateWifiBroadcastModel(context.Background(), plan)
	if err == nil {
		t.Fatal("validateWifiBroadcastModel() error = nil, want error")
	}
	expected := "security_configuration.passphrase and security_configuration.preshared_keys are mutually exclusive"
	if err.Error() != expected {
		t.Fatalf("validateWifiBroadcastModel() error = %q, want %q", err.Error(), expected)
	}
}

func TestValidateWifiBroadcastModelNetworkRequiredWithoutPresharedKeys(t *testing.T) {
	t.Parallel()

	plan := wifiStandardPlan(types.ObjectNull(wifiNetworkAttrTypes()), wifiSecurityWithField(t, "", nil))

	err := validateWifiBroadcastModel(context.Background(), plan)
	if err == nil {
		t.Fatal("validateWifiBroadcastModel() error = nil, want error")
	}
	expected := "network is required unless security_configuration.preshared_keys is set"
	if err.Error() != expected {
		t.Fatalf("validateWifiBroadcastModel() error = %q, want %q", err.Error(), expected)
	}
}
