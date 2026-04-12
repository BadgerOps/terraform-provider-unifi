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
	_ datasource.DataSource              = (*radiusProfileDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*radiusProfileDataSource)(nil)
)

type radiusProfileDataSource struct {
	clientProvider *providerData
}

type radiusProfileDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Name   types.String `tfsdk:"name"`
}

func NewRadiusProfileDataSource() datasource.DataSource {
	return &radiusProfileDataSource{}
}

func (d *radiusProfileDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_radius_profile"
}

func (d *radiusProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi RADIUS profile by `id` or `name` within a site.",
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

func (d *radiusProfileDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *radiusProfileDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state radiusProfileDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateTrafficMatchingListLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid RADIUS profile lookup arguments", err.Error())
		return
	}

	profiles, err := d.clientProvider.client.ListRadiusProfiles(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi RADIUS profiles", err.Error())
		return
	}

	var matchCount int
	for _, profile := range profiles {
		if matchesRadiusProfileLookup(state, profile) {
			matchCount++
			state.ID = types.StringValue(profile.ID)
			state.Name = types.StringValue(profile.Name)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("RADIUS profile not found", "No RADIUS profile matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple RADIUS profiles matched", fmt.Sprintf("%d RADIUS profiles matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesRadiusProfileLookup(state radiusProfileDataSourceModel, profile client.RadiusProfile) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return profile.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return profile.Name == state.Name.ValueString()
	default:
		return false
	}
}
