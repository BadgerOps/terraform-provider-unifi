package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ datasource.DataSource              = (*wanDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*wanDataSource)(nil)
)

type wanDataSource struct {
	clientProvider *providerData
}

type wanDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Name   types.String `tfsdk:"name"`
}

func NewWANDataSource() datasource.DataSource {
	return &wanDataSource{}
}

func (d *wanDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_wan"
}

func (d *wanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi WAN interface by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (d *wanDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *wanDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state wanDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid WAN lookup arguments", err.Error())
		return
	}

	wans, err := d.clientProvider.client.ListWANs(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi WAN interfaces", err.Error())
		return
	}

	var matchCount int
	for _, wan := range wans {
		if matchesWANLookup(state, wan) {
			matchCount++
			state.ID = types.StringValue(wan.ID)
			state.Name = types.StringValue(wan.Name)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("WAN interface not found", "No WAN interface matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple WAN interfaces matched", fmt.Sprintf("%d WAN interfaces matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesWANLookup(state wanDataSourceModel, wan client.WAN) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return wan.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return wan.Name == state.Name.ValueString()
	default:
		return false
	}
}
