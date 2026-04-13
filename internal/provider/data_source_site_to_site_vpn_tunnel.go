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
	_ datasource.DataSource              = (*siteToSiteVPNTunnelDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*siteToSiteVPNTunnelDataSource)(nil)
)

type siteToSiteVPNTunnelDataSource struct {
	clientProvider *providerData
}

type siteToSiteVPNTunnelDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Origin types.String `tfsdk:"origin"`
}

func NewSiteToSiteVPNTunnelDataSource() datasource.DataSource {
	return &siteToSiteVPNTunnelDataSource{}
}

func (d *siteToSiteVPNTunnelDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_site_to_site_vpn_tunnel"
}

func (d *siteToSiteVPNTunnelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi site-to-site VPN tunnel by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Optional: true, Computed: true},
			"site_id": schema.StringAttribute{Required: true},
			"name":    schema.StringAttribute{Optional: true, Computed: true},
			"type":    schema.StringAttribute{Computed: true},
			"origin":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *siteToSiteVPNTunnelDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *siteToSiteVPNTunnelDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state siteToSiteVPNTunnelDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid site-to-site VPN tunnel lookup arguments", err.Error())
		return
	}

	tunnels, err := d.clientProvider.client.ListSiteToSiteVPNTunnels(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi site-to-site VPN tunnels", err.Error())
		return
	}

	var matchCount int
	for _, tunnel := range tunnels {
		if matchesSiteToSiteVPNTunnelLookup(state, tunnel) {
			matchCount++
			state.ID = types.StringValue(tunnel.ID)
			state.Name = types.StringValue(tunnel.Name)
			state.Type = types.StringValue(tunnel.Type)
			state.Origin = types.StringValue(tunnel.Metadata.Origin)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Site-to-site VPN tunnel not found", "No site-to-site VPN tunnel matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple site-to-site VPN tunnels matched", fmt.Sprintf("%d site-to-site VPN tunnels matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesSiteToSiteVPNTunnelLookup(state siteToSiteVPNTunnelDataSourceModel, tunnel client.SiteToSiteVPNTunnel) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return tunnel.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return tunnel.Name == state.Name.ValueString()
	default:
		return false
	}
}
