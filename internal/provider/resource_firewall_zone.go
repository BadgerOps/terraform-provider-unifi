package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*firewallZoneResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallZoneResource)(nil)
	_ resource.ResourceWithImportState = (*firewallZoneResource)(nil)
)

type firewallZoneResource struct {
	providerData *providerData
}

type firewallZoneResourceModel struct {
	ID         types.String `tfsdk:"id"`
	SiteID     types.String `tfsdk:"site_id"`
	Name       types.String `tfsdk:"name"`
	NetworkIDs types.Set    `tfsdk:"network_ids"`
}

func NewFirewallZoneResource() resource.Resource {
	return &firewallZoneResource{}
}

func (r *firewallZoneResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_zone"
}

func (r *firewallZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi firewall zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Firewall zone UUID.",
			},
			"site_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Site UUID that owns the firewall zone.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Firewall zone name.",
			},
			"network_ids": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Network UUIDs attached to the zone.",
			},
		},
	}
}

func (r *firewallZoneResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *firewallZoneResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan firewallZoneResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiZone := client.FirewallZone{
		Name:       plan.Name.ValueString(),
		NetworkIDs: setToStrings(ctx, plan.NetworkIDs, "network_ids", &response.Diagnostics),
	}
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateFirewallZone(ctx, plan.SiteID.ValueString(), apiZone)
	if err != nil {
		response.Diagnostics.AddError("Unable to create firewall zone", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *firewallZoneResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state firewallZoneResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallZone, err := r.providerData.client.GetFirewallZone(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read firewall zone", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, firewallZone)
}

func (r *firewallZoneResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan firewallZoneResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state firewallZoneResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiZone := client.FirewallZone{
		Name:       plan.Name.ValueString(),
		NetworkIDs: setToStrings(ctx, plan.NetworkIDs, "network_ids", &response.Diagnostics),
	}
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateFirewallZone(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiZone)
	if err != nil {
		response.Diagnostics.AddError("Unable to update firewall zone", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *firewallZoneResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state firewallZoneResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteFirewallZone(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete firewall zone", err.Error())
	}
}

func (r *firewallZoneResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func (r *firewallZoneResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, firewallZone *client.FirewallZone) {
	networkIDs, diagnostics := types.SetValueFrom(ctx, types.StringType, firewallZone.NetworkIDs)
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	model := firewallZoneResourceModel{
		ID:         types.StringValue(firewallZone.ID),
		SiteID:     siteID,
		Name:       types.StringValue(firewallZone.Name),
		NetworkIDs: networkIDs,
	}

	diags.Append(state.Set(ctx, &model)...)
}
