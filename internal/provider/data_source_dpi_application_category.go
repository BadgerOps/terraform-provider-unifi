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
	_ datasource.DataSource = (*dpiApplicationCategoryDataSource)(nil)
)

type dpiApplicationCategoryDataSource struct {
	clientProvider *providerData
}

type dpiApplicationCategoryDataSourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewDPIApplicationCategoryDataSource() datasource.DataSource {
	return &dpiApplicationCategoryDataSource{}
}

func (d *dpiApplicationCategoryDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dpi_application_category"
}

func (d *dpiApplicationCategoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi DPI application category by numeric `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.Int64Attribute{Optional: true, Computed: true},
			"name": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *dpiApplicationCategoryDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *dpiApplicationCategoryDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state dpiApplicationCategoryDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateInt64OrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid DPI application category lookup arguments", err.Error())
		return
	}

	categories, err := d.clientProvider.client.ListDPIApplicationCategories(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi DPI application categories", err.Error())
		return
	}

	var matchCount int
	for _, category := range categories {
		if matchesDPIApplicationCategoryLookup(state, category) {
			matchCount++
			state.ID = types.Int64Value(category.ID)
			state.Name = types.StringValue(category.Name)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("DPI application category not found", "No DPI application category matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple DPI application categories matched", fmt.Sprintf("%d DPI application categories matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesDPIApplicationCategoryLookup(state dpiApplicationCategoryDataSourceModel, category client.DPIApplicationCategory) bool {
	switch {
	case !state.ID.IsNull():
		return category.ID == state.ID.ValueInt64()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return category.Name == state.Name.ValueString()
	default:
		return false
	}
}
