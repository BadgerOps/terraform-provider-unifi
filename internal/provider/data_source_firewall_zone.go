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
	_ datasource.DataSource              = (*firewallZoneDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*firewallZoneDataSource)(nil)
)

type firewallZoneDataSource struct {
	clientProvider *providerData
}

type firewallZoneDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	SiteID     types.String `tfsdk:"site_id"`
	Name       types.String `tfsdk:"name"`
	NetworkIDs types.Set    `tfsdk:"network_ids"`
}

func NewFirewallZoneDataSource() datasource.DataSource {
	return &firewallZoneDataSource{}
}

func (d *firewallZoneDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_zone"
}

func (d *firewallZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi firewall zone by `id` or `name` within a site.",
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
			"network_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *firewallZoneDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *firewallZoneDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state firewallZoneDataSourceModel
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
			"Invalid firewall zone lookup arguments",
			"Exactly one of `id` or `name` must be set.",
		)
		return
	}

	zones, err := d.clientProvider.client.ListFirewallZones(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi firewall zones", err.Error())
		return
	}

	var matchCount int
	for _, zone := range zones {
		if matchesFirewallZone(state, zone) {
			matchCount++
			state.ID = types.StringValue(zone.ID)
			state.Name = types.StringValue(zone.Name)
			var diagnostics diag.Diagnostics
			state.NetworkIDs, diagnostics = stringSetValue(ctx, zone.NetworkIDs)
			response.Diagnostics.Append(diagnostics...)
			if response.Diagnostics.HasError() {
				return
			}
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Firewall zone not found", "No firewall zone matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple firewall zones matched", fmt.Sprintf("%d firewall zones matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesFirewallZone(state firewallZoneDataSourceModel, zone client.FirewallZone) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return zone.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return zone.Name == state.Name.ValueString()
	default:
		return false
	}
}
