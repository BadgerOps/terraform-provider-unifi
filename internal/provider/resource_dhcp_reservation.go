package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*dhcpReservationResource)(nil)
	_ resource.ResourceWithConfigure   = (*dhcpReservationResource)(nil)
	_ resource.ResourceWithImportState = (*dhcpReservationResource)(nil)
)

type dhcpReservationResource struct {
	providerData *providerData
}

type dhcpReservationResourceModel struct {
	ID         types.String `tfsdk:"id"`
	SiteID     types.String `tfsdk:"site_id"`
	MACAddress types.String `tfsdk:"mac_address"`
	FixedIP    types.String `tfsdk:"fixed_ip"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func NewDHCPReservationResource() resource.Resource {
	return &dhcpReservationResource{}
}

func (r *dhcpReservationResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dhcp_reservation"
}

func (r *dhcpReservationResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi DHCP reservation for a client MAC address. This resource uses the legacy Network client database path because the current integration API snapshot does not expose DHCP reservation writes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the form `<site_id>/<mac_address>`.",
			},
			"site_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Site UUID that owns the reservation.",
			},
			"mac_address": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Client MAC address used to locate the reservation target in the controller client database.",
			},
			"fixed_ip": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Reserved IPv4 address to assign when the reservation is enabled.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the reservation is active on the controller.",
			},
		},
	}
}

func (r *dhcpReservationResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *dhcpReservationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan dhcpReservationResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	reservation, err := r.providerData.client.UpsertDHCPReservation(
		ctx,
		plan.SiteID.ValueString(),
		plan.MACAddress.ValueString(),
		plan.FixedIP.ValueString(),
		plan.Enabled.ValueBool(),
	)
	if err != nil {
		response.Diagnostics.AddError("Unable to create DHCP reservation", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, plan.MACAddress, plan.FixedIP, reservation)
}

func (r *dhcpReservationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state dhcpReservationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	reservation, err := r.providerData.client.GetDHCPReservation(ctx, state.SiteID.ValueString(), state.MACAddress.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to read DHCP reservation", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, state.MACAddress, state.FixedIP, reservation)
}

func (r *dhcpReservationResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan dhcpReservationResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	reservation, err := r.providerData.client.UpsertDHCPReservation(
		ctx,
		plan.SiteID.ValueString(),
		plan.MACAddress.ValueString(),
		plan.FixedIP.ValueString(),
		plan.Enabled.ValueBool(),
	)
	if err != nil {
		response.Diagnostics.AddError("Unable to update DHCP reservation", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, plan.MACAddress, plan.FixedIP, reservation)
}

func (r *dhcpReservationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state dhcpReservationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteDHCPReservation(ctx, state.SiteID.ValueString(), state.MACAddress.ValueString())
	if err != nil && !client.IsMissingClient(err) {
		response.Diagnostics.AddError("Unable to delete DHCP reservation", err.Error())
	}
}

func (r *dhcpReservationResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	siteID, macAddress, err := parseCompositeImportID(request.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid import identifier", err.Error())
		return
	}

	normalizedID := dhcpReservationResourceID(siteID, macAddress)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("site_id"), siteID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("mac_address"), normalizeMACAddress(macAddress))...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("id"), normalizedID)...)
}

func (r *dhcpReservationResource) writeState(
	ctx context.Context,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	siteID types.String,
	fallbackMACAddress types.String,
	fallbackFixedIP types.String,
	reservation *client.DHCPReservation,
) {
	fixedIP := nullableString(reservation.FixedIP)
	if reservation.FixedIP == nil && !fallbackFixedIP.IsNull() && !fallbackFixedIP.IsUnknown() {
		fixedIP = fallbackFixedIP
	}

	macAddress := reservation.MACAddress
	if !fallbackMACAddress.IsNull() && !fallbackMACAddress.IsUnknown() && strings.EqualFold(fallbackMACAddress.ValueString(), reservation.MACAddress) {
		macAddress = fallbackMACAddress.ValueString()
	}

	model := dhcpReservationResourceModel{
		ID:         types.StringValue(dhcpReservationResourceID(siteID.ValueString(), reservation.MACAddress)),
		SiteID:     siteID,
		MACAddress: types.StringValue(macAddress),
		FixedIP:    fixedIP,
		Enabled:    types.BoolValue(reservation.Enabled),
	}

	diags.Append(state.Set(ctx, &model)...)
}

func dhcpReservationResourceID(siteID, macAddress string) string {
	return fmt.Sprintf("%s/%s", siteID, normalizeMACAddress(macAddress))
}

func normalizeMACAddress(macAddress string) string {
	return strings.ToLower(strings.TrimSpace(macAddress))
}
