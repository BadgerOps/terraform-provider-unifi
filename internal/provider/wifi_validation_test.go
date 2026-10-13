package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
