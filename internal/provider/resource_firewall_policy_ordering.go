package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*firewallPolicyOrderingResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallPolicyOrderingResource)(nil)
	_ resource.ResourceWithImportState = (*firewallPolicyOrderingResource)(nil)
)

type firewallPolicyOrderingResource struct {
	providerData *providerData
}

type firewallPolicyOrderingResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	SiteID                       types.String `tfsdk:"site_id"`
	SourceZoneID                 types.String `tfsdk:"source_zone_id"`
	DestinationZoneID            types.String `tfsdk:"destination_zone_id"`
	BeforeSystemDefinedPolicyIDs types.List   `tfsdk:"before_system_defined_policy_ids"`
	AfterSystemDefinedPolicyIDs  types.List   `tfsdk:"after_system_defined_policy_ids"`
}

func NewFirewallPolicyOrderingResource() resource.Resource {
	return &firewallPolicyOrderingResource{}
}

func (r *firewallPolicyOrderingResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_policy_ordering"
}

func (r *firewallPolicyOrderingResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage firewall policy ordering for a source and destination firewall zone pair.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic resource ID in the form `<site_id>/<source_zone_id>/<destination_zone_id>`.",
			},
			"site_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Site UUID whose firewall policy ordering is managed.",
			},
			"source_zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source firewall zone UUID for the ordering scope.",
			},
			"destination_zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination firewall zone UUID for the ordering scope.",
			},
			"before_system_defined_policy_ids": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Ordered user-defined firewall policy UUIDs that must run before system-defined policies.",
			},
			"after_system_defined_policy_ids": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Ordered user-defined firewall policy UUIDs that must run after system-defined policies.",
			},
		},
	}
}

func (r *firewallPolicyOrderingResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *firewallPolicyOrderingResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan firewallPolicyOrderingResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	r.applyOrdering(ctx, plan, &response.State, &response.Diagnostics)
}

func (r *firewallPolicyOrderingResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state firewallPolicyOrderingResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	ordering, err := r.providerData.client.GetFirewallPolicyOrdering(
		ctx,
		state.SiteID.ValueString(),
		state.SourceZoneID.ValueString(),
		state.DestinationZoneID.ValueString(),
	)
	if err != nil {
		response.Diagnostics.AddError("Unable to read firewall policy ordering", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID.ValueString(), state.SourceZoneID.ValueString(), state.DestinationZoneID.ValueString(), ordering)
}

func (r *firewallPolicyOrderingResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan firewallPolicyOrderingResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	r.applyOrdering(ctx, plan, &response.State, &response.Diagnostics)
}

func (r *firewallPolicyOrderingResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *firewallPolicyOrderingResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	siteID, sourceZoneID, destinationZoneID, err := parseFirewallPolicyOrderingImportID(request.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid import identifier", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("site_id"), siteID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("source_zone_id"), sourceZoneID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("destination_zone_id"), destinationZoneID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("id"), firewallPolicyOrderingResourceID(siteID, sourceZoneID, destinationZoneID))...)
}

func (r *firewallPolicyOrderingResource) applyOrdering(ctx context.Context, plan firewallPolicyOrderingResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	beforeIDs := listToStrings(ctx, plan.BeforeSystemDefinedPolicyIDs, "before_system_defined_policy_ids", diags)
	afterIDs := listToStrings(ctx, plan.AfterSystemDefinedPolicyIDs, "after_system_defined_policy_ids", diags)
	if diags.HasError() {
		return
	}

	ordering, err := r.providerData.client.UpdateFirewallPolicyOrdering(
		ctx,
		plan.SiteID.ValueString(),
		plan.SourceZoneID.ValueString(),
		plan.DestinationZoneID.ValueString(),
		client.FirewallPolicyOrdering{
			OrderedFirewallPolicyIDs: client.FirewallPolicyOrderedIDs{
				BeforeSystemDefined: beforeIDs,
				AfterSystemDefined:  afterIDs,
			},
		},
	)
	if err != nil {
		diags.AddError("Unable to update firewall policy ordering", err.Error())
		return
	}

	r.writeState(ctx, state, diags, plan.SiteID.ValueString(), plan.SourceZoneID.ValueString(), plan.DestinationZoneID.ValueString(), ordering)
}

func (r *firewallPolicyOrderingResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID, sourceZoneID, destinationZoneID string, ordering *client.FirewallPolicyOrdering) {
	beforeIDs, valueDiags := stringListValue(ctx, ordering.OrderedFirewallPolicyIDs.BeforeSystemDefined)
	diags.Append(valueDiags...)
	afterIDs, valueDiags := stringListValue(ctx, ordering.OrderedFirewallPolicyIDs.AfterSystemDefined)
	diags.Append(valueDiags...)
	if diags.HasError() {
		return
	}

	diags.Append(state.Set(ctx, &firewallPolicyOrderingResourceModel{
		ID:                           types.StringValue(firewallPolicyOrderingResourceID(siteID, sourceZoneID, destinationZoneID)),
		SiteID:                       types.StringValue(siteID),
		SourceZoneID:                 types.StringValue(sourceZoneID),
		DestinationZoneID:            types.StringValue(destinationZoneID),
		BeforeSystemDefinedPolicyIDs: beforeIDs,
		AfterSystemDefinedPolicyIDs:  afterIDs,
	})...)
}

func parseFirewallPolicyOrderingImportID(raw string) (string, string, string, error) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected import ID in the form <site_id>/<source_zone_id>/<destination_zone_id>")
	}

	return parts[0], parts[1], parts[2], nil
}

func firewallPolicyOrderingResourceID(siteID, sourceZoneID, destinationZoneID string) string {
	return fmt.Sprintf("%s/%s/%s", siteID, sourceZoneID, destinationZoneID)
}
