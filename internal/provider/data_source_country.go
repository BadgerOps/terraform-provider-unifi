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
	_ datasource.DataSource = (*countryDataSource)(nil)
)

type countryDataSource struct {
	clientProvider *providerData
}

type countryDataSourceModel struct {
	Code types.String `tfsdk:"code"`
	Name types.String `tfsdk:"name"`
}

func NewCountryDataSource() datasource.DataSource {
	return &countryDataSource{}
}

func (d *countryDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_country"
}

func (d *countryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi country definition by ISO 3166-1 alpha-2 `code` or `name`.",
		Attributes: map[string]schema.Attribute{
			"code": schema.StringAttribute{Optional: true, Computed: true},
			"name": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *countryDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *countryDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state countryDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateCodeOrNameLookup(state.Code, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid country lookup arguments", err.Error())
		return
	}

	countries, err := d.clientProvider.client.ListCountries(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi countries", err.Error())
		return
	}

	var matchCount int
	for _, country := range countries {
		if matchesCountryLookup(state, country) {
			matchCount++
			state.Code = types.StringValue(country.Code)
			state.Name = types.StringValue(country.Name)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Country not found", "No country matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple countries matched", fmt.Sprintf("%d countries matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesCountryLookup(state countryDataSourceModel, country client.Country) bool {
	switch {
	case !state.Code.IsNull() && state.Code.ValueString() != "":
		return country.Code == state.Code.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return country.Name == state.Name.ValueString()
	default:
		return false
	}
}
