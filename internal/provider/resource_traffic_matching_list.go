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
	ID            types.String `tfsdk:"id"`
	SiteID        types.String `tfsdk:"site_id"`
	Type          types.String `tfsdk:"type"`
	Name          types.String `tfsdk:"name"`
	Ports         types.Set    `tfsdk:"ports"`
	IPv4Addresses types.Set    `tfsdk:"ipv4_addresses"`
	IPv6Addresses types.Set    `tfsdk:"ipv6_addresses"`
}

func NewTrafficMatchingListResource() resource.Resource {
	return &trafficMatchingListResource{}
}

func (r *trafficMatchingListResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_traffic_matching_list"
}

func (r *trafficMatchingListResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi traffic matching list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Traffic matching list type. Supported values: `PORTS`, `IPV4_ADDRESSES`, `IPV6_ADDRESSES`.",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"ports": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Port numbers or ranges such as `80`, `443`, or `10000-20000`. Required when `type` is `PORTS`.",
			},
			"ipv4_addresses": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IPv4 addresses, CIDR subnets, or ranges such as `192.168.1.10`, `192.168.1.0/24`, or `192.168.1.10-192.168.1.20`. Required when `type` is `IPV4_ADDRESSES`.",
			},
			"ipv6_addresses": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IPv6 addresses or CIDR subnets such as `2001:db8::10` or `2001:db8::/64`. Required when `type` is `IPV6_ADDRESSES`.",
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

	ports := setToStrings(ctx, model.Ports, "ports", diags)
	ipv4Addresses := setToStrings(ctx, model.IPv4Addresses, "ipv4_addresses", diags)
	ipv6Addresses := setToStrings(ctx, model.IPv6Addresses, "ipv6_addresses", diags)
	if diags.HasError() {
		return result
	}

	var (
		values []string
		path   string
	)

	switch result.Type {
	case "PORTS":
		values = ports
		path = "ports"
		if len(ipv4Addresses) > 0 || len(ipv6Addresses) > 0 {
			diags.AddError("Invalid traffic matching list configuration", "`ipv4_addresses` and `ipv6_addresses` are only valid for IP-based traffic matching lists.")
			return result
		}
	case "IPV4_ADDRESSES":
		values = ipv4Addresses
		path = "ipv4_addresses"
		if len(ports) > 0 || len(ipv6Addresses) > 0 {
			diags.AddError("Invalid traffic matching list configuration", "`ports` and `ipv6_addresses` are not valid when `type` is `IPV4_ADDRESSES`.")
			return result
		}
	case "IPV6_ADDRESSES":
		values = ipv6Addresses
		path = "ipv6_addresses"
		if len(ports) > 0 || len(ipv4Addresses) > 0 {
			diags.AddError("Invalid traffic matching list configuration", "`ports` and `ipv4_addresses` are not valid when `type` is `IPV6_ADDRESSES`.")
			return result
		}
	default:
		diags.AddError("Unsupported traffic matching list type", "Supported traffic matching list types are `PORTS`, `IPV4_ADDRESSES`, and `IPV6_ADDRESSES`.")
		return result
	}

	if len(values) == 0 {
		diags.AddError("Missing traffic matching list items", fmt.Sprintf("`%s` must contain at least one entry when `type` is `%s`.", path, result.Type))
		return result
	}

	items, err := expandTrafficMatchingListItems(result.Type, values)
	if err != nil {
		diags.AddError("Invalid traffic matching list items", err.Error())
		return result
	}
	result.Items = items

	return result
}

func (r *trafficMatchingListResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, list *client.TrafficMatchingList) {
	values, err := flattenTrafficMatchingListItems(list.Type, list.Items)
	if err != nil {
		diags.AddError("Unable to flatten traffic matching list", err.Error())
		return
	}

	ports := types.SetNull(types.StringType)
	ipv4Addresses := types.SetNull(types.StringType)
	ipv6Addresses := types.SetNull(types.StringType)

	var diagnostics diag.Diagnostics
	switch list.Type {
	case "PORTS":
		ports, diagnostics = stringSetValue(ctx, values)
	case "IPV4_ADDRESSES":
		ipv4Addresses, diagnostics = stringSetValue(ctx, values)
	case "IPV6_ADDRESSES":
		ipv6Addresses, diagnostics = stringSetValue(ctx, values)
	default:
		diags.AddError("Unsupported traffic matching list type", fmt.Sprintf("Unable to persist unsupported traffic matching list type %q.", list.Type))
		return
	}
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	model := trafficMatchingListResourceModel{
		ID:            types.StringValue(list.ID),
		SiteID:        siteID,
		Type:          types.StringValue(list.Type),
		Name:          types.StringValue(list.Name),
		Ports:         ports,
		IPv4Addresses: ipv4Addresses,
		IPv6Addresses: ipv6Addresses,
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
