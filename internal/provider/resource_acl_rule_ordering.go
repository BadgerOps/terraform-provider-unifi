package provider

import (
	"context"
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
	_ resource.Resource                = (*aclRuleOrderingResource)(nil)
	_ resource.ResourceWithConfigure   = (*aclRuleOrderingResource)(nil)
	_ resource.ResourceWithImportState = (*aclRuleOrderingResource)(nil)
)

type aclRuleOrderingResource struct {
	providerData *providerData
}

type aclRuleOrderingResourceModel struct {
	ID                types.String `tfsdk:"id"`
	SiteID            types.String `tfsdk:"site_id"`
	OrderedACLRuleIDs types.List   `tfsdk:"ordered_acl_rule_ids"`
}

func NewACLRuleOrderingResource() resource.Resource {
	return &aclRuleOrderingResource{}
}

func (r *aclRuleOrderingResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_acl_rule_ordering"
}

func (r *aclRuleOrderingResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage the ordered ACL rule list for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic resource ID equal to `site_id`.",
			},
			"site_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Site UUID whose ACL rule ordering is managed.",
			},
			"ordered_acl_rule_ids": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Ordered ACL rule UUIDs from highest priority to lowest priority.",
			},
		},
	}
}

func (r *aclRuleOrderingResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *aclRuleOrderingResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan aclRuleOrderingResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	r.applyOrdering(ctx, plan, &response.State, &response.Diagnostics)
}

func (r *aclRuleOrderingResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state aclRuleOrderingResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	ordering, err := r.providerData.client.GetACLRuleOrdering(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to read ACL rule ordering", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID.ValueString(), ordering)
}

func (r *aclRuleOrderingResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan aclRuleOrderingResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	r.applyOrdering(ctx, plan, &response.State, &response.Diagnostics)
}

func (r *aclRuleOrderingResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *aclRuleOrderingResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	siteID := strings.TrimSpace(request.ID)
	if siteID == "" {
		response.Diagnostics.AddError("Invalid import identifier", "Expected import ID in the form <site_id>.")
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("site_id"), siteID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("id"), siteID)...)
}

func (r *aclRuleOrderingResource) applyOrdering(ctx context.Context, plan aclRuleOrderingResourceModel, state *tfsdk.State, diags *diag.Diagnostics) {
	orderedRuleIDs := listToStrings(ctx, plan.OrderedACLRuleIDs, "ordered_acl_rule_ids", diags)
	if diags.HasError() {
		return
	}

	ordering, err := r.providerData.client.UpdateACLRuleOrdering(ctx, plan.SiteID.ValueString(), client.ACLRuleOrdering{
		OrderedACLRuleIDs: orderedRuleIDs,
	})
	if err != nil {
		diags.AddError("Unable to update ACL rule ordering", err.Error())
		return
	}

	r.writeState(ctx, state, diags, plan.SiteID.ValueString(), ordering)
}

func (r *aclRuleOrderingResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID string, ordering *client.ACLRuleOrdering) {
	orderedRuleIDs, valueDiags := stringListValue(ctx, ordering.OrderedACLRuleIDs)
	diags.Append(valueDiags...)
	if diags.HasError() {
		return
	}

	diags.Append(state.Set(ctx, &aclRuleOrderingResourceModel{
		ID:                types.StringValue(siteID),
		SiteID:            types.StringValue(siteID),
		OrderedACLRuleIDs: orderedRuleIDs,
	})...)
}
