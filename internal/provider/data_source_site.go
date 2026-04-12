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
	_ datasource.DataSource              = (*siteDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*siteDataSource)(nil)
)

type siteDataSource struct {
	clientProvider *providerData
}

type siteDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	InternalReference types.String `tfsdk:"internal_reference"`
}

func NewSiteDataSource() datasource.DataSource {
	return &siteDataSource{}
}

func (d *siteDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_site"
}

func (d *siteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi site by `id`, `name`, or `internal_reference`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Site UUID.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Human-readable site name.",
			},
			"internal_reference": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Legacy internal site reference used by older APIs.",
			},
		},
	}
}

func (d *siteDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *siteDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state siteDataSourceModel
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
	if !state.InternalReference.IsNull() && state.InternalReference.ValueString() != "" {
		lookupCount++
	}

	if lookupCount != 1 {
		response.Diagnostics.AddError(
			"Invalid site lookup arguments",
			"Exactly one of `id`, `name`, or `internal_reference` must be set.",
		)
		return
	}

	sites, err := d.clientProvider.client.ListSites(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi sites", err.Error())
		return
	}

	var matchCount int
	for _, site := range sites {
		if matchesSite(state, site) {
			matchCount++
			state.ID = types.StringValue(site.ID)
			state.Name = types.StringValue(site.Name)
			state.InternalReference = types.StringValue(site.InternalReference)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Site not found", "No site matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple sites matched", fmt.Sprintf("%d sites matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesSite(state siteDataSourceModel, site client.Site) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return site.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return site.Name == state.Name.ValueString()
	case !state.InternalReference.IsNull() && state.InternalReference.ValueString() != "":
		return site.InternalReference == state.InternalReference.ValueString()
	default:
		return false
	}
}
