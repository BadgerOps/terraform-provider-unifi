package provider

import (
	"context"
	"strings"
	"reflect"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

func TestFirewallPolicyPortFilterRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	portSet, diags := types.SetValueFrom(ctx, types.StringType, []string{"443", "8443-8444"})
	if diags.HasError() {
		t.Fatalf("types.SetValueFrom() diagnostics = %v", diags)
	}

	value, diags := types.ObjectValueFrom(ctx, firewallPolicyPortFilterAttrTypes(), firewallPolicyPortFilterModel{
		Type:          types.StringValue("PORTS"),
		MatchOpposite: types.BoolValue(true),
		Ports:         portSet,
	})
	if diags.HasError() {
		t.Fatalf("types.ObjectValueFrom() diagnostics = %v", diags)
	}

	var expandDiags diag.Diagnostics
	filter := expandFirewallPolicyPortFilter(ctx, value, "destination_filter.port_filter", &expandDiags)
	if expandDiags.HasError() {
		t.Fatalf("expandFirewallPolicyPortFilter() diagnostics = %v", expandDiags)
	}
	if filter == nil {
		t.Fatal("expandFirewallPolicyPortFilter() returned nil")
	}

	if filter.Type != "PORTS" {
		t.Fatalf("Type = %q, want PORTS", filter.Type)
	}
	if !filter.MatchOpposite {
		t.Fatal("MatchOpposite = false, want true")
	}

	got, diags := flattenFirewallPolicyPortFilter(ctx, filter)
	if diags.HasError() {
		t.Fatalf("flattenFirewallPolicyPortFilter() diagnostics = %v", diags)
	}

	model, err := decodeFirewallPolicyPortFilter(ctx, got, "destination_filter.port_filter")
	if err != nil {
		t.Fatalf("decodeFirewallPolicyPortFilter() error = %v", err)
	}

	var ports []string
	modelDiags := model.Ports.ElementsAs(ctx, &ports, false)
	if modelDiags.HasError() {
		t.Fatalf("model.Ports.ElementsAs() diagnostics = %v", modelDiags)
	}

	if model.Type.ValueString() != "PORTS" {
		t.Fatalf("decoded type = %q, want PORTS", model.Type.ValueString())
	}
	if !model.MatchOpposite.ValueBool() {
		t.Fatal("decoded match_opposite = false, want true")
	}
	if !reflect.DeepEqual(ports, []string{"443", "8443-8444"}) {
		t.Fatalf("decoded ports = %#v, want %#v", ports, []string{"443", "8443-8444"})
	}
}

func TestBuildFirewallPolicyStateModelComplex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	policy := &client.FirewallPolicy{
		ID:          "00000000-0000-0000-0000-000000000101",
		Enabled:     true,
		Name:        "trusted-to-apps",
		Description: stringPtrProvider("policy description"),
		Action: &client.FirewallPolicyAction{
			Type:               "ALLOW",
			AllowReturnTraffic: boolPtrProvider(true),
		},
		Source: &client.FirewallPolicyEndpoint{
			ZoneID: "00000000-0000-0000-0000-000000000201",
			TrafficFilter: &client.FirewallPolicyTrafficFilter{
				Type: "NETWORK",
				NetworkFilter: &client.FirewallPolicyNetworkFilter{
					NetworkIDs:    []string{"00000000-0000-0000-0000-000000000301"},
					MatchOpposite: true,
				},
				MACAddress: stringPtrProvider("AA:BB:CC:DD:EE:FF"),
				PortFilter: &client.FirewallPolicyPortFilter{
					Type:          "PORTS",
					MatchOpposite: false,
					Items: []client.PortMatch{
						{Type: "PORT_NUMBER", Value: int64PtrProvider(443)},
					},
				},
			},
		},
		Destination: &client.FirewallPolicyEndpoint{
			ZoneID: "00000000-0000-0000-0000-000000000202",
			TrafficFilter: &client.FirewallPolicyTrafficFilter{
				Type: "APPLICATION_CATEGORY",
				ApplicationCategoryFilter: &client.FirewallPolicyApplicationCategoryFilter{
					ApplicationCategoryIDs: []int64{5, 7},
				},
				PortFilter: &client.FirewallPolicyPortFilter{
					Type:                  "TRAFFIC_MATCHING_LIST",
					MatchOpposite:         true,
					TrafficMatchingListID: stringPtrProvider("00000000-0000-0000-0000-000000000401"),
				},
			},
		},
		IPProtocolScope: &client.FirewallPolicyIPProtocolScope{
			IPVersion: "IPV4_AND_IPV6",
			ProtocolFilter: &client.FirewallPolicyProtocolFilter{
				Type:           "PROTOCOL_NUMBER",
				ProtocolNumber: int64PtrProvider(17),
				MatchOpposite:  boolPtrProvider(true),
			},
		},
		ConnectionStateFilter: []string{"NEW", "ESTABLISHED"},
		IPsecFilter:           stringPtrProvider("MATCH_ENCRYPTED"),
		LoggingEnabled:        true,
		Schedule: &client.FirewallSchedule{
			Mode: "CUSTOM",
			TimeFilter: &client.FirewallScheduleTime{
				StartTime: "08:00",
				StopTime:  "18:00",
			},
			RepeatOnDays: []string{"MONDAY", "WEDNESDAY"},
			StartDate:    stringPtrProvider("2026-01-01"),
			StopDate:     stringPtrProvider("2026-12-31"),
		},
		Index: 9,
	}

	model, diags := buildFirewallPolicyStateModel(ctx, types.StringValue("00000000-0000-0000-0000-000000000001"), policy)
	if diags.HasError() {
		t.Fatalf("buildFirewallPolicyStateModel() diagnostics = %v", diags)
	}

	if model.Action.ValueString() != "ALLOW" {
		t.Fatalf("Action = %q, want ALLOW", model.Action.ValueString())
	}
	if !model.AllowReturnTraffic.ValueBool() {
		t.Fatal("AllowReturnTraffic = false, want true")
	}
	if model.IPVersion.ValueString() != "IPV4_AND_IPV6" {
		t.Fatalf("IPVersion = %q, want IPV4_AND_IPV6", model.IPVersion.ValueString())
	}
	if model.Index.ValueInt64() != 9 {
		t.Fatalf("Index = %d, want 9", model.Index.ValueInt64())
	}

	sourceFilter, err := decodeFirewallPolicyEndpointFilter(ctx, model.SourceFilter, "source_filter")
	if err != nil {
		t.Fatalf("decodeFirewallPolicyEndpointFilter(source) error = %v", err)
	}
	if sourceFilter.Type.ValueString() != "NETWORK" {
		t.Fatalf("source filter type = %q, want NETWORK", sourceFilter.Type.ValueString())
	}
	if sourceFilter.MACAddress.ValueString() != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("source MACAddress = %q, want AA:BB:CC:DD:EE:FF", sourceFilter.MACAddress.ValueString())
	}

	destinationFilter, err := decodeFirewallPolicyEndpointFilter(ctx, model.DestinationFilter, "destination_filter")
	if err != nil {
		t.Fatalf("decodeFirewallPolicyEndpointFilter(destination) error = %v", err)
	}
	if destinationFilter.Type.ValueString() != "APPLICATION_CATEGORY" {
		t.Fatalf("destination filter type = %q, want APPLICATION_CATEGORY", destinationFilter.Type.ValueString())
	}
	var categoryIDs []int64
	destinationDiags := destinationFilter.ApplicationCategoryIDs.ElementsAs(ctx, &categoryIDs, false)
	if destinationDiags.HasError() {
		t.Fatalf("destination category IDs diagnostics = %v", destinationDiags)
	}
	slices.Sort(categoryIDs)
	if !reflect.DeepEqual(categoryIDs, []int64{5, 7}) {
		t.Fatalf("destination category IDs = %#v, want %#v", categoryIDs, []int64{5, 7})
	}

	protocolFilter, err := decodeFirewallPolicyProtocolFilter(ctx, model.ProtocolFilter, "protocol_filter")
	if err != nil {
		t.Fatalf("decodeFirewallPolicyProtocolFilter() error = %v", err)
	}
	if protocolFilter.Type.ValueString() != "PROTOCOL_NUMBER" {
		t.Fatalf("protocol filter type = %q, want PROTOCOL_NUMBER", protocolFilter.Type.ValueString())
	}
	if protocolFilter.ProtocolNumber.ValueInt64() != 17 {
		t.Fatalf("protocol number = %d, want 17", protocolFilter.ProtocolNumber.ValueInt64())
	}
	if !protocolFilter.MatchOpposite.ValueBool() {
		t.Fatal("protocol match_opposite = false, want true")
	}

	schedule, err := decodeFirewallPolicySchedule(ctx, model.Schedule, "schedule")
	if err != nil {
		t.Fatalf("decodeFirewallPolicySchedule() error = %v", err)
	}
	if schedule.Mode.ValueString() != "CUSTOM" {
		t.Fatalf("schedule mode = %q, want CUSTOM", schedule.Mode.ValueString())
	}
	if schedule.StartTime.ValueString() != "08:00" || schedule.StopTime.ValueString() != "18:00" {
		t.Fatalf("schedule time = %s-%s, want 08:00-18:00", schedule.StartTime.ValueString(), schedule.StopTime.ValueString())
	}
}

func TestValidateFirewallPolicyBaseRequiresAllowReturnTrafficForAllow(t *testing.T) {
	t.Parallel()

	err := validateFirewallPolicyBase(firewallPolicyModel{
		Action:             types.StringValue("ALLOW"),
		AllowReturnTraffic: types.BoolNull(),
		IPVersion:          types.StringValue("IPV4"),
	})
	if err == nil {
		t.Fatal("validateFirewallPolicyBase() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "allow_return_traffic must be set explicitly") {
		t.Fatalf("validateFirewallPolicyBase() error = %q, want explicit allow_return_traffic error", err.Error())
	}
}

func TestExpandFirewallPolicyProtocolFilterNamedProtocolValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	validValue, diags := types.ObjectValueFrom(ctx, firewallPolicyProtocolFilterAttrTypes(), firewallPolicyProtocolFilterModel{
		Type:           types.StringValue("NAMED_PROTOCOL"),
		NamedProtocol:  types.StringValue("icmp"),
		MatchOpposite:  types.BoolValue(false),
		ProtocolNumber: types.Int64Null(),
		PresetName:     types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("types.ObjectValueFrom(valid) diagnostics = %v", diags)
	}

	var expandDiags diag.Diagnostics
	filter := expandFirewallPolicyProtocolFilter(ctx, validValue, &expandDiags)
	if expandDiags.HasError() {
		t.Fatalf("expandFirewallPolicyProtocolFilter(valid) diagnostics = %v", expandDiags)
	}
	if filter == nil || filter.Protocol == nil || filter.Protocol.Name != "ICMP" {
		t.Fatalf("expandFirewallPolicyProtocolFilter(valid) = %#v, want ICMP protocol", filter)
	}

	invalidValue, diags := types.ObjectValueFrom(ctx, firewallPolicyProtocolFilterAttrTypes(), firewallPolicyProtocolFilterModel{
		Type:           types.StringValue("NAMED_PROTOCOL"),
		NamedProtocol:  types.StringValue("tcp"),
		MatchOpposite:  types.BoolValue(false),
		ProtocolNumber: types.Int64Null(),
		PresetName:     types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("types.ObjectValueFrom(invalid) diagnostics = %v", diags)
	}

	expandDiags = nil
	filter = expandFirewallPolicyProtocolFilter(ctx, invalidValue, &expandDiags)
	if filter != nil {
		t.Fatalf("expandFirewallPolicyProtocolFilter(invalid) = %#v, want nil", filter)
	}
	if !expandDiags.HasError() {
		t.Fatal("expandFirewallPolicyProtocolFilter(invalid) diagnostics = nil, want error")
	}
	if !strings.Contains(expandDiags[0].Detail(), "currently supports only ICMP") {
		t.Fatalf("expandFirewallPolicyProtocolFilter(invalid) detail = %q, want ICMP guidance", expandDiags[0].Detail())
	}
}

func stringPtrProvider(value string) *string {
	return &value
}

func boolPtrProvider(value bool) *bool {
	return &value
}

func int64PtrProvider(value int64) *int64 {
	return &value
}
