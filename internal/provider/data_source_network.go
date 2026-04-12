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
	_ datasource.DataSource              = (*networkDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networkDataSource)(nil)
)

type networkDataSource struct {
	clientProvider *providerData
}

type networkDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	SiteID     types.String `tfsdk:"site_id"`
	Management types.String `tfsdk:"management"`
	Name       types.String `tfsdk:"name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	VLANID     types.Int64  `tfsdk:"vlan_id"`
	Default    types.Bool   `tfsdk:"default"`
}

func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

func (d *networkDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_network"
}

func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi network by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"management": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Computed: true,
			},
			"vlan_id": schema.Int64Attribute{
				Computed: true,
			},
			"default": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (d *networkDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *networkDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state networkDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	lookupCount := 0
	if !state.ID.IsNull() && state.ID.ValueString() != "" {
		lookupCount++
	}
	if !state.Name.IsNull() && state.Name.ValueString() != "" {
		lookupCount++
	}

	if lookupCount != 1 {
		response.Diagnostics.AddError(
			"Invalid network lookup arguments",
			"Exactly one of `id` or `name` must be set.",
		)
		return
	}

	networks, err := d.clientProvider.client.ListNetworks(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi networks", err.Error())
		return
	}

	var matchCount int
	for _, network := range networks {
		if matchesNetwork(state, network) {
			matchCount++
			state.ID = types.StringValue(network.ID)
			state.Management = types.StringValue(network.Management)
			state.Name = types.StringValue(network.Name)
			state.Enabled = types.BoolValue(network.Enabled)
			state.VLANID = types.Int64Value(network.VLANID)
			state.Default = types.BoolValue(network.Default)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Network not found", "No network matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple networks matched", fmt.Sprintf("%d networks matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesNetwork(state networkDataSourceModel, network client.Network) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return network.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return network.Name == state.Name.ValueString()
	default:
		return false
	}
}
