package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*deviceManagementNetworkResource)(nil)
	_ resource.ResourceWithConfigure   = (*deviceManagementNetworkResource)(nil)
	_ resource.ResourceWithImportState = (*deviceManagementNetworkResource)(nil)
)

type deviceManagementNetworkResource struct{ providerData *providerData }

type deviceManagementNetworkModel struct {
	ID        types.String `tfsdk:"id"`
	SiteID    types.String `tfsdk:"site_id"`
	DeviceMAC types.String `tfsdk:"device_mac"`
	NetworkID types.String `tfsdk:"network_id"`
}

func NewDeviceManagementNetworkResource() resource.Resource {
	return &deviceManagementNetworkResource{}
}

func (r *deviceManagementNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_management_network"
}

func (r *deviceManagementNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the network a UniFi device itself lives in (its management network, the legacy device field `mgmt_network_id`), for example to keep a switch in the management network while its ports serve other networks. Changing it re-provisions the device, which then takes an address in the new network; the device's uplink must carry that network. Import with `<site_id>/<device_mac>`. Destroying the resource only removes it from Terraform state and leaves the device where it is.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "`<site_id>/<device_mac>`."},
			"site_id":    schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Integration site UUID."},
			"device_mac": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{lowerCaseMACValidator{}}, MarkdownDescription: "MAC address of the device in lower case, for example `0c:ea:14:00:00:01`."},
			"network_id": schema.StringAttribute{Required: true, MarkdownDescription: "The management network, as an integration network ID (`unifi_network.<name>.id`). The provider translates it to the legacy networkconf ID."},
		},
	}
}

func (r *deviceManagementNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *deviceManagementNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *deviceManagementNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *deviceManagementNetworkResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan deviceManagementNetworkModel
	diags.Append(planned.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	siteID := plan.SiteID.ValueString()
	legacyIDs, err := r.providerData.client.LegacyNetworkIDs(ctx, siteID)
	if err != nil {
		diags.AddError("Unable to read networks", err.Error())
		return
	}
	legacyID, ok := legacyIDs[plan.NetworkID.ValueString()]
	if !ok {
		diags.AddAttributeError(path.Root("network_id"), "Unknown network", fmt.Sprintf("No network with ID %q in site %s.", plan.NetworkID.ValueString(), siteID))
		return
	}
	stored, err := r.providerData.client.SetDeviceManagementNetwork(ctx, siteID, plan.DeviceMAC.ValueString(), legacyID)
	if err != nil {
		diags.AddError("Unable to update management network", err.Error())
		return
	}
	model, d := flattenDeviceManagementNetwork(siteID, plan.DeviceMAC.ValueString(), stored, legacyIDs)
	diags.Append(d...)
	diags.Append(state.Set(ctx, &model)...)
}

func (r *deviceManagementNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceManagementNetworkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	siteID := state.SiteID.ValueString()
	stored, err := r.providerData.client.GetDeviceManagementNetwork(ctx, siteID, state.DeviceMAC.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read management network", err.Error())
		return
	}
	legacyIDs, err := r.providerData.client.LegacyNetworkIDs(ctx, siteID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read networks", err.Error())
		return
	}
	model, d := flattenDeviceManagementNetwork(siteID, state.DeviceMAC.ValueString(), stored, legacyIDs)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves the device in its network: there is no neutral value to go
// back to, and moving a device away can cut it off from the controller.
func (r *deviceManagementNetworkResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *deviceManagementNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import identifier", "Import with <site_id>/<device_mac>.")
		return
	}
	mac := strings.ToLower(parts[1])
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), deviceManagementNetworkID(parts[0], mac))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_mac"), mac)...)
}

func deviceManagementNetworkID(siteID, mac string) string {
	return fmt.Sprintf("%s/%s", siteID, strings.ToLower(mac))
}

// flattenDeviceManagementNetwork reads an unset management network as null,
// so a configuration naming one shows the change instead of hiding it.
func flattenDeviceManagementNetwork(siteID, mac, legacyID string, legacyIDs map[string]string) (deviceManagementNetworkModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := deviceManagementNetworkModel{
		ID:        types.StringValue(deviceManagementNetworkID(siteID, mac)),
		SiteID:    types.StringValue(siteID),
		DeviceMAC: types.StringValue(mac),
		NetworkID: types.StringNull(),
	}
	if legacyID == "" {
		return model, diags
	}
	for integrationID, candidate := range legacyIDs {
		if candidate == legacyID {
			model.NetworkID = types.StringValue(integrationID)
			return model, diags
		}
	}
	diags.AddAttributeError(path.Root("network_id"), "Network without an integration ID", fmt.Sprintf("Device %s is managed in legacy network %s, which has no integration network ID.", mac, legacyID))
	return model, diags
}
