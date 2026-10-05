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
	_ datasource.DataSource              = (*switchStackDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*switchStackDataSource)(nil)
)

type switchStackDataSource struct {
	clientProvider *providerData
}

type switchStackDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	SiteID           types.String `tfsdk:"site_id"`
	Name             types.String `tfsdk:"name"`
	DeviceID         types.String `tfsdk:"device_id"`
	MemberDeviceIDs  types.Set    `tfsdk:"member_device_ids"`
	UnitMacAddresses types.Set    `tfsdk:"unit_mac_addresses"`
	LagIDs           types.Set    `tfsdk:"lag_ids"`
}

func NewSwitchStackDataSource() datasource.DataSource {
	return &switchStackDataSource{}
}

func (d *switchStackDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_switch_stack"
}

func (d *switchStackDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi switch stack by `id` or `name` within a site.",
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
			"device_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Device ID of the stack itself. Reported by UniFi Network `10.6` and newer; null on older controllers.",
			},
			"member_device_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Device IDs of the stack members. On UniFi Network `10.6` and newer the API reports stack units by MAC address, so these are resolved by matching `unit_mac_addresses` against the site's adopted devices.",
			},
			"unit_mac_addresses": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "MAC addresses of the stack units. Reported by UniFi Network `10.6` and newer; empty on older controllers.",
			},
			"lag_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *switchStackDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *switchStackDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state switchStackDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid switch stack lookup arguments", err.Error())
		return
	}

	stacks, err := d.clientProvider.client.ListSwitchStacks(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list UniFi switch stacks", err.Error())
		return
	}

	var matchCount int
	for _, stack := range stacks {
		if matchesSwitchStackLookup(state, stack) {
			matchCount++
			state.ID = types.StringValue(stack.ID)
			state.Name = types.StringValue(stack.Name)

			state.DeviceID = nullableString(stack.DeviceID)

			unitMacAddresses := flattenSwitchStackUnitMacAddresses(stack.Units)
			memberDeviceIDs := flattenSwitchStackMemberDeviceIDs(stack.Members)
			if len(memberDeviceIDs) == 0 && len(unitMacAddresses) > 0 {
				memberDeviceIDs, err = d.clientProvider.client.ResolveDeviceIDsByMAC(ctx, state.SiteID.ValueString(), unitMacAddresses)
				if err != nil {
					response.Diagnostics.AddError("Unable to resolve UniFi switch stack units", err.Error())
					return
				}
			}

			var diagnostics diag.Diagnostics
			state.MemberDeviceIDs, diagnostics = stringSetValue(ctx, memberDeviceIDs)
			response.Diagnostics.Append(diagnostics...)
			state.UnitMacAddresses, diagnostics = stringSetValue(ctx, unitMacAddresses)
			response.Diagnostics.Append(diagnostics...)
			state.LagIDs, diagnostics = stringSetValue(ctx, flattenSwitchStackLagIDs(stack.Lags))
			response.Diagnostics.Append(diagnostics...)
			if response.Diagnostics.HasError() {
				return
			}
		}
	}

	switch matchCount {
	case 0:
		response.Diagnostics.AddError("Switch stack not found", "No switch stack matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple switch stacks matched", fmt.Sprintf("%d switch stacks matched the given selector", matchCount))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func matchesSwitchStackLookup(state switchStackDataSourceModel, stack client.SwitchStack) bool {
	switch {
	case !state.ID.IsNull() && state.ID.ValueString() != "":
		return stack.ID == state.ID.ValueString()
	case !state.Name.IsNull() && state.Name.ValueString() != "":
		return stack.Name == state.Name.ValueString()
	default:
		return false
	}
}

func flattenSwitchStackUnitMacAddresses(units []client.SwitchStackUnit) []string {
	macAddresses := make([]string, 0, len(units))
	for _, unit := range units {
		macAddresses = append(macAddresses, unit.MacAddress)
	}

	return macAddresses
}

func flattenSwitchStackMemberDeviceIDs(members []client.SwitchStackMember) []string {
	deviceIDs := make([]string, 0, len(members))
	for _, member := range members {
		deviceIDs = append(deviceIDs, member.DeviceID)
	}

	return deviceIDs
}

func flattenSwitchStackLagIDs(lags []client.SwitchStackLag) []string {
	lagIDs := make([]string, 0, len(lags))
	for _, lag := range lags {
		lagIDs = append(lagIDs, lag.ID)
	}

	return lagIDs
}
