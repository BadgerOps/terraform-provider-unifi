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
	_ datasource.DataSource = (*dpiApplicationDataSource)(nil)
)

type dpiApplicationDataSource struct {
	clientProvider *providerData
}

type dpiApplicationDataSourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewDPIApplicationDataSource() datasource.DataSource {
	return &dpiApplicationDataSource{}
}

func (d *dpiApplicationDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dpi_application"
}

func (d *dpiApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi DPI application by numeric `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.Int64Attribute{Optional: true, Computed: true},
			"name": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *dpiApplicationDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *dpiApplicationDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state dpiApplicationDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateInt64OrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid DPI application lookup arguments", err.Error())
		return
	}

	applications, err := d.clientProvider.client.ListDPIApplications(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi DPI applications", err.Error())
		return
	}

	var matchCount int
	for _, application := range applications {
		if matchesDPIApplicationLookup(state, application) {
			matchCount++
			state.ID = types.Int64Value(application.ID)
			state.Name = types.StringValue(application.Name)
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("DPI application not found", "No DPI application matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple DPI applications matched", fmt.Sprintf("%d DPI applications matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesDPIApplicationLookup(state dpiApplicationDataSourceModel, application client.DPIApplication) bool {
	switch {
	case !state.ID.IsNull():
		return application.ID == state.ID.ValueInt64()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return application.Name == state.Name.ValueString()
	default:
		return false
	}
}
