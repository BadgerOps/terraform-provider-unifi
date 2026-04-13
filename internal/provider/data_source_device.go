package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ datasource.DataSource              = (*deviceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*deviceDataSource)(nil)
)

type deviceDataSource struct {
	clientProvider *providerData
}

type deviceDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	SiteID            types.String `tfsdk:"site_id"`
	Name              types.String `tfsdk:"name"`
	MacAddress        types.String `tfsdk:"mac_address"`
	RequiredFeature   types.String `tfsdk:"required_feature"`
	Model             types.String `tfsdk:"model"`
	IPAddress         types.String `tfsdk:"ip_address"`
	State             types.String `tfsdk:"state"`
	Supported         types.Bool   `tfsdk:"supported"`
	FirmwareUpdatable types.Bool   `tfsdk:"firmware_updatable"`
	FirmwareVersion   types.String `tfsdk:"firmware_version"`
	Features          types.Set    `tfsdk:"features"`
	Interfaces        types.Set    `tfsdk:"interfaces"`
}

func NewDeviceDataSource() datasource.DataSource {
	return &deviceDataSource{}
}

func (d *deviceDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_device"
}

func (d *deviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up an adopted UniFi device by `id`, `name`, or `mac_address` within a site. Use `required_feature` to constrain matches, for example `switching` for switches.",
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
			"mac_address": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"required_feature": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional device feature that must be present on the matched device. Common values include `switching`, `gateway`, and `accessPoint`.",
			},
			"model": schema.StringAttribute{
				Computed: true,
			},
			"ip_address": schema.StringAttribute{
				Computed: true,
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
			"supported": schema.BoolAttribute{
				Computed: true,
			},
			"firmware_updatable": schema.BoolAttribute{
				Computed: true,
			},
			"firmware_version": schema.StringAttribute{
				Computed: true,
			},
			"features": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"interfaces": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *deviceDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *deviceDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state deviceDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateDeviceLookup(state.ID, state.Name, state.MacAddress); err != nil {
		response.Diagnostics.AddError("Invalid device lookup arguments", err.Error())
		return
	}

	devices, err := d.clientProvider.client.ListDevices(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi devices", err.Error())
		return
	}

	var matchCount int
	for _, device := range devices {
		if matchesDeviceLookup(state, device) {
			matchCount++
			state.ID = types.StringValue(device.ID)
			state.Name = types.StringValue(device.Name)
			state.MacAddress = types.StringValue(device.MacAddress)
			state.Model = types.StringValue(device.Model)
			state.IPAddress = types.StringValue(device.IPAddress)
			state.State = types.StringValue(device.State)
			state.Supported = types.BoolValue(device.Supported)
			state.FirmwareUpdatable = types.BoolValue(device.FirmwareUpdatable)
			state.FirmwareVersion = nullableString(device.FirmwareVersion)

			var diagnostics diag.Diagnostics
			state.Features, diagnostics = stringSetValue(ctx, device.Features)
			response.Diagnostics.Append(diagnostics...)
			state.Interfaces, diagnostics = stringSetValue(ctx, device.Interfaces)
			response.Diagnostics.Append(diagnostics...)
			if response.Diagnostics.HasError() {
				return
			}
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Device not found", "No device matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple devices matched", fmt.Sprintf("%d devices matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func validateDeviceLookup(id, name, macAddress types.String) error {
	lookupCount := 0
	if !id.IsNull() && id.ValueString() != "" {
		lookupCount++
	}
	if !name.IsNull() && name.ValueString() != "" {
		lookupCount++
	}
	if !macAddress.IsNull() && macAddress.ValueString() != "" {
		lookupCount++
	}
	if lookupCount != 1 {
		return fmt.Errorf("exactly one of `id`, `name`, or `mac_address` must be set")
	}

	return nil
}

func matchesDeviceLookup(state deviceDataSourceModel, device client.Device) bool {
	requiredFeature := ""
	if !state.RequiredFeature.IsNull() && state.RequiredFeature.ValueString() != "" {
		requiredFeature = strings.TrimSpace(state.RequiredFeature.ValueString())
		if !deviceHasFeature(device, requiredFeature) {
			return false
		}
	}

	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return device.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return device.Name == state.Name.ValueString()
	case !state.MacAddress.IsNull() && state.MacAddress.ValueString() != "":
		return strings.EqualFold(device.MacAddress, state.MacAddress.ValueString())
	default:
		return false
	}
}

func deviceHasFeature(device client.Device, feature string) bool {
	for _, candidate := range device.Features {
		if strings.EqualFold(candidate, feature) {
			return true
		}
	}

	return false
}
