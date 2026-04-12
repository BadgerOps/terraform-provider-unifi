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
	_ datasource.DataSource              = (*deviceTagDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*deviceTagDataSource)(nil)
)

type deviceTagDataSource struct {
	clientProvider *providerData
}

type deviceTagDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	SiteID    types.String `tfsdk:"site_id"`
	Name      types.String `tfsdk:"name"`
	DeviceIDs types.Set    `tfsdk:"device_ids"`
}

func NewDeviceTagDataSource() datasource.DataSource {
	return &deviceTagDataSource{}
}

func (d *deviceTagDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_device_tag"
}

func (d *deviceTagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi device tag by `id` or `name` within a site.",
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
			"device_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *deviceTagDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *deviceTagDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state deviceTagDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateTrafficMatchingListLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid device tag lookup arguments", err.Error())
		return
	}

	tags, err := d.clientProvider.client.ListDeviceTags(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi device tags", err.Error())
		return
	}

	var matchCount int
	for _, tag := range tags {
		if matchesDeviceTagLookup(state, tag) {
			matchCount++
			state.ID = types.StringValue(tag.ID)
			state.Name = types.StringValue(tag.Name)
			var diagnostics diag.Diagnostics
			state.DeviceIDs, diagnostics = stringSetValue(ctx, tag.DeviceIDs)
			response.Diagnostics.Append(diagnostics...)
			if response.Diagnostics.HasError() {
				return
			}
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Device tag not found", "No device tag matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple device tags matched", fmt.Sprintf("%d device tags matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesDeviceTagLookup(state deviceTagDataSourceModel, tag client.DeviceTag) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return tag.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return tag.Name == state.Name.ValueString()
	default:
		return false
	}
}
