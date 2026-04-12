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
	_ resource.Resource                = (*trafficMatchingListResource)(nil)
	_ resource.ResourceWithConfigure   = (*trafficMatchingListResource)(nil)
	_ resource.ResourceWithImportState = (*trafficMatchingListResource)(nil)
)

type trafficMatchingListResource struct {
	providerData *providerData
}

type trafficMatchingListResourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	Ports  types.Set    `tfsdk:"ports"`
}

func NewTrafficMatchingListResource() resource.Resource {
	return &trafficMatchingListResource{}
}

func (r *trafficMatchingListResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_traffic_matching_list"
}

func (r *trafficMatchingListResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi traffic matching list. The current provider implementation supports `type = \"PORTS\"`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Traffic matching list type. Currently only `PORTS` is supported.",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"ports": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Port numbers or ranges such as `80`, `443`, or `10000-20000`. Required when `type` is `PORTS`.",
			},
		},
	}
}

func (r *trafficMatchingListResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *trafficMatchingListResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan trafficMatchingListResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiList := expandTrafficMatchingList(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateTrafficMatchingList(ctx, plan.SiteID.ValueString(), apiList)
	if err != nil {
		response.Diagnostics.AddError("Unable to create traffic matching list", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *trafficMatchingListResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state trafficMatchingListResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := r.providerData.client.GetTrafficMatchingList(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read traffic matching list", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, list)
}

func (r *trafficMatchingListResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan trafficMatchingListResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state trafficMatchingListResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiList := expandTrafficMatchingList(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateTrafficMatchingList(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiList)
	if err != nil {
		response.Diagnostics.AddError("Unable to update traffic matching list", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *trafficMatchingListResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state trafficMatchingListResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteTrafficMatchingList(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete traffic matching list", err.Error())
	}
}

func (r *trafficMatchingListResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func expandTrafficMatchingList(ctx context.Context, model trafficMatchingListResourceModel, diags *diag.Diagnostics) client.TrafficMatchingList {
	result := client.TrafficMatchingList{
		Type: model.Type.ValueString(),
		Name: model.Name.ValueString(),
	}

	if result.Type != "PORTS" {
		diags.AddError("Unsupported traffic matching list type", "The current provider implementation only supports `PORTS` traffic matching lists.")
		return result
	}

	ports := setToStrings(ctx, model.Ports, "ports", diags)
	if len(ports) == 0 {
		diags.AddError("Missing ports", "`ports` must contain at least one port or range when `type` is `PORTS`.")
		return result
	}

	items, err := expandPortMatches(ports)
	if err != nil {
		diags.AddError("Invalid ports", err.Error())
		return result
	}
	result.Items = items

	return result
}

func (r *trafficMatchingListResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, list *client.TrafficMatchingList) {
	ports, diagnostics := stringSetValue(ctx, flattenPortMatches(list.Items))
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	model := trafficMatchingListResourceModel{
		ID:     types.StringValue(list.ID),
		SiteID: siteID,
		Type:   types.StringValue(list.Type),
		Name:   types.StringValue(list.Name),
		Ports:  ports,
	}

	diags.Append(state.Set(ctx, &model)...)
}

func validateTrafficMatchingListLookup(id, name types.String) error {
	lookupCount := 0
	if !id.IsNull() && id.ValueString() != "" {
		lookupCount++
	}
	if !name.IsNull() && name.ValueString() != "" {
		lookupCount++
	}
	if lookupCount != 1 {
		return fmt.Errorf("exactly one of `id` or `name` must be set")
	}
	return nil
}
