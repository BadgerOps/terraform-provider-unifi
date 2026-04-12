package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
	DestinationZoneID               types.String `tfsdk:"destination_zone_id"`
	DestinationNetworkIDs           types.Set    `tfsdk:"destination_network_ids"`
	DestinationNetworkMatchOpposite types.Bool   `tfsdk:"destination_network_match_opposite"`
	IPVersion                       types.String `tfsdk:"ip_version"`
	ConnectionStateFilter           types.Set    `tfsdk:"connection_state_filter"`
	IPsecFilter                     types.String `tfsdk:"ipsec_filter"`
	LoggingEnabled                  types.Bool   `tfsdk:"logging_enabled"`
	Index                           types.Int64  `tfsdk:"index"`
}

func NewFirewallPolicyResource() resource.Resource {
	return &firewallPolicyResource{}
}

func (r *firewallPolicyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_policy"
}

func (r *firewallPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi firewall policy with zone-based matching and optional source and destination network filters.",
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
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := r.expandPolicy(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateFirewallPolicy(ctx, plan.SiteID.ValueString(), plan.ID.ValueString(), apiPolicy)
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

	if !plan.ConnectionStateFilter.IsNull() {
		policy.ConnectionStateFilter = setToStrings(ctx, plan.ConnectionStateFilter, "connection_state_filter", diags)
	}

	sourceNetworkIDs := setToStrings(ctx, plan.SourceNetworkIDs, "source_network_ids", diags)
	if len(sourceNetworkIDs) > 0 {
		policy.Source.TrafficFilter = &client.FirewallPolicyNetworkTrafficFilter{
			Type: "NETWORK",
			NetworkFilter: client.FirewallPolicyNetworkFilter{
				NetworkIDs:    sourceNetworkIDs,
				MatchOpposite: !plan.SourceNetworkMatchOpposite.IsNull() && plan.SourceNetworkMatchOpposite.ValueBool(),
			},
		}
	}

	destinationNetworkIDs := setToStrings(ctx, plan.DestinationNetworkIDs, "destination_network_ids", diags)
	if len(destinationNetworkIDs) > 0 {
		policy.Destination.TrafficFilter = &client.FirewallPolicyNetworkTrafficFilter{
			Type: "NETWORK",
			NetworkFilter: client.FirewallPolicyNetworkFilter{
				NetworkIDs:    destinationNetworkIDs,
				MatchOpposite: !plan.DestinationNetworkMatchOpposite.IsNull() && plan.DestinationNetworkMatchOpposite.ValueBool(),
			},
		}
	}

	if err := validateFirewallPolicyModel(plan); err != nil {
		diags.AddError("Invalid firewall policy configuration", err.Error())
	}

	return policy
}

func validateFirewallPolicyModel(plan firewallPolicyResourceModel) error {
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

	return nil
}

func (r *firewallPolicyResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, firewallPolicy *client.FirewallPolicy) {
	connectionStates, diagnostics := stringSetValue(ctx, firewallPolicy.ConnectionStateFilter)
	diags.Append(diagnostics...)
	sourceNetworkIDs, diagnostics := extractFirewallNetworkIDs(ctx, firewallPolicy.Source)
	diags.Append(diagnostics...)
	destinationNetworkIDs, diagnostics := extractFirewallNetworkIDs(ctx, firewallPolicy.Destination)
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
		DestinationZoneID:               types.StringValue(firewallPolicy.Destination.ZoneID),
		DestinationNetworkIDs:           destinationNetworkIDs,
		DestinationNetworkMatchOpposite: extractFirewallNetworkMatchOpposite(firewallPolicy.Destination),
		IPVersion:                       types.StringValue(firewallPolicy.IPProtocolScope.IPVersion),
		ConnectionStateFilter:           connectionStates,
		IPsecFilter:                     nullableString(firewallPolicy.IPsecFilter),
		LoggingEnabled:                  types.BoolValue(firewallPolicy.LoggingEnabled),
		Index:                           types.Int64Value(firewallPolicy.Index),
	}

	diags.Append(state.Set(ctx, &model)...)
}

func extractFirewallNetworkIDs(ctx context.Context, endpoint *client.FirewallPolicyEndpoint) (types.Set, diag.Diagnostics) {
	if endpoint == nil || endpoint.TrafficFilter == nil {
		return types.SetNull(types.StringType), nil
	}

	values, diagnostics := types.SetValueFrom(ctx, types.StringType, endpoint.TrafficFilter.NetworkFilter.NetworkIDs)
	return values, diagnostics
}

func extractFirewallNetworkMatchOpposite(endpoint *client.FirewallPolicyEndpoint) types.Bool {
	if endpoint == nil || endpoint.TrafficFilter == nil {
		return types.BoolNull()
	}

	return types.BoolValue(endpoint.TrafficFilter.NetworkFilter.MatchOpposite)
}
