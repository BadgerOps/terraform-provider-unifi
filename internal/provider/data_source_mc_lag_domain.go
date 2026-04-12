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
	_ datasource.DataSource              = (*mcLagDomainDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*mcLagDomainDataSource)(nil)
)

type mcLagDomainDataSource struct {
	clientProvider *providerData
}

type mcLagDomainDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	SiteID        types.String `tfsdk:"site_id"`
	Name          types.String `tfsdk:"name"`
	PeerDeviceIDs types.Set    `tfsdk:"peer_device_ids"`
	LagIDs        types.Set    `tfsdk:"lag_ids"`
}

func NewMcLagDomainDataSource() datasource.DataSource {
	return &mcLagDomainDataSource{}
}

func (d *mcLagDomainDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mc_lag_domain"
}

func (d *mcLagDomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi MC-LAG domain by `id` or `name` within a site.",
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
			"peer_device_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"lag_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *mcLagDomainDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *mcLagDomainDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state mcLagDomainDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid MC-LAG domain lookup arguments", err.Error())
		return
	}

	domains, err := d.clientProvider.client.ListMcLagDomains(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi MC-LAG domains", err.Error())
		return
	}

	var matchCount int
	for _, domain := range domains {
		if matchesMcLagDomainLookup(state, domain) {
			matchCount++
			state.ID = types.StringValue(domain.ID)
			state.Name = types.StringValue(domain.Name)

			var diagnostics diag.Diagnostics
			state.PeerDeviceIDs, diagnostics = stringSetValue(ctx, flattenMcLagPeerDeviceIDs(domain.Peers))
			response.Diagnostics.Append(diagnostics...)
			state.LagIDs, diagnostics = stringSetValue(ctx, flattenMcLagLagIDs(domain.Lags))
			response.Diagnostics.Append(diagnostics...)
			if response.Diagnostics.HasError() {
				return
			}
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("MC-LAG domain not found", "No MC-LAG domain matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple MC-LAG domains matched", fmt.Sprintf("%d MC-LAG domains matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesMcLagDomainLookup(state mcLagDomainDataSourceModel, domain client.McLagDomain) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return domain.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return domain.Name == state.Name.ValueString()
	default:
		return false
	}
}

func flattenMcLagPeerDeviceIDs(peers []client.McLagPeer) []string {
	deviceIDs := make([]string, 0, len(peers))
	for _, peer := range peers {
		deviceIDs = append(deviceIDs, peer.DeviceID)
	}

	return deviceIDs
}

func flattenMcLagLagIDs(lags []client.McLagLocalLag) []string {
	lagIDs := make([]string, 0, len(lags))
	for _, lag := range lags {
		lagIDs = append(lagIDs, lag.ID)
	}

	return lagIDs
}
