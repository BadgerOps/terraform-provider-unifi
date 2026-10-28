package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// objectWith builds an object of attrTypes with every attribute null except
// those given in values, so a test only spells out what it cares about.
func objectWith(t *testing.T, attrTypes map[string]attr.Type, values map[string]attr.Value) types.Object {
	t.Helper()

	ctx := context.Background()
	attributes := make(map[string]attr.Value, len(attrTypes))
	for name, attrType := range attrTypes {
		value, err := attrType.ValueFromTerraform(ctx, tftypes.NewValue(attrType.TerraformType(ctx), nil))
		if err != nil {
			t.Fatalf("null value for %s: %v", name, err)
		}
		attributes[name] = value
	}
	for name, value := range values {
		if _, ok := attrTypes[name]; !ok {
			t.Fatalf("objectWith: unknown attribute %q", name)
		}
		attributes[name] = value
	}

	object, diagnostics := types.ObjectValue(attrTypes, attributes)
	if diagnostics.HasError() {
		t.Fatalf("construct object: %v", diagnostics)
	}
	return object
}

func TestValidateWifiBroadcastModelUnknownValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		plan func(t *testing.T) wifiBroadcastResourceModel
	}{
		{
			name: "broadcast type unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "", nil))
				plan.Type = types.StringUnknown()
				return plan
			},
		},
		{
			name: "security configuration unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				return wifiStandardPlan(wifiNativeNetwork(t), types.ObjectUnknown(wifiSecurityConfigurationAttrTypes()))
			},
		},
		{
			name: "security type unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				return wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "type", types.StringUnknown()))
			},
		},
		{
			name: "passphrase unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				return wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "passphrase", types.StringUnknown()))
			},
		},
		{
			name: "encryption unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				return wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "encryption", types.StringUnknown()))
			},
		},
		{
			name: "network unknown with preshared keys",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				// false: keys only, no broadcast-level passphrase (true would trip the mutual-exclusion rule).
				return wifiStandardPlan(types.ObjectUnknown(wifiNetworkAttrTypes()), wifiPresharedKeySecurity(t, false))
			},
		},
		{
			name: "network type unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				network := objectWith(t, wifiNetworkAttrTypes(), map[string]attr.Value{"type": types.StringUnknown()})
				return wifiStandardPlan(network, wifiSecurityWithField(t, "", nil))
			},
		},
		{
			name: "preshared key network unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				key := objectWith(t, wifiPresharedKeyAttrTypes(), map[string]attr.Value{
					"passphrase": types.StringValue("acceptance-passphrase"),
					"network":    types.ObjectUnknown(wifiNetworkAttrTypes()),
				})
				keys := types.ListValueMust(types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}, []attr.Value{key})
				security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
					"type":           types.StringValue("WPA2_PERSONAL"),
					"preshared_keys": keys,
				})
				return wifiStandardPlan(types.ObjectNull(wifiNetworkAttrTypes()), security)
			},
		},
		{
			name: "radius nas_id source unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				nasID := objectWith(t, wifiNASIDConfigurationAttrTypes(), map[string]attr.Value{
					"type":   types.StringValue("DERIVED"),
					"source": types.StringUnknown(),
				})
				radius := objectWith(t, wifiRadiusConfigurationAttrTypes(), map[string]attr.Value{
					"profile_id": types.StringValue("00000000-0000-0000-0000-000000000101"),
					"nas_id":     nasID,
				})
				security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
					"type":                 types.StringValue("WPA2_ENTERPRISE"),
					"radius_configuration": radius,
					"coa_enabled":          types.BoolValue(false),
				})
				return wifiStandardPlan(wifiNativeNetwork(t), security)
			},
		},
		{
			name: "iot pmf_mode unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				return wifiBroadcastResourceModel{
					Type:                  types.StringValue("IOT_OPTIMIZED"),
					Network:               wifiNativeNetwork(t),
					SecurityConfiguration: wifiSecurityWithField(t, "pmf_mode", types.StringUnknown()),
				}
			},
		},
		{
			name: "dns assistance mode unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "", nil))
				plan.DNSAssistanceConfiguration = objectWith(t, wifiDNSAssistanceConfigurationAttrTypes(), map[string]attr.Value{"mode": types.StringUnknown()})
				return plan
			},
		},
		{
			name: "device filter type unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "", nil))
				plan.BroadcastingDeviceFilter = objectWith(t, wifiBroadcastingDeviceFilterAttrTypes(), map[string]attr.Value{"type": types.StringUnknown()})
				return plan
			},
		},
		{
			name: "device tag ids unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "", nil))
				plan.BroadcastingDeviceFilter = objectWith(t, wifiBroadcastingDeviceFilterAttrTypes(), map[string]attr.Value{
					"type":           types.StringValue("DEVICE_TAGS"),
					"device_tag_ids": types.SetUnknown(types.StringType),
				})
				return plan
			},
		},
		{
			name: "dns servers unknown with AUTO mode",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "", nil))
				plan.DNSAssistanceConfiguration = objectWith(t, wifiDNSAssistanceConfigurationAttrTypes(), map[string]attr.Value{
					"mode":    types.StringValue("AUTO"),
					"servers": types.ListUnknown(types.StringType),
				})
				return plan
			},
		},
		{
			name: "native network_id unknown",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				network := objectWith(t, wifiNetworkAttrTypes(), map[string]attr.Value{
					"type":       types.StringValue("NATIVE"),
					"network_id": types.StringUnknown(),
				})
				return wifiStandardPlan(network, wifiSecurityWithField(t, "", nil))
			},
		},
		{
			name: "preshared keys list unknown with known network",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
					"type":           types.StringValue("WPA2_PERSONAL"),
					"preshared_keys": types.ListUnknown(types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}),
				})
				return wifiStandardPlan(wifiNativeNetwork(t), security)
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := validateWifiBroadcastModel(context.Background(), testCase.plan(t)); err != nil {
				t.Fatalf("validateWifiBroadcastModel() error = %v, want nil while a value is unknown", err)
			}
		})
	}
}

// The unknown-value guards must not swallow errors for known values.
func TestValidateWifiBroadcastModelKnownInvalidStillRejected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		plan     func(t *testing.T) wifiBroadcastResourceModel
		expected string
	}{
		{
			name: "encryption on WPA2_PERSONAL",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				return wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "encryption", types.StringValue("ENHANCED_OPEN")))
			},
			expected: "security_configuration.encryption is not valid for WPA2_PERSONAL",
		},
		{
			name: "empty device tag ids",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "", nil))
				plan.BroadcastingDeviceFilter = objectWith(t, wifiBroadcastingDeviceFilterAttrTypes(), map[string]attr.Value{
					"type":           types.StringValue("DEVICE_TAGS"),
					"device_tag_ids": types.SetValueMust(types.StringType, []attr.Value{}),
				})
				return plan
			},
			expected: "broadcasting_device_filter.device_tag_ids must contain at least one device tag id when type is DEVICE_TAGS",
		},
		{
			name: "network type invalid",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				network := objectWith(t, wifiNetworkAttrTypes(), map[string]attr.Value{"type": types.StringValue("BOGUS")})
				return wifiStandardPlan(network, wifiSecurityWithField(t, "", nil))
			},
			expected: "network.type must be NATIVE or SPECIFIC",
		},
		{
			name: "native network with known network_id",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				network := objectWith(t, wifiNetworkAttrTypes(), map[string]attr.Value{
					"type":       types.StringValue("NATIVE"),
					"network_id": types.StringValue("00000000-0000-0000-0000-000000000201"),
				})
				return wifiStandardPlan(network, wifiSecurityWithField(t, "", nil))
			},
			expected: "network.network_id must not be set when network.type is NATIVE",
		},
		{
			name: "radius nas_id source bogus",
			plan: func(t *testing.T) wifiBroadcastResourceModel {
				nasID := objectWith(t, wifiNASIDConfigurationAttrTypes(), map[string]attr.Value{
					"type":   types.StringValue("DERIVED"),
					"source": types.StringValue("BOGUS"),
				})
				radius := objectWith(t, wifiRadiusConfigurationAttrTypes(), map[string]attr.Value{
					"profile_id": types.StringValue("00000000-0000-0000-0000-000000000101"),
					"nas_id":     nasID,
				})
				security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
					"type":                 types.StringValue("WPA2_ENTERPRISE"),
					"radius_configuration": radius,
					"coa_enabled":          types.BoolValue(false),
				})
				return wifiStandardPlan(wifiNativeNetwork(t), security)
			},
			expected: "NAS-ID source must be DEVICE_MAC_ADDRESS, DEVICE_NAME, SITE_NAME, or BSSID",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateWifiBroadcastModel(context.Background(), testCase.plan(t))
			if err == nil || err.Error() != testCase.expected {
				t.Fatalf("validateWifiBroadcastModel() error = %v, want %q", err, testCase.expected)
			}
		})
	}
}

func TestWifiBroadcastDeprecations(t *testing.T) {
	t.Parallel()

	openSecurity := func(t *testing.T, field string, value attr.Value) types.Object {
		return objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
			"type": types.StringValue("OPEN"),
			field:  value,
		})
	}
	sae := func(t *testing.T) types.Object {
		return objectWith(t, wifiSAEConfigurationAttrTypes(), map[string]attr.Value{
			"anticlogging_threshold_seconds": types.Int64Value(5),
			"sync_time_seconds":              types.Int64Value(5),
		})
	}

	testCases := []struct {
		name  string
		plan  func(t *testing.T) wifiBroadcastResourceModel
		field string
	}{
		{name: "OPEN pmf_mode", field: "pmf_mode", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), openSecurity(t, "pmf_mode", types.StringValue("OPTIONAL")))
		}},
		{name: "OPEN fast_roaming_enabled", field: "fast_roaming_enabled", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), openSecurity(t, "fast_roaming_enabled", types.BoolValue(false)))
		}},
		{name: "OPEN group_rekey_interval_seconds", field: "group_rekey_interval_seconds", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), openSecurity(t, "group_rekey_interval_seconds", types.Int64Value(3600)))
		}},
		{name: "OPEN wpa3_fast_roaming_enabled", field: "wpa3_fast_roaming_enabled", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), openSecurity(t, "wpa3_fast_roaming_enabled", types.BoolValue(false)))
		}},
		{name: "OPEN sae_configuration", field: "sae_configuration", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), openSecurity(t, "sae_configuration", sae(t)))
		}},
		{name: "WPA2_PERSONAL wpa3_fast_roaming_enabled", field: "wpa3_fast_roaming_enabled", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "wpa3_fast_roaming_enabled", types.BoolValue(false)))
		}},
		{name: "WPA2_PERSONAL sae_configuration", field: "sae_configuration", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "sae_configuration", sae(t)))
		}},
		{name: "WPA3_PERSONAL wpa3_fast_roaming_enabled", field: "wpa3_fast_roaming_enabled", plan: func(t *testing.T) wifiBroadcastResourceModel {
			security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
				"type":                      types.StringValue("WPA3_PERSONAL"),
				"passphrase":                types.StringValue("acceptance-passphrase"),
				"sae_configuration":         sae(t),
				"wpa3_fast_roaming_enabled": types.BoolValue(false),
			})
			return wifiStandardPlan(wifiNativeNetwork(t), security)
		}},
		{name: "IOT_OPTIMIZED pmf_mode", field: "pmf_mode", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiBroadcastResourceModel{Type: types.StringValue("IOT_OPTIMIZED"), Network: wifiNativeNetwork(t), SecurityConfiguration: wifiSecurityWithField(t, "pmf_mode", types.StringValue("OPTIONAL"))}
		}},
		{name: "IOT_OPTIMIZED fast_roaming_enabled", field: "fast_roaming_enabled", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiBroadcastResourceModel{Type: types.StringValue("IOT_OPTIMIZED"), Network: wifiNativeNetwork(t), SecurityConfiguration: wifiSecurityWithField(t, "fast_roaming_enabled", types.BoolValue(true))}
		}},
		{name: "IOT_OPTIMIZED group_rekey_interval_seconds", field: "group_rekey_interval_seconds", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiBroadcastResourceModel{Type: types.StringValue("IOT_OPTIMIZED"), Network: wifiNativeNetwork(t), SecurityConfiguration: wifiSecurityWithField(t, "group_rekey_interval_seconds", types.Int64Value(3600))}
		}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			plan := testCase.plan(t)
			if err := validateWifiBroadcastModel(context.Background(), plan); err != nil {
				t.Fatalf("validateWifiBroadcastModel() error = %v, want nil for a deprecated combination", err)
			}
			deprecations := wifiBroadcastDeprecations(context.Background(), plan)
			if len(deprecations) != 1 || deprecations[0].Field != testCase.field {
				t.Fatalf("wifiBroadcastDeprecations() = %+v, want exactly one for %s", deprecations, testCase.field)
			}
			if !strings.Contains(deprecations[0].Detail, "v"+wifiDeprecationRemovalVersion) {
				t.Fatalf("deprecation detail %q does not name the removal version", deprecations[0].Detail)
			}
		})
	}
}

func TestWifiBroadcastDeprecationsNoneForSupportedConfig(t *testing.T) {
	t.Parallel()

	plan := wifiStandardPlan(wifiNativeNetwork(t), wifiSecurityWithField(t, "fast_roaming_enabled", types.BoolValue(false)))
	if deprecations := wifiBroadcastDeprecations(context.Background(), plan); len(deprecations) != 0 {
		t.Fatalf("wifiBroadcastDeprecations() = %+v, want none", deprecations)
	}
}

// Enterprise types never accepted these fields, so they stay hard errors.
func TestValidateWifiBroadcastModelEnterpriseRejectsPersonalFields(t *testing.T) {
	t.Parallel()

	nasID := objectWith(t, wifiNASIDConfigurationAttrTypes(), map[string]attr.Value{
		"type": types.StringValue("DERIVED"), "source": types.StringValue("DEVICE_NAME"),
	})
	radius := objectWith(t, wifiRadiusConfigurationAttrTypes(), map[string]attr.Value{
		"profile_id": types.StringValue("00000000-0000-0000-0000-000000000101"), "nas_id": nasID,
	})
	security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
		"type":                 types.StringValue("WPA2_ENTERPRISE"),
		"radius_configuration": radius,
		"coa_enabled":          types.BoolValue(false),
		"passphrase":           types.StringValue("acceptance-passphrase"),
	})

	err := validateWifiBroadcastModel(context.Background(), wifiStandardPlan(wifiNativeNetwork(t), security))
	if err == nil || err.Error() != "security_configuration.passphrase is not valid for WPA2_ENTERPRISE" {
		t.Fatalf("validateWifiBroadcastModel() error = %v, want passphrase rejected for WPA2_ENTERPRISE", err)
	}
}

// Deprecated fields must still reach the controller, as they did in 0.4.0.
func TestExpandWifiSecurityConfigurationSendsDeprecatedFields(t *testing.T) {
	t.Parallel()

	model := wifiSecurityConfigurationModel{
		Type:                      types.StringValue("OPEN"),
		Passphrase:                types.StringNull(),
		Encryption:                types.StringNull(),
		PMFMode:                   types.StringValue("OPTIONAL"),
		FastRoamingEnabled:        types.BoolValue(false),
		GroupRekeyIntervalSeconds: types.Int64Value(3600),
		WPA3FastRoamingEnabled:    types.BoolValue(false),
		SAEConfiguration:          types.ObjectNull(wifiSAEConfigurationAttrTypes()),
		RadiusConfiguration:       types.ObjectNull(wifiRadiusConfigurationAttrTypes()),
		CoAEnabled:                types.BoolNull(),
		SecurityMode:              types.StringNull(),
		PresharedKeys:             types.ListNull(types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}),
	}

	expand := func(model wifiSecurityConfigurationModel) *client.WifiSecurityConfiguration {
		var diagnostics diag.Diagnostics
		configuration := expandWifiSecurityConfiguration(context.Background(), model, &diagnostics)
		if diagnostics.HasError() {
			t.Fatalf("expandWifiSecurityConfiguration() diagnostics = %v", diagnostics)
		}
		return configuration
	}

	configuration := expand(model)
	if configuration.PMFMode == nil || configuration.FastRoamingEnabled == nil || configuration.GroupRekeyIntervalSeconds == nil || configuration.WPA3FastRoamingEnabled == nil {
		t.Fatalf("expandWifiSecurityConfiguration() dropped a deprecated OPEN field: %+v", configuration)
	}

	for _, securityType := range []string{"WPA2_PERSONAL", "WPA3_PERSONAL"} {
		model.Type = types.StringValue(securityType)
		model.Passphrase = types.StringValue("acceptance-passphrase")
		if configuration := expand(model); configuration.WPA3FastRoamingEnabled == nil {
			t.Fatalf("expandWifiSecurityConfiguration() dropped wpa3_fast_roaming_enabled for %s", securityType)
		}
	}
}

func TestWifiBroadcastDeprecationsUnknownFieldStillWarns(t *testing.T) {
	t.Parallel()

	security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
		"type":     types.StringValue("OPEN"),
		"pmf_mode": types.StringUnknown(),
	})
	deprecations := wifiBroadcastDeprecations(context.Background(), wifiStandardPlan(wifiNativeNetwork(t), security))
	if len(deprecations) != 1 || deprecations[0].Field != "pmf_mode" {
		t.Fatalf("wifiBroadcastDeprecations() = %+v, want exactly one for pmf_mode", deprecations)
	}
}

func TestWifiBroadcastDeprecationsReportsFieldOnce(t *testing.T) {
	t.Parallel()

	security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
		"type":     types.StringValue("OPEN"),
		"pmf_mode": types.StringValue("OPTIONAL"),
	})
	plan := wifiBroadcastResourceModel{Type: types.StringValue("IOT_OPTIMIZED"), Network: wifiNativeNetwork(t), SecurityConfiguration: security}
	deprecations := wifiBroadcastDeprecations(context.Background(), plan)
	if len(deprecations) != 1 || deprecations[0].Field != "pmf_mode" {
		t.Fatalf("wifiBroadcastDeprecations() = %+v, want exactly one for pmf_mode", deprecations)
	}
	if !strings.Contains(deprecations[0].Detail, "for OPEN security") {
		t.Fatalf("deprecation detail %q, want the OPEN message", deprecations[0].Detail)
	}
}

func TestWifiBroadcastDeprecationsNoneForUnknownSecurityObject(t *testing.T) {
	t.Parallel()

	plan := wifiStandardPlan(wifiNativeNetwork(t), types.ObjectUnknown(wifiSecurityConfigurationAttrTypes()))
	if deprecations := wifiBroadcastDeprecations(context.Background(), plan); len(deprecations) != 0 {
		t.Fatalf("wifiBroadcastDeprecations() = %+v, want none", deprecations)
	}
}

func TestWifiBroadcastValidateConfigWarnsOnDeprecatedFieldPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := &wifiBroadcastResource{}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)

	security := objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
		"type":     types.StringValue("OPEN"),
		"pmf_mode": types.StringValue("OPTIONAL"),
	})
	model := wifiStandardPlan(wifiNativeNetwork(t), security)
	model.BroadcastingDeviceFilter = types.ObjectNull(wifiBroadcastingDeviceFilterAttrTypes())
	model.DNSAssistanceConfiguration = types.ObjectNull(wifiDNSAssistanceConfigurationAttrTypes())

	plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(schemaResponse.Schema.Type().TerraformType(ctx), nil)}
	if diagnostics := plan.Set(ctx, model); diagnostics.HasError() {
		t.Fatalf("plan.Set() diagnostics = %v", diagnostics)
	}
	config := tfsdk.Config{Schema: schemaResponse.Schema, Raw: plan.Raw}

	var response resource.ValidateConfigResponse
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("ValidateConfig() diagnostics = %v, want no errors", response.Diagnostics)
	}
	if response.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("ValidateConfig() warnings = %d, want 1: %v", response.Diagnostics.WarningsCount(), response.Diagnostics)
	}
	withPath, ok := response.Diagnostics[0].(diag.DiagnosticWithPath)
	if !ok {
		t.Fatalf("warning %T carries no attribute path", response.Diagnostics[0])
	}
	if want := path.Root("security_configuration").AtName("pmf_mode"); !withPath.Path().Equal(want) {
		t.Fatalf("warning path = %s, want %s", withPath.Path(), want)
	}
}

func TestValidateWifiBroadcastModelMixedSecurityPMFMode(t *testing.T) {
	t.Parallel()

	sae := objectWith(t, wifiSAEConfigurationAttrTypes(), map[string]attr.Value{
		"anticlogging_threshold_seconds": types.Int64Value(5),
		"sync_time_seconds":              types.Int64Value(5),
	})
	nasID := objectWith(t, wifiNASIDConfigurationAttrTypes(), map[string]attr.Value{
		"type": types.StringValue("DERIVED"), "source": types.StringValue("DEVICE_NAME"),
	})
	radius := objectWith(t, wifiRadiusConfigurationAttrTypes(), map[string]attr.Value{
		"profile_id": types.StringValue("00000000-0000-0000-0000-000000000101"), "nas_id": nasID,
	})
	personal := func(t *testing.T, extra map[string]attr.Value) types.Object {
		values := map[string]attr.Value{
			"type":                      types.StringValue("WPA2_WPA3_PERSONAL"),
			"passphrase":                types.StringValue("acceptance-passphrase"),
			"sae_configuration":         sae,
			"wpa3_fast_roaming_enabled": types.BoolValue(true),
		}
		for k, v := range extra {
			values[k] = v
		}
		return objectWith(t, wifiSecurityConfigurationAttrTypes(), values)
	}
	enterprise := func(t *testing.T) types.Object {
		return objectWith(t, wifiSecurityConfigurationAttrTypes(), map[string]attr.Value{
			"type":                      types.StringValue("WPA2_WPA3_ENTERPRISE"),
			"radius_configuration":      radius,
			"coa_enabled":               types.BoolValue(false),
			"wpa3_fast_roaming_enabled": types.BoolValue(true),
		})
	}
	iotPlan := func(t *testing.T, security types.Object) wifiBroadcastResourceModel {
		return wifiBroadcastResourceModel{Type: types.StringValue("IOT_OPTIMIZED"), Network: wifiNativeNetwork(t), SecurityConfiguration: security}
	}

	iotRejected := "security_configuration.type %s is not supported for IOT_OPTIMIZED broadcasts; use OPEN, WPA2_PERSONAL, or WPA2_ENTERPRISE"
	testCases := []struct {
		name     string
		plan     func(t *testing.T) wifiBroadcastResourceModel
		expected string
	}{
		{name: "IoT mixed personal is rejected", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return iotPlan(t, personal(t, nil))
		}, expected: fmt.Sprintf(iotRejected, "WPA2_WPA3_PERSONAL")},
		{name: "STANDARD personal without pmf_mode", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), personal(t, nil))
		}, expected: "security_configuration.passphrase, pmf_mode, sae_configuration, and wpa3_fast_roaming_enabled are required for WPA2_WPA3_PERSONAL"},
		{name: "IoT mixed enterprise is rejected", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return iotPlan(t, enterprise(t))
		}, expected: fmt.Sprintf(iotRejected, "WPA2_WPA3_ENTERPRISE")},
		{name: "STANDARD enterprise without pmf_mode", plan: func(t *testing.T) wifiBroadcastResourceModel {
			return wifiStandardPlan(wifiNativeNetwork(t), enterprise(t))
		}, expected: "security_configuration.radius_configuration, coa_enabled, pmf_mode, and wpa3_fast_roaming_enabled are required for WPA2_WPA3_ENTERPRISE"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			plan := testCase.plan(t)
			err := validateWifiBroadcastModel(context.Background(), plan)
			if testCase.expected == "" {
				if err != nil {
					t.Fatalf("validateWifiBroadcastModel() error = %v, want nil", err)
				}
				if deprecations := wifiBroadcastDeprecations(context.Background(), plan); len(deprecations) != 0 {
					t.Fatalf("wifiBroadcastDeprecations() = %+v, want none", deprecations)
				}
				return
			}
			if err == nil || err.Error() != testCase.expected {
				t.Fatalf("validateWifiBroadcastModel() error = %v, want %q", err, testCase.expected)
			}
		})
	}
}
