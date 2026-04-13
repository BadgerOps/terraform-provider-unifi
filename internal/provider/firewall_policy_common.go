package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

type firewallPolicyModel struct {
	ID                    types.String `tfsdk:"id"`
	SiteID                types.String `tfsdk:"site_id"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Action                types.String `tfsdk:"action"`
	AllowReturnTraffic    types.Bool   `tfsdk:"allow_return_traffic"`
	SourceZoneID          types.String `tfsdk:"source_zone_id"`
	SourceFilter          types.Object `tfsdk:"source_filter"`
	DestinationZoneID     types.String `tfsdk:"destination_zone_id"`
	DestinationFilter     types.Object `tfsdk:"destination_filter"`
	IPVersion             types.String `tfsdk:"ip_version"`
	ProtocolFilter        types.Object `tfsdk:"protocol_filter"`
	ConnectionStateFilter types.Set    `tfsdk:"connection_state_filter"`
	IPsecFilter           types.String `tfsdk:"ipsec_filter"`
	LoggingEnabled        types.Bool   `tfsdk:"logging_enabled"`
	Schedule              types.Object `tfsdk:"schedule"`
	Index                 types.Int64  `tfsdk:"index"`
}

type firewallPolicyEndpointFilterModel struct {
	Type                           types.String `tfsdk:"type"`
	NetworkIDs                     types.Set    `tfsdk:"network_ids"`
	NetworkMatchOpposite           types.Bool   `tfsdk:"network_match_opposite"`
	MACAddress                     types.String `tfsdk:"mac_address"`
	MACAddresses                   types.Set    `tfsdk:"mac_addresses"`
	IPAddresses                    types.Set    `tfsdk:"ip_addresses"`
	IPAddressMatchOpposite         types.Bool   `tfsdk:"ip_address_match_opposite"`
	IPAddressTrafficMatchingListID types.String `tfsdk:"ip_address_traffic_matching_list_id"`
	IPv6IID                        types.String `tfsdk:"ipv6_iid"`
	IPv6IIDMatchOpposite           types.Bool   `tfsdk:"ipv6_iid_match_opposite"`
	RegionCodes                    types.Set    `tfsdk:"region_codes"`
	VPNServerIDs                   types.Set    `tfsdk:"vpn_server_ids"`
	VPNServerMatchOpposite         types.Bool   `tfsdk:"vpn_server_match_opposite"`
	SiteToSiteVPNTunnelID          types.String `tfsdk:"site_to_site_vpn_tunnel_id"`
	Domains                        types.Set    `tfsdk:"domains"`
	ApplicationIDs                 types.Set    `tfsdk:"application_ids"`
	ApplicationCategoryIDs         types.Set    `tfsdk:"application_category_ids"`
	PortFilter                     types.Object `tfsdk:"port_filter"`
}

type firewallPolicyPortFilterModel struct {
	Type                  types.String `tfsdk:"type"`
	MatchOpposite         types.Bool   `tfsdk:"match_opposite"`
	Ports                 types.Set    `tfsdk:"ports"`
	TrafficMatchingListID types.String `tfsdk:"traffic_matching_list_id"`
}

type firewallPolicyProtocolFilterModel struct {
	Type           types.String `tfsdk:"type"`
	NamedProtocol  types.String `tfsdk:"named_protocol"`
	MatchOpposite  types.Bool   `tfsdk:"match_opposite"`
	ProtocolNumber types.Int64  `tfsdk:"protocol_number"`
	PresetName     types.String `tfsdk:"preset_name"`
}

type firewallPolicyScheduleModel struct {
	Mode         types.String `tfsdk:"mode"`
	StartTime    types.String `tfsdk:"start_time"`
	StopTime     types.String `tfsdk:"stop_time"`
	RepeatOnDays types.Set    `tfsdk:"repeat_on_days"`
	Date         types.String `tfsdk:"date"`
	StartDate    types.String `tfsdk:"start_date"`
	StopDate     types.String `tfsdk:"stop_date"`
}

func firewallPolicyPortFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":                     types.StringType,
		"match_opposite":           types.BoolType,
		"ports":                    types.SetType{ElemType: types.StringType},
		"traffic_matching_list_id": types.StringType,
	}
}

func firewallPolicyEndpointFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":                                types.StringType,
		"network_ids":                         types.SetType{ElemType: types.StringType},
		"network_match_opposite":              types.BoolType,
		"mac_address":                         types.StringType,
		"mac_addresses":                       types.SetType{ElemType: types.StringType},
		"ip_addresses":                        types.SetType{ElemType: types.StringType},
		"ip_address_match_opposite":           types.BoolType,
		"ip_address_traffic_matching_list_id": types.StringType,
		"ipv6_iid":                            types.StringType,
		"ipv6_iid_match_opposite":             types.BoolType,
		"region_codes":                        types.SetType{ElemType: types.StringType},
		"vpn_server_ids":                      types.SetType{ElemType: types.StringType},
		"vpn_server_match_opposite":           types.BoolType,
		"site_to_site_vpn_tunnel_id":          types.StringType,
		"domains":                             types.SetType{ElemType: types.StringType},
		"application_ids":                     types.SetType{ElemType: types.Int64Type},
		"application_category_ids":            types.SetType{ElemType: types.Int64Type},
		"port_filter":                         types.ObjectType{AttrTypes: firewallPolicyPortFilterAttrTypes()},
	}
}

func firewallPolicyProtocolFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":            types.StringType,
		"named_protocol":  types.StringType,
		"match_opposite":  types.BoolType,
		"protocol_number": types.Int64Type,
		"preset_name":     types.StringType,
	}
}

func firewallPolicyScheduleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":           types.StringType,
		"start_time":     types.StringType,
		"stop_time":      types.StringType,
		"repeat_on_days": types.SetType{ElemType: types.StringType},
		"date":           types.StringType,
		"start_date":     types.StringType,
		"stop_date":      types.StringType,
	}
}

func decodeFirewallPolicyEndpointFilter(ctx context.Context, value types.Object, path string) (firewallPolicyEndpointFilterModel, error) {
	var model firewallPolicyEndpointFilterModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return model, fmt.Errorf("unable to decode %s", path)
	}
	return model, nil
}

func decodeFirewallPolicyPortFilter(ctx context.Context, value types.Object, path string) (firewallPolicyPortFilterModel, error) {
	var model firewallPolicyPortFilterModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return model, fmt.Errorf("unable to decode %s", path)
	}
	return model, nil
}

func decodeFirewallPolicyProtocolFilter(ctx context.Context, value types.Object, path string) (firewallPolicyProtocolFilterModel, error) {
	var model firewallPolicyProtocolFilterModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return model, fmt.Errorf("unable to decode %s", path)
	}
	return model, nil
}

func decodeFirewallPolicySchedule(ctx context.Context, value types.Object, path string) (firewallPolicyScheduleModel, error) {
	var model firewallPolicyScheduleModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return model, fmt.Errorf("unable to decode %s", path)
	}
	return model, nil
}

func expandFirewallPolicyPortFilter(ctx context.Context, value types.Object, path string, diags *diag.Diagnostics) *client.FirewallPolicyPortFilter {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	model, err := decodeFirewallPolicyPortFilter(ctx, value, path)
	if err != nil {
		diags.AddError("Invalid firewall policy port filter", err.Error())
		return nil
	}

	filter := &client.FirewallPolicyPortFilter{
		Type:          model.Type.ValueString(),
		MatchOpposite: boolValueOrFalse(model.MatchOpposite),
	}

	switch model.Type.ValueString() {
	case "PORTS":
		ports := setToStrings(ctx, model.Ports, path+".ports", diags)
		if diags.HasError() {
			return nil
		}
		items, err := expandPortMatches(ports)
		if err != nil {
			diags.AddError("Invalid firewall policy port filter", fmt.Sprintf("%s.ports: %s", path, err.Error()))
			return nil
		}
		filter.Items = items
	case "TRAFFIC_MATCHING_LIST":
		filter.TrafficMatchingListID = stringPointerValue(model.TrafficMatchingListID)
	default:
		diags.AddError("Invalid firewall policy port filter", fmt.Sprintf("%s.type must be one of PORTS or TRAFFIC_MATCHING_LIST", path))
		return nil
	}

	return filter
}

func flattenFirewallPolicyPortFilter(ctx context.Context, filter *client.FirewallPolicyPortFilter) (types.Object, diag.Diagnostics) {
	if filter == nil {
		return types.ObjectNull(firewallPolicyPortFilterAttrTypes()), nil
	}

	ports := types.SetNull(types.StringType)
	var diagnostics diag.Diagnostics
	if len(filter.Items) > 0 {
		values := flattenPortMatches(filter.Items)
		ports, diagnostics = stringSetValue(ctx, values)
		if diagnostics.HasError() {
			return types.ObjectNull(firewallPolicyPortFilterAttrTypes()), diagnostics
		}
	}

	model := firewallPolicyPortFilterModel{
		Type:                  types.StringValue(filter.Type),
		MatchOpposite:         types.BoolValue(filter.MatchOpposite),
		Ports:                 ports,
		TrafficMatchingListID: nullableString(filter.TrafficMatchingListID),
	}

	value, objectDiags := types.ObjectValueFrom(ctx, firewallPolicyPortFilterAttrTypes(), model)
	diagnostics.Append(objectDiags...)
	return value, diagnostics
}

func buildFirewallPolicyStateModel(ctx context.Context, siteID types.String, firewallPolicy *client.FirewallPolicy) (firewallPolicyModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	connectionStates, setDiags := stringSetValue(ctx, firewallPolicy.ConnectionStateFilter)
	diagnostics.Append(setDiags...)
	sourceFilter, objectDiags := flattenFirewallPolicyEndpointFilter(ctx, "source", firewallPolicy.Source)
	diagnostics.Append(objectDiags...)
	destinationFilter, objectDiags := flattenFirewallPolicyEndpointFilter(ctx, "destination", firewallPolicy.Destination)
	diagnostics.Append(objectDiags...)
	protocolFilter, objectDiags := flattenFirewallPolicyProtocolFilter(ctx, firewallPolicy.IPProtocolScope)
	diagnostics.Append(objectDiags...)
	schedule, objectDiags := flattenFirewallPolicySchedule(ctx, firewallPolicy.Schedule)
	diagnostics.Append(objectDiags...)
	if diagnostics.HasError() {
		return firewallPolicyModel{}, diagnostics
	}

	allowReturnTraffic := types.BoolNull()
	if firewallPolicy.Action != nil {
		allowReturnTraffic = nullableBool(firewallPolicy.Action.AllowReturnTraffic)
	}

	model := firewallPolicyModel{
		ID:                    types.StringValue(firewallPolicy.ID),
		SiteID:                siteID,
		Enabled:               types.BoolValue(firewallPolicy.Enabled),
		Name:                  types.StringValue(firewallPolicy.Name),
		Description:           nullableString(firewallPolicy.Description),
		Action:                types.StringValue(firewallPolicy.Action.Type),
		AllowReturnTraffic:    allowReturnTraffic,
		SourceZoneID:          types.StringValue(firewallPolicy.Source.ZoneID),
		SourceFilter:          sourceFilter,
		DestinationZoneID:     types.StringValue(firewallPolicy.Destination.ZoneID),
		DestinationFilter:     destinationFilter,
		IPVersion:             types.StringValue(firewallPolicy.IPProtocolScope.IPVersion),
		ProtocolFilter:        protocolFilter,
		ConnectionStateFilter: connectionStates,
		IPsecFilter:           nullableString(firewallPolicy.IPsecFilter),
		LoggingEnabled:        types.BoolValue(firewallPolicy.LoggingEnabled),
		Schedule:              schedule,
		Index:                 types.Int64Value(firewallPolicy.Index),
	}

	return model, diagnostics
}

func flattenFirewallPolicyEndpointFilter(ctx context.Context, side string, endpoint *client.FirewallPolicyEndpoint) (types.Object, diag.Diagnostics) {
	if endpoint == nil || endpoint.TrafficFilter == nil {
		return types.ObjectNull(firewallPolicyEndpointFilterAttrTypes()), nil
	}

	filter := endpoint.TrafficFilter
	model := firewallPolicyEndpointFilterModel{
		Type:                   types.StringValue(filter.Type),
		NetworkIDs:             types.SetNull(types.StringType),
		MACAddresses:           types.SetNull(types.StringType),
		IPAddresses:            types.SetNull(types.StringType),
		RegionCodes:            types.SetNull(types.StringType),
		VPNServerIDs:           types.SetNull(types.StringType),
		Domains:                types.SetNull(types.StringType),
		ApplicationIDs:         types.SetNull(types.Int64Type),
		ApplicationCategoryIDs: types.SetNull(types.Int64Type),
		PortFilter:             types.ObjectNull(firewallPolicyPortFilterAttrTypes()),
	}

	var diagnostics diag.Diagnostics
	var setDiags diag.Diagnostics

	switch filter.Type {
	case "PORT":
		model.PortFilter, diagnostics = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
	case "NETWORK":
		model.NetworkIDs, setDiags = stringSetValue(ctx, filter.NetworkFilter.NetworkIDs)
		diagnostics.Append(setDiags...)
		model.NetworkMatchOpposite = types.BoolValue(filter.NetworkFilter.MatchOpposite)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
		if side == "source" {
			model.MACAddress = nullableString(filter.MACAddress)
		}
	case "MAC_ADDRESS":
		model.MACAddresses, setDiags = stringSetValue(ctx, filter.MACAddressFilter.MacAddresses)
		diagnostics.Append(setDiags...)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	case "IP_ADDRESS":
		if filter.IPAddressFilter != nil {
			values, err := flattenGenericIPMatches(filter.IPAddressFilter.Items)
			if err != nil {
				diagnostics.AddError("Unable to flatten firewall policy IP address filter", err.Error())
				break
			}
			model.IPAddresses, setDiags = stringSetValue(ctx, values)
			diagnostics.Append(setDiags...)
			model.IPAddressMatchOpposite = types.BoolValue(filter.IPAddressFilter.MatchOpposite)
			model.IPAddressTrafficMatchingListID = nullableString(filter.IPAddressFilter.TrafficMatchingListID)
		}
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
		if side == "source" {
			model.MACAddress = nullableString(filter.MACAddress)
		}
	case "IPV6_IID":
		if filter.IPv6IIDFilter != nil {
			model.IPv6IID = types.StringValue(filter.IPv6IIDFilter.IPv6IID)
			model.IPv6IIDMatchOpposite = types.BoolValue(filter.IPv6IIDFilter.MatchOpposite)
		}
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
		if side == "source" {
			model.MACAddress = nullableString(filter.MACAddress)
		}
	case "REGION":
		model.RegionCodes, setDiags = stringSetValue(ctx, filter.RegionFilter.Regions)
		diagnostics.Append(setDiags...)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	case "VPN_SERVER":
		model.VPNServerIDs, setDiags = stringSetValue(ctx, filter.VPNServerFilter.VPNServerIDs)
		diagnostics.Append(setDiags...)
		model.VPNServerMatchOpposite = types.BoolValue(filter.VPNServerFilter.MatchOpposite)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	case "SITE_TO_SITE_VPN_TUNNEL":
		if filter.SiteToSiteVPNTunnelFilter != nil {
			model.SiteToSiteVPNTunnelID = types.StringValue(filter.SiteToSiteVPNTunnelFilter.SiteToSiteVPNTunnelID)
		}
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	case "DOMAIN":
		model.Domains, setDiags = stringSetValue(ctx, filter.DomainFilter.Domains)
		diagnostics.Append(setDiags...)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	case "APPLICATION":
		model.ApplicationIDs, setDiags = int64SetValue(ctx, filter.ApplicationFilter.ApplicationIDs)
		diagnostics.Append(setDiags...)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	case "APPLICATION_CATEGORY":
		model.ApplicationCategoryIDs, setDiags = int64SetValue(ctx, filter.ApplicationCategoryFilter.ApplicationCategoryIDs)
		diagnostics.Append(setDiags...)
		model.PortFilter, setDiags = flattenFirewallPolicyPortFilter(ctx, filter.PortFilter)
		diagnostics.Append(setDiags...)
	}

	value, objectDiags := types.ObjectValueFrom(ctx, firewallPolicyEndpointFilterAttrTypes(), model)
	diagnostics.Append(objectDiags...)
	return value, diagnostics
}

func flattenFirewallPolicyProtocolFilter(ctx context.Context, scope *client.FirewallPolicyIPProtocolScope) (types.Object, diag.Diagnostics) {
	if scope == nil || scope.ProtocolFilter == nil {
		return types.ObjectNull(firewallPolicyProtocolFilterAttrTypes()), nil
	}

	model := firewallPolicyProtocolFilterModel{
		Type:           types.StringValue(scope.ProtocolFilter.Type),
		NamedProtocol:  types.StringNull(),
		MatchOpposite:  nullableBool(scope.ProtocolFilter.MatchOpposite),
		ProtocolNumber: nullableInt64(scope.ProtocolFilter.ProtocolNumber),
		PresetName:     types.StringNull(),
	}
	if scope.ProtocolFilter.Protocol != nil {
		model.NamedProtocol = types.StringValue(scope.ProtocolFilter.Protocol.Name)
	}
	if scope.ProtocolFilter.Preset != nil {
		model.PresetName = types.StringValue(scope.ProtocolFilter.Preset.Name)
	}

	value, diagnostics := types.ObjectValueFrom(ctx, firewallPolicyProtocolFilterAttrTypes(), model)
	return value, diagnostics
}

func flattenFirewallPolicySchedule(ctx context.Context, schedule *client.FirewallSchedule) (types.Object, diag.Diagnostics) {
	if schedule == nil {
		return types.ObjectNull(firewallPolicyScheduleAttrTypes()), nil
	}

	repeatOnDays := types.SetNull(types.StringType)
	var diagnostics diag.Diagnostics
	if len(schedule.RepeatOnDays) > 0 {
		repeatOnDays, diagnostics = stringSetValue(ctx, schedule.RepeatOnDays)
		if diagnostics.HasError() {
			return types.ObjectNull(firewallPolicyScheduleAttrTypes()), diagnostics
		}
	}

	model := firewallPolicyScheduleModel{
		Mode:         types.StringValue(schedule.Mode),
		StartTime:    types.StringNull(),
		StopTime:     types.StringNull(),
		RepeatOnDays: repeatOnDays,
		Date:         nullableString(schedule.Date),
		StartDate:    nullableString(schedule.StartDate),
		StopDate:     nullableString(schedule.StopDate),
	}
	if schedule.TimeFilter != nil {
		model.StartTime = types.StringValue(schedule.TimeFilter.StartTime)
		model.StopTime = types.StringValue(schedule.TimeFilter.StopTime)
	}

	value, objectDiags := types.ObjectValueFrom(ctx, firewallPolicyScheduleAttrTypes(), model)
	diagnostics.Append(objectDiags...)
	return value, diagnostics
}
