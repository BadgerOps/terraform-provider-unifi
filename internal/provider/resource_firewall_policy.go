package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*firewallPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*firewallPolicyResource)(nil)
)

type firewallPolicyResource struct {
	providerData *providerData
}

type firewallPolicyResourceModel struct {
	ID                              types.String `tfsdk:"id"`
	SiteID                          types.String `tfsdk:"site_id"`
	Enabled                         types.Bool   `tfsdk:"enabled"`
	Name                            types.String `tfsdk:"name"`
	Description                     types.String `tfsdk:"description"`
	Action                          types.String `tfsdk:"action"`
	SourceZoneID                    types.String `tfsdk:"source_zone_id"`
	SourceNetworkIDs                types.Set    `tfsdk:"source_network_ids"`
	SourceNetworkMatchOpposite      types.Bool   `tfsdk:"source_network_match_opposite"`
	SourcePortFilter                types.Object `tfsdk:"source_port_filter"`
	DestinationZoneID               types.String `tfsdk:"destination_zone_id"`
	DestinationNetworkIDs           types.Set    `tfsdk:"destination_network_ids"`
	DestinationNetworkMatchOpposite types.Bool   `tfsdk:"destination_network_match_opposite"`
	DestinationPortFilter           types.Object `tfsdk:"destination_port_filter"`
	IPVersion                       types.String `tfsdk:"ip_version"`
	ConnectionStateFilter           types.Set    `tfsdk:"connection_state_filter"`
	IPsecFilter                     types.String `tfsdk:"ipsec_filter"`
	LoggingEnabled                  types.Bool   `tfsdk:"logging_enabled"`
	Index                           types.Int64  `tfsdk:"index"`
}

type firewallPolicyPortFilterModel struct {
	Type                  types.String `tfsdk:"type"`
	MatchOpposite         types.Bool   `tfsdk:"match_opposite"`
	Ports                 types.Set    `tfsdk:"ports"`
	TrafficMatchingListID types.String `tfsdk:"traffic_matching_list_id"`
}

func firewallPolicyPortFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":                     types.StringType,
		"match_opposite":           types.BoolType,
		"ports":                    types.SetType{ElemType: types.StringType},
		"traffic_matching_list_id": types.StringType,
	}
}

func NewFirewallPolicyResource() resource.Resource {
	return &firewallPolicyResource{}
}

func (r *firewallPolicyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_policy"
}

func (r *firewallPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi firewall policy with zone-based matching and optional source and destination network or port filters.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"action": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Policy action. Supported values: `ALLOW`, `BLOCK`, `REJECT`.",
			},
			"source_zone_id": schema.StringAttribute{
				Required: true,
			},
			"source_network_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"source_network_match_opposite": schema.BoolAttribute{
				Optional: true,
			},
			"source_port_filter": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Source port filter. Set either inline `ports` with `type = \"PORTS\"` or `traffic_matching_list_id` with `type = \"TRAFFIC_MATCHING_LIST\"`.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"match_opposite": schema.BoolAttribute{
						Optional: true,
					},
					"ports": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"traffic_matching_list_id": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"destination_zone_id": schema.StringAttribute{
				Required: true,
			},
			"destination_network_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"destination_network_match_opposite": schema.BoolAttribute{
				Optional: true,
			},
			"destination_port_filter": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Destination port filter. Set either inline `ports` with `type = \"PORTS\"` or `traffic_matching_list_id` with `type = \"TRAFFIC_MATCHING_LIST\"`.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"match_opposite": schema.BoolAttribute{
						Optional: true,
					},
					"ports": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"traffic_matching_list_id": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"ip_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "IP version scope. Supported values: `IPV4`, `IPV6`, `IPV4_AND_IPV6`.",
			},
			"connection_state_filter": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"ipsec_filter": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional IPsec traffic filter. Supported values: `MATCH_ENCRYPTED`, `MATCH_NOT_ENCRYPTED`.",
			},
			"logging_enabled": schema.BoolAttribute{
				Required: true,
			},
			"index": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Controller-assigned firewall policy ordering index.",
			},
		},
	}
}

func (r *firewallPolicyResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *firewallPolicyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan firewallPolicyResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := r.expandPolicy(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateFirewallPolicy(ctx, plan.SiteID.ValueString(), apiPolicy)
	if err != nil {
		response.Diagnostics.AddError("Unable to create firewall policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *firewallPolicyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state firewallPolicyResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallPolicy, err := r.providerData.client.GetFirewallPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read firewall policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, firewallPolicy)
}

func (r *firewallPolicyResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan firewallPolicyResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state firewallPolicyResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := r.expandPolicy(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateFirewallPolicy(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiPolicy)
	if err != nil {
		response.Diagnostics.AddError("Unable to update firewall policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *firewallPolicyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state firewallPolicyResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteFirewallPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete firewall policy", err.Error())
	}
}

func (r *firewallPolicyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func (r *firewallPolicyResource) expandPolicy(ctx context.Context, plan firewallPolicyResourceModel, diags *diag.Diagnostics) client.FirewallPolicy {
	policy := client.FirewallPolicy{
		Enabled: plan.Enabled.ValueBool(),
		Name:    plan.Name.ValueString(),
		Action: &client.FirewallPolicyAction{
			Type: plan.Action.ValueString(),
		},
		Source: &client.FirewallPolicyEndpoint{
			ZoneID: plan.SourceZoneID.ValueString(),
		},
		Destination: &client.FirewallPolicyEndpoint{
			ZoneID: plan.DestinationZoneID.ValueString(),
		},
		IPProtocolScope: &client.FirewallPolicyIPProtocolScope{
			IPVersion: plan.IPVersion.ValueString(),
		},
		LoggingEnabled: plan.LoggingEnabled.ValueBool(),
		Description:    stringPointerValue(plan.Description),
		IPsecFilter:    stringPointerValue(plan.IPsecFilter),
	}

	if err := validateFirewallPolicyModel(ctx, plan); err != nil {
		diags.AddError("Invalid firewall policy configuration", err.Error())
		return policy
	}

	if !plan.ConnectionStateFilter.IsNull() {
		policy.ConnectionStateFilter = setToStrings(ctx, plan.ConnectionStateFilter, "connection_state_filter", diags)
	}

	policy.Source.TrafficFilter = expandFirewallPolicyTrafficFilter(
		ctx,
		plan.SourceNetworkIDs,
		plan.SourceNetworkMatchOpposite,
		plan.SourcePortFilter,
		"source_network_ids",
		"source_port_filter",
		diags,
	)
	policy.Destination.TrafficFilter = expandFirewallPolicyTrafficFilter(
		ctx,
		plan.DestinationNetworkIDs,
		plan.DestinationNetworkMatchOpposite,
		plan.DestinationPortFilter,
		"destination_network_ids",
		"destination_port_filter",
		diags,
	)

	return policy
}

func validateFirewallPolicyModel(ctx context.Context, plan firewallPolicyResourceModel) error {
	if value := plan.Action.ValueString(); value != "ALLOW" && value != "BLOCK" && value != "REJECT" {
		return fmt.Errorf("action must be one of ALLOW, BLOCK, or REJECT")
	}

	if value := plan.IPVersion.ValueString(); value != "IPV4" && value != "IPV6" && value != "IPV4_AND_IPV6" {
		return fmt.Errorf("ip_version must be one of IPV4, IPV6, or IPV4_AND_IPV6")
	}

	if !plan.IPsecFilter.IsNull() {
		value := plan.IPsecFilter.ValueString()
		if value != "MATCH_ENCRYPTED" && value != "MATCH_NOT_ENCRYPTED" {
			return fmt.Errorf("ipsec_filter must be one of MATCH_ENCRYPTED or MATCH_NOT_ENCRYPTED")
		}
	}

	if err := validateFirewallPolicyEndpointFilters(ctx, "source", plan.SourceNetworkIDs, plan.SourceNetworkMatchOpposite, plan.SourcePortFilter); err != nil {
		return err
	}
	if err := validateFirewallPolicyEndpointFilters(ctx, "destination", plan.DestinationNetworkIDs, plan.DestinationNetworkMatchOpposite, plan.DestinationPortFilter); err != nil {
		return err
	}

	return nil
}

func validateFirewallPolicyEndpointFilters(ctx context.Context, endpointName string, networkIDs types.Set, networkMatchOpposite types.Bool, portFilter types.Object) error {
	decodedNetworkIDs, err := decodeFirewallPolicyNetworkIDs(ctx, networkIDs)
	if err != nil {
		return fmt.Errorf("unable to decode %s_network_ids", endpointName)
	}

	hasNetworkFilter := len(decodedNetworkIDs) > 0
	hasNetworkMatchOpposite := !networkMatchOpposite.IsNull() && !networkMatchOpposite.IsUnknown()
	hasPortFilter := !portFilter.IsNull() && !portFilter.IsUnknown()

	if hasNetworkFilter && hasPortFilter {
		return fmt.Errorf("%s endpoint cannot use both network and port filters", endpointName)
	}
	if !hasNetworkFilter && hasNetworkMatchOpposite {
		return fmt.Errorf("%s_network_match_opposite requires %s_network_ids", endpointName, endpointName)
	}

	if !hasPortFilter {
		return nil
	}

	model, err := decodeFirewallPolicyPortFilter(ctx, portFilter, endpointName+"_port_filter")
	if err != nil {
		return err
	}

	switch model.Type.ValueString() {
	case "PORTS":
		if !model.TrafficMatchingListID.IsNull() && model.TrafficMatchingListID.ValueString() != "" {
			return fmt.Errorf("%s_port_filter.traffic_matching_list_id is not valid when %s_port_filter.type is PORTS", endpointName, endpointName)
		}

		ports := setToStrings(ctx, model.Ports, endpointName+"_port_filter.ports", &diag.Diagnostics{})
		if len(ports) == 0 {
			return fmt.Errorf("%s_port_filter.ports must contain at least one port or range when %s_port_filter.type is PORTS", endpointName, endpointName)
		}
		if _, err := expandPortMatches(ports); err != nil {
			return fmt.Errorf("invalid %s_port_filter.ports: %w", endpointName, err)
		}
	case "TRAFFIC_MATCHING_LIST":
		if model.TrafficMatchingListID.IsNull() || model.TrafficMatchingListID.ValueString() == "" {
			return fmt.Errorf("%s_port_filter.traffic_matching_list_id is required when %s_port_filter.type is TRAFFIC_MATCHING_LIST", endpointName, endpointName)
		}

		ports, err := decodeFirewallPolicyPortSet(ctx, model.Ports)
		if err != nil {
			return fmt.Errorf("unable to decode %s_port_filter.ports", endpointName)
		}
		if len(ports) > 0 {
			return fmt.Errorf("%s_port_filter.ports is not valid when %s_port_filter.type is TRAFFIC_MATCHING_LIST", endpointName, endpointName)
		}
	default:
		return fmt.Errorf("%s_port_filter.type must be one of PORTS or TRAFFIC_MATCHING_LIST", endpointName)
	}

	return nil
}

func expandFirewallPolicyTrafficFilter(
	ctx context.Context,
	networkIDs types.Set,
	networkMatchOpposite types.Bool,
	portFilter types.Object,
	networkPath string,
	portFilterPath string,
	diags *diag.Diagnostics,
) *client.FirewallPolicyTrafficFilter {
	decodedNetworkIDs := setToStrings(ctx, networkIDs, networkPath, diags)
	if diags.HasError() {
		return nil
	}

	if len(decodedNetworkIDs) > 0 {
		return &client.FirewallPolicyTrafficFilter{
			Type: "NETWORK",
			NetworkFilter: &client.FirewallPolicyNetworkFilter{
				NetworkIDs:    decodedNetworkIDs,
				MatchOpposite: boolValueOrFalse(networkMatchOpposite),
			},
		}
	}

	if portFilter.IsNull() || portFilter.IsUnknown() {
		return nil
	}

	model, err := decodeFirewallPolicyPortFilter(ctx, portFilter, portFilterPath)
	if err != nil {
		diags.AddError("Invalid firewall policy port filter", err.Error())
		return nil
	}

	filter := &client.FirewallPolicyTrafficFilter{
		Type: "PORT",
		PortFilter: &client.FirewallPolicyPortFilter{
			Type:          model.Type.ValueString(),
			MatchOpposite: boolValueOrFalse(model.MatchOpposite),
		},
	}

	switch model.Type.ValueString() {
	case "PORTS":
		ports := setToStrings(ctx, model.Ports, portFilterPath+".ports", diags)
		if diags.HasError() {
			return nil
		}

		items, err := expandPortMatches(ports)
		if err != nil {
			diags.AddError("Invalid firewall policy port filter", fmt.Sprintf("%s.ports: %s", portFilterPath, err.Error()))
			return nil
		}
		filter.PortFilter.Items = items
	case "TRAFFIC_MATCHING_LIST":
		filter.PortFilter.TrafficMatchingListID = stringPointerValue(model.TrafficMatchingListID)
	default:
		diags.AddError("Invalid firewall policy port filter", fmt.Sprintf("%s.type must be one of PORTS or TRAFFIC_MATCHING_LIST", portFilterPath))
		return nil
	}

	return filter
}

func decodeFirewallPolicyNetworkIDs(ctx context.Context, value types.Set) ([]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var ids []string
	if diags := value.ElementsAs(ctx, &ids, false); diags.HasError() {
		return nil, fmt.Errorf("unable to decode set")
	}

	return ids, nil
}

func decodeFirewallPolicyPortSet(ctx context.Context, value types.Set) ([]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var ports []string
	if diags := value.ElementsAs(ctx, &ports, false); diags.HasError() {
		return nil, fmt.Errorf("unable to decode set")
	}

	return ports, nil
}

func decodeFirewallPolicyPortFilter(ctx context.Context, value types.Object, path string) (firewallPolicyPortFilterModel, error) {
	var model firewallPolicyPortFilterModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return model, fmt.Errorf("unable to decode %s", path)
	}

	return model, nil
}

func boolValueOrFalse(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}

func (r *firewallPolicyResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, firewallPolicy *client.FirewallPolicy) {
	connectionStates, diagnostics := stringSetValue(ctx, firewallPolicy.ConnectionStateFilter)
	diags.Append(diagnostics...)
	sourceNetworkIDs, diagnostics := extractFirewallNetworkIDs(ctx, firewallPolicy.Source)
	diags.Append(diagnostics...)
	destinationNetworkIDs, diagnostics := extractFirewallNetworkIDs(ctx, firewallPolicy.Destination)
	diags.Append(diagnostics...)
	sourcePortFilter, diagnostics := flattenFirewallPolicyPortFilter(ctx, firewallPolicy.Source)
	diags.Append(diagnostics...)
	destinationPortFilter, diagnostics := flattenFirewallPolicyPortFilter(ctx, firewallPolicy.Destination)
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	model := firewallPolicyResourceModel{
		ID:                              types.StringValue(firewallPolicy.ID),
		SiteID:                          siteID,
		Enabled:                         types.BoolValue(firewallPolicy.Enabled),
		Name:                            types.StringValue(firewallPolicy.Name),
		Description:                     nullableString(firewallPolicy.Description),
		Action:                          types.StringValue(firewallPolicy.Action.Type),
		SourceZoneID:                    types.StringValue(firewallPolicy.Source.ZoneID),
		SourceNetworkIDs:                sourceNetworkIDs,
		SourceNetworkMatchOpposite:      extractFirewallNetworkMatchOpposite(firewallPolicy.Source),
		SourcePortFilter:                sourcePortFilter,
		DestinationZoneID:               types.StringValue(firewallPolicy.Destination.ZoneID),
		DestinationNetworkIDs:           destinationNetworkIDs,
		DestinationNetworkMatchOpposite: extractFirewallNetworkMatchOpposite(firewallPolicy.Destination),
		DestinationPortFilter:           destinationPortFilter,
		IPVersion:                       types.StringValue(firewallPolicy.IPProtocolScope.IPVersion),
		ConnectionStateFilter:           connectionStates,
		IPsecFilter:                     nullableString(firewallPolicy.IPsecFilter),
		LoggingEnabled:                  types.BoolValue(firewallPolicy.LoggingEnabled),
		Index:                           types.Int64Value(firewallPolicy.Index),
	}

	diags.Append(state.Set(ctx, &model)...)
}

func extractFirewallNetworkIDs(ctx context.Context, endpoint *client.FirewallPolicyEndpoint) (types.Set, diag.Diagnostics) {
	if endpoint == nil || endpoint.TrafficFilter == nil || endpoint.TrafficFilter.Type != "NETWORK" || endpoint.TrafficFilter.NetworkFilter == nil {
		return types.SetNull(types.StringType), nil
	}

	values, diagnostics := types.SetValueFrom(ctx, types.StringType, endpoint.TrafficFilter.NetworkFilter.NetworkIDs)
	return values, diagnostics
}

func extractFirewallNetworkMatchOpposite(endpoint *client.FirewallPolicyEndpoint) types.Bool {
	if endpoint == nil || endpoint.TrafficFilter == nil || endpoint.TrafficFilter.Type != "NETWORK" || endpoint.TrafficFilter.NetworkFilter == nil {
		return types.BoolNull()
	}

	return types.BoolValue(endpoint.TrafficFilter.NetworkFilter.MatchOpposite)
}

func flattenFirewallPolicyPortFilter(ctx context.Context, endpoint *client.FirewallPolicyEndpoint) (types.Object, diag.Diagnostics) {
	if endpoint == nil || endpoint.TrafficFilter == nil || endpoint.TrafficFilter.Type != "PORT" || endpoint.TrafficFilter.PortFilter == nil {
		return types.ObjectNull(firewallPolicyPortFilterAttrTypes()), nil
	}

	ports, diagnostics := stringSetValue(ctx, flattenPortMatches(endpoint.TrafficFilter.PortFilter.Items))
	if diagnostics.HasError() {
		return types.ObjectNull(firewallPolicyPortFilterAttrTypes()), diagnostics
	}

	model := firewallPolicyPortFilterModel{
		Type:                  types.StringValue(endpoint.TrafficFilter.PortFilter.Type),
		MatchOpposite:         types.BoolValue(endpoint.TrafficFilter.PortFilter.MatchOpposite),
		Ports:                 ports,
		TrafficMatchingListID: nullableString(endpoint.TrafficFilter.PortFilter.TrafficMatchingListID),
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, firewallPolicyPortFilterAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}
