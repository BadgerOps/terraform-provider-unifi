package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*devicePortOverrideResource)(nil)
	_ resource.ResourceWithConfigure   = (*devicePortOverrideResource)(nil)
	_ resource.ResourceWithImportState = (*devicePortOverrideResource)(nil)
)

type devicePortOverrideResource struct{ providerData *providerData }

type devicePortOverrideModel struct {
	ID                 types.String `tfsdk:"id"`
	SiteID             types.String `tfsdk:"site_id"`
	DeviceMAC          types.String `tfsdk:"device_mac"`
	PortIdx            types.Int64  `tfsdk:"port_idx"`
	Name               types.String `tfsdk:"name"`
	NativeNetworkID    types.String `tfsdk:"native_network_id"`
	TaggedVLANMgmt     types.String `tfsdk:"tagged_vlan_mgmt"`
	ExcludedNetworkIDs types.Set    `tfsdk:"excluded_network_ids"`
}

func NewDevicePortOverrideResource() resource.Resource { return &devicePortOverrideResource{} }

func (r *devicePortOverrideResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_port_override"
}

func (r *devicePortOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the settings of one switch port (an entry of the device's legacy `port_overrides`): its name, native network and tagged networks. Only this port's entry changes; the device's other ports and this port's unmodelled settings (PoE, STP, port security, rate limits, ...) are kept as the controller has them. Attributes left unset keep the controller's value. Import with `<site_id>/<device_mac>/<port_idx>`. Destroying the resource only removes it from Terraform state: the port keeps its settings, because resetting a port to its defaults moves whatever is plugged into it to the default network.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "`<site_id>/<device_mac>/<port_idx>`."},
			"site_id":    schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Integration site UUID."},
			"device_mac": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{lowerCaseMACValidator{}}, MarkdownDescription: "MAC address of the switch in lower case, for example `0c:ea:14:00:00:01`."},
			"port_idx":   schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, MarkdownDescription: "Port number as the controller counts it, starting at 1."},
			"name":       schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Port name shown in the UI."},
			"native_network_id": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Network carried untagged on the port, as an integration network ID (`unifi_network.<name>.id`). The provider translates it to the legacy networkconf ID.",
			},
			"tagged_vlan_mgmt": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringOneOf{"auto", "block_all", "custom"}},
				MarkdownDescription: "Tagged networks: `auto` (all), `block_all` (none, native only) or `custom` (all except `excluded_network_ids`).",
			},
			"excluded_network_ids": schema.SetAttribute{
				Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "With `tagged_vlan_mgmt = \"custom\"`: networks not carried tagged, as integration network IDs.",
			},
		},
	}
}

func (r *devicePortOverrideResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *devicePortOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *devicePortOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *devicePortOverrideResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan devicePortOverrideModel
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
	override := client.PortOverride{PortIdx: plan.PortIdx.ValueInt64()}
	if known(plan.Name) {
		override.Name = plan.Name.ValueString()
	}
	if known(plan.NativeNetworkID) {
		legacyID, ok := legacyIDs[plan.NativeNetworkID.ValueString()]
		if !ok {
			diags.AddAttributeError(path.Root("native_network_id"), "Unknown network", fmt.Sprintf("No network with ID %q in site %s.", plan.NativeNetworkID.ValueString(), siteID))
			return
		}
		override.NativeNetworkconfID = legacyID
	}
	if known(plan.TaggedVLANMgmt) {
		override.TaggedVLANMgmt = plan.TaggedVLANMgmt.ValueString()
	}
	var excluded []string
	if !plan.ExcludedNetworkIDs.IsNull() && !plan.ExcludedNetworkIDs.IsUnknown() {
		diags.Append(plan.ExcludedNetworkIDs.ElementsAs(ctx, &excluded, false)...)
	}
	// Written only when it means something, so a port whose entry has no
	// exclusion list doesn't get an empty one added.
	if len(excluded) > 0 || override.TaggedVLANMgmt == "custom" {
		override.ExcludedNetworkconfIDs = []string{}
		for _, id := range excluded {
			legacyID, ok := legacyIDs[id]
			if !ok {
				diags.AddAttributeError(path.Root("excluded_network_ids"), "Unknown network", fmt.Sprintf("No network with ID %q in site %s.", id, siteID))
				return
			}
			override.ExcludedNetworkconfIDs = append(override.ExcludedNetworkconfIDs, legacyID)
		}
	}
	stored, err := r.providerData.client.SetDevicePortOverride(ctx, siteID, plan.DeviceMAC.ValueString(), override)
	if err != nil {
		diags.AddError("Unable to update port override", err.Error())
		return
	}
	model, d := flattenDevicePortOverride(siteID, plan.DeviceMAC.ValueString(), stored, legacyIDs)
	diags.Append(d...)
	diags.Append(state.Set(ctx, &model)...)
}

func (r *devicePortOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state devicePortOverrideModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	siteID := state.SiteID.ValueString()
	mac := state.DeviceMAC.ValueString()
	stored, err := r.providerData.client.GetDevicePortOverride(ctx, siteID, mac, state.PortIdx.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read port override", err.Error())
		return
	}
	legacyIDs, err := r.providerData.client.LegacyNetworkIDs(ctx, siteID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read networks", err.Error())
		return
	}
	model, d := flattenDevicePortOverride(siteID, mac, stored, legacyIDs)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves the port as it is: removing the override resets the port to
// "all networks, Default untagged", which silently moves the device plugged
// into it to the default network.
func (r *devicePortOverrideResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *devicePortOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import identifier", "Import with <site_id>/<device_mac>/<port_idx>.")
		return
	}
	portIdx, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || portIdx < 1 {
		resp.Diagnostics.AddError("Invalid import identifier", fmt.Sprintf("port_idx must be a positive number, got %q.", parts[2]))
		return
	}
	mac := strings.ToLower(parts[1])
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), devicePortOverrideID(parts[0], mac, portIdx))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_mac"), mac)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("port_idx"), portIdx)...)
}

func devicePortOverrideID(siteID, mac string, portIdx int64) string {
	return fmt.Sprintf("%s/%s/%d", siteID, strings.ToLower(mac), portIdx)
}

func known(value types.String) bool { return !value.IsNull() && !value.IsUnknown() }

// flattenDevicePortOverride translates the legacy networkconf IDs back to
// integration network IDs. A network without one (a WAN or VPN network) can't
// be named in Terraform, so it reads as an error instead of a silent diff.
func flattenDevicePortOverride(siteID, mac string, override *client.PortOverride, legacyIDs map[string]string) (devicePortOverrideModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	integrationIDs := make(map[string]string, len(legacyIDs))
	for integrationID, legacyID := range legacyIDs {
		integrationIDs[legacyID] = integrationID
	}
	toIntegration := func(attribute, legacyID string) string {
		id, ok := integrationIDs[legacyID]
		if !ok {
			diags.AddAttributeError(path.Root(attribute), "Network without an integration ID", fmt.Sprintf("Port %d of %s refers to legacy network %s, which has no integration network ID.", override.PortIdx, mac, legacyID))
		}
		return id
	}
	model := devicePortOverrideModel{
		ID:             types.StringValue(devicePortOverrideID(siteID, mac, override.PortIdx)),
		SiteID:         types.StringValue(siteID),
		DeviceMAC:      types.StringValue(mac),
		PortIdx:        types.Int64Value(override.PortIdx),
		Name:           types.StringValue(override.Name),
		TaggedVLANMgmt: types.StringValue(override.TaggedVLANMgmt),
	}
	model.NativeNetworkID = types.StringNull()
	if override.NativeNetworkconfID != "" {
		model.NativeNetworkID = types.StringValue(toIntegration("native_network_id", override.NativeNetworkconfID))
	}
	excluded := make([]string, 0, len(override.ExcludedNetworkconfIDs))
	for _, legacyID := range override.ExcludedNetworkconfIDs {
		excluded = append(excluded, toIntegration("excluded_network_ids", legacyID))
	}
	sort.Strings(excluded)
	set, d := types.SetValueFrom(context.Background(), types.StringType, excluded)
	diags.Append(d...)
	model.ExcludedNetworkIDs = set
	return model, diags
}
