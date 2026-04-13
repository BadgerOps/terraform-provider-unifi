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
	_ datasource.DataSource              = (*vpnServerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vpnServerDataSource)(nil)
)

type vpnServerDataSource struct {
	clientProvider *providerData
}

type vpnServerDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	SiteID  types.String `tfsdk:"site_id"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Origin  types.String `tfsdk:"origin"`
}

func NewVPNServerDataSource() datasource.DataSource {
	return &vpnServerDataSource{}
}

func (d *vpnServerDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_vpn_server"
}

func (d *vpnServerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi VPN server by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Optional: true, Computed: true},
			"site_id": schema.StringAttribute{Required: true},
			"name":    schema.StringAttribute{Optional: true, Computed: true},
			"type":    schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{Computed: true},
			"origin":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *vpnServerDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *vpnServerDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state vpnServerDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid VPN server lookup arguments", err.Error())
		return
	}

	servers, err := d.clientProvider.client.ListVPNServers(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi VPN servers", err.Error())
		return
	}

	var matchCount int
	for _, server := range servers {
		if matchesVPNServerLookup(state, server) {
			matchCount++
			state.ID = types.StringValue(server.ID)
			state.Name = types.StringValue(server.Name)
			state.Type = types.StringValue(server.Type)
			state.Enabled = types.BoolValue(server.Enabled)
			state.Origin = types.StringValue(server.Metadata.Origin)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("VPN server not found", "No VPN server matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple VPN servers matched", fmt.Sprintf("%d VPN servers matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesVPNServerLookup(state vpnServerDataSourceModel, server client.VPNServer) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return server.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return server.Name == state.Name.ValueString()
	default:
		return false
	}
}
