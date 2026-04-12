package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ datasource.DataSource              = (*trafficMatchingListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*trafficMatchingListDataSource)(nil)
)

type trafficMatchingListDataSource struct {
	clientProvider *providerData
}

type trafficMatchingListDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	Ports  types.Set    `tfsdk:"ports"`
}

func NewTrafficMatchingListDataSource() datasource.DataSource {
	return &trafficMatchingListDataSource{}
}

func (d *trafficMatchingListDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_traffic_matching_list"
}

func (d *trafficMatchingListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi traffic matching list by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"ports": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *trafficMatchingListDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *trafficMatchingListDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state trafficMatchingListDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateTrafficMatchingListLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid traffic matching list lookup arguments", err.Error())
		return
	}

	lists, err := d.clientProvider.client.ListTrafficMatchingLists(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list traffic matching lists", err.Error())
		return
	}

	var matchCount int
	for _, list := range lists {
		if matchesTrafficMatchingListLookup(state, list) {
			matchCount++
			state.ID = types.StringValue(list.ID)
			state.Type = types.StringValue(list.Type)
			state.Name = types.StringValue(list.Name)
			var diagnostics diag.Diagnostics
			state.Ports, diagnostics = stringSetValue(ctx, flattenPortMatches(list.Items))
			response.Diagnostics.Append(diagnostics...)
			if response.Diagnostics.HasError() {
				return
			}
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Traffic matching list not found", "No traffic matching list matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple traffic matching lists matched", fmt.Sprintf("%d traffic matching lists matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesTrafficMatchingListLookup(state trafficMatchingListDataSourceModel, list client.TrafficMatchingList) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return list.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return list.Name == state.Name.ValueString()
	default:
		return false
	}
}
