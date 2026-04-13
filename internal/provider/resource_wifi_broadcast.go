package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*wifiBroadcastResource)(nil)
	_ resource.ResourceWithConfigure   = (*wifiBroadcastResource)(nil)
	_ resource.ResourceWithImportState = (*wifiBroadcastResource)(nil)
)

type wifiBroadcastResource struct {
	providerData *providerData
}

type wifiBroadcastResourceModel struct {
	ID                                  types.String `tfsdk:"id"`
	SiteID                              types.String `tfsdk:"site_id"`
	Type                                types.String `tfsdk:"type"`
	Name                                types.String `tfsdk:"name"`
	Enabled                             types.Bool   `tfsdk:"enabled"`
	Network                             types.Object `tfsdk:"network"`
	SecurityConfiguration               types.Object `tfsdk:"security_configuration"`
	ClientIsolationEnabled              types.Bool   `tfsdk:"client_isolation_enabled"`
	HideName                            types.Bool   `tfsdk:"hide_name"`
	UAPSDEnabled                        types.Bool   `tfsdk:"uapsd_enabled"`
	MulticastToUnicastConversionEnabled types.Bool   `tfsdk:"multicast_to_unicast_conversion_enabled"`
	BroadcastingFrequenciesGHz          types.Set    `tfsdk:"broadcasting_frequencies_ghz"`
	BroadcastingDeviceFilter            types.Object `tfsdk:"broadcasting_device_filter"`
	AdvertiseDeviceName                 types.Bool   `tfsdk:"advertise_device_name"`
	ARPProxyEnabled                     types.Bool   `tfsdk:"arp_proxy_enabled"`
	BandSteeringEnabled                 types.Bool   `tfsdk:"band_steering_enabled"`
	BSSTransitionEnabled                types.Bool   `tfsdk:"bss_transition_enabled"`
}

type wifiNetworkModel struct {
	Type      types.String `tfsdk:"type"`
	NetworkID types.String `tfsdk:"network_id"`
}

type wifiSAEConfigurationModel struct {
	AnticloggingThresholdSeconds types.Int64 `tfsdk:"anticlogging_threshold_seconds"`
	SyncTimeSeconds              types.Int64 `tfsdk:"sync_time_seconds"`
}

type wifiSecurityConfigurationModel struct {
	Type                      types.String `tfsdk:"type"`
	Passphrase                types.String `tfsdk:"passphrase"`
	PMFMode                   types.String `tfsdk:"pmf_mode"`
	FastRoamingEnabled        types.Bool   `tfsdk:"fast_roaming_enabled"`
	GroupRekeyIntervalSeconds types.Int64  `tfsdk:"group_rekey_interval_seconds"`
	WPA3FastRoamingEnabled    types.Bool   `tfsdk:"wpa3_fast_roaming_enabled"`
	SAEConfiguration          types.Object `tfsdk:"sae_configuration"`
}

type wifiBroadcastingDeviceFilterModel struct {
	Type         types.String `tfsdk:"type"`
	DeviceTagIDs types.Set    `tfsdk:"device_tag_ids"`
}

func wifiNetworkAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":       types.StringType,
		"network_id": types.StringType,
	}
}

func wifiSAEConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"anticlogging_threshold_seconds": types.Int64Type,
		"sync_time_seconds":              types.Int64Type,
	}
}

func wifiSecurityConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":                         types.StringType,
		"passphrase":                   types.StringType,
		"pmf_mode":                     types.StringType,
		"fast_roaming_enabled":         types.BoolType,
		"group_rekey_interval_seconds": types.Int64Type,
		"wpa3_fast_roaming_enabled":    types.BoolType,
		"sae_configuration":            types.ObjectType{AttrTypes: wifiSAEConfigurationAttrTypes()},
	}
}

func wifiBroadcastingDeviceFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":           types.StringType,
		"device_tag_ids": types.SetType{ElemType: types.StringType},
	}
}

func NewWifiBroadcastResource() resource.Resource {
	return &wifiBroadcastResource{}
}

func (r *wifiBroadcastResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_wifi_broadcast"
}

func (r *wifiBroadcastResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi WiFi broadcast.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Broadcast type. Supported values: `STANDARD`, `IOT_OPTIMIZED`.",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"network": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "WiFi network binding. Supported values: `NATIVE`, `SPECIFIC`.",
					},
					"network_id": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"security_configuration": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Security mode. Supported values: `OPEN`, `WPA2_PERSONAL`, `WPA3_PERSONAL`, `WPA2_WPA3_PERSONAL`.",
					},
					"passphrase": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
					},
					"pmf_mode": schema.StringAttribute{
						Optional: true,
					},
					"fast_roaming_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"group_rekey_interval_seconds": schema.Int64Attribute{
						Optional: true,
					},
					"wpa3_fast_roaming_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"sae_configuration": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"anticlogging_threshold_seconds": schema.Int64Attribute{
								Required: true,
							},
							"sync_time_seconds": schema.Int64Attribute{
								Required: true,
							},
						},
					},
				},
			},
			"client_isolation_enabled": schema.BoolAttribute{
				Required: true,
			},
			"hide_name": schema.BoolAttribute{
				Required: true,
			},
			"uapsd_enabled": schema.BoolAttribute{
				Required: true,
			},
			"multicast_to_unicast_conversion_enabled": schema.BoolAttribute{
				Required: true,
			},
			"broadcasting_frequencies_ghz": schema.SetAttribute{
				Optional:    true,
				ElementType: types.Float64Type,
			},
			"broadcasting_device_filter": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Broadcasting device filter type. Current supported value: `DEVICE_TAGS`.",
					},
					"device_tag_ids": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"advertise_device_name": schema.BoolAttribute{
				Optional: true,
			},
			"arp_proxy_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"band_steering_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"bss_transition_enabled": schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (r *wifiBroadcastResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *wifiBroadcastResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan wifiBroadcastResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiBroadcast := r.expandWifiBroadcast(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateWifiBroadcast(ctx, plan.SiteID.ValueString(), apiBroadcast)
	if err != nil {
		response.Diagnostics.AddError("Unable to create WiFi broadcast", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *wifiBroadcastResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state wifiBroadcastResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	broadcast, err := r.providerData.client.GetWifiBroadcast(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read WiFi broadcast", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, broadcast)
}

func (r *wifiBroadcastResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan wifiBroadcastResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state wifiBroadcastResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiBroadcast := r.expandWifiBroadcast(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateWifiBroadcast(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiBroadcast)
	if err != nil {
		response.Diagnostics.AddError("Unable to update WiFi broadcast", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *wifiBroadcastResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state wifiBroadcastResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteWifiBroadcast(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete WiFi broadcast", err.Error())
	}
}

func (r *wifiBroadcastResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func (r *wifiBroadcastResource) expandWifiBroadcast(ctx context.Context, plan wifiBroadcastResourceModel, diags *diag.Diagnostics) client.WifiBroadcast {
	broadcast := client.WifiBroadcast{
		Type:                                plan.Type.ValueString(),
		Name:                                plan.Name.ValueString(),
		Enabled:                             plan.Enabled.ValueBool(),
		ClientIsolationEnabled:              plan.ClientIsolationEnabled.ValueBool(),
		HideName:                            plan.HideName.ValueBool(),
		UAPSDEnabled:                        plan.UAPSDEnabled.ValueBool(),
		MulticastToUnicastConversionEnabled: plan.MulticastToUnicastConversionEnabled.ValueBool(),
		BroadcastingFrequenciesGHz:          setToFloat64s(ctx, plan.BroadcastingFrequenciesGHz, "broadcasting_frequencies_ghz", diags),
		AdvertiseDeviceName:                 boolPointerValue(plan.AdvertiseDeviceName),
		ARPProxyEnabled:                     boolPointerValue(plan.ARPProxyEnabled),
		BandSteeringEnabled:                 boolPointerValue(plan.BandSteeringEnabled),
		BSSTransitionEnabled:                boolPointerValue(plan.BSSTransitionEnabled),
	}

	if err := validateWifiBroadcastModel(ctx, plan); err != nil {
		diags.AddError("Invalid WiFi broadcast configuration", err.Error())
		return broadcast
	}

	var network wifiNetworkModel
	diags.Append(plan.Network.As(ctx, &network, basetypes.ObjectAsOptions{})...)
	broadcast.Network = &client.WifiNetworkReference{
		Type:      network.Type.ValueString(),
		NetworkID: network.NetworkID.ValueString(),
	}

	var security wifiSecurityConfigurationModel
	diags.Append(plan.SecurityConfiguration.As(ctx, &security, basetypes.ObjectAsOptions{})...)
	broadcast.SecurityConfiguration = expandWifiSecurityConfiguration(ctx, security, diags)

	if !plan.BroadcastingDeviceFilter.IsNull() && !plan.BroadcastingDeviceFilter.IsUnknown() {
		var deviceFilter wifiBroadcastingDeviceFilterModel
		diags.Append(plan.BroadcastingDeviceFilter.As(ctx, &deviceFilter, basetypes.ObjectAsOptions{})...)
		broadcast.BroadcastingDeviceFilter = &client.WifiBroadcastingDeviceFilter{
			Type:         deviceFilter.Type.ValueString(),
			DeviceTagIDs: setToStrings(ctx, deviceFilter.DeviceTagIDs, "broadcasting_device_filter.device_tag_ids", diags),
		}
	}

	return broadcast
}

func expandWifiSecurityConfiguration(ctx context.Context, model wifiSecurityConfigurationModel, diags *diag.Diagnostics) *client.WifiSecurityConfiguration {
	configuration := &client.WifiSecurityConfiguration{
		Type:                      model.Type.ValueString(),
		Passphrase:                stringPointerValue(model.Passphrase),
		PMFMode:                   stringPointerValue(model.PMFMode),
		FastRoamingEnabled:        boolPointerValue(model.FastRoamingEnabled),
		GroupRekeyIntervalSeconds: int64PointerValue(model.GroupRekeyIntervalSeconds),
		WPA3FastRoamingEnabled:    boolPointerValue(model.WPA3FastRoamingEnabled),
	}

	if model.SAEConfiguration.IsNull() || model.SAEConfiguration.IsUnknown() {
		return configuration
	}

	var saeModel wifiSAEConfigurationModel
	diags.Append(model.SAEConfiguration.As(ctx, &saeModel, basetypes.ObjectAsOptions{})...)
	configuration.SAEConfiguration = &client.SAEConfiguration{
		AnticloggingThresholdSeconds: saeModel.AnticloggingThresholdSeconds.ValueInt64(),
		SyncTimeSeconds:              saeModel.SyncTimeSeconds.ValueInt64(),
	}

	return configuration
}

func validateWifiBroadcastModel(ctx context.Context, plan wifiBroadcastResourceModel) error {
	broadcastType := plan.Type.ValueString()
	if broadcastType != "STANDARD" && broadcastType != "IOT_OPTIMIZED" {
		return fmt.Errorf("type must be STANDARD or IOT_OPTIMIZED")
	}

	var network wifiNetworkModel
	if diags := plan.Network.As(ctx, &network, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode network block")
	}

	switch network.Type.ValueString() {
	case "NATIVE":
		if !network.NetworkID.IsNull() {
			return fmt.Errorf("network.network_id must not be set when network.type is NATIVE")
		}
	case "SPECIFIC":
		if network.NetworkID.IsNull() {
			return fmt.Errorf("network.network_id is required when network.type is SPECIFIC")
		}
	default:
		return fmt.Errorf("network.type must be NATIVE or SPECIFIC")
	}

	var security wifiSecurityConfigurationModel
	if diags := plan.SecurityConfiguration.As(ctx, &security, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode security_configuration block")
	}

	switch security.Type.ValueString() {
	case "OPEN":
	case "WPA2_PERSONAL":
		if security.Passphrase.IsNull() {
			return fmt.Errorf("security_configuration.passphrase is required for WPA2_PERSONAL")
		}
	case "WPA3_PERSONAL":
		if security.Passphrase.IsNull() || security.SAEConfiguration.IsNull() {
			return fmt.Errorf("security_configuration.passphrase and security_configuration.sae_configuration are required for WPA3_PERSONAL")
		}
	case "WPA2_WPA3_PERSONAL":
		if security.Passphrase.IsNull() || security.PMFMode.IsNull() || security.SAEConfiguration.IsNull() || security.WPA3FastRoamingEnabled.IsNull() {
			return fmt.Errorf("security_configuration.passphrase, pmf_mode, sae_configuration, and wpa3_fast_roaming_enabled are required for WPA2_WPA3_PERSONAL")
		}
	default:
		return fmt.Errorf("security_configuration.type must be OPEN, WPA2_PERSONAL, WPA3_PERSONAL, or WPA2_WPA3_PERSONAL")
	}

	if security.PMFMode.ValueString() != "" && security.PMFMode.ValueString() != "OPTIONAL" && security.PMFMode.ValueString() != "REQUIRED" {
		return fmt.Errorf("security_configuration.pmf_mode must be OPTIONAL or REQUIRED")
	}

	if broadcastType == "STANDARD" {
		if plan.BroadcastingFrequenciesGHz.IsNull() || plan.AdvertiseDeviceName.IsNull() || plan.ARPProxyEnabled.IsNull() || plan.BSSTransitionEnabled.IsNull() {
			return fmt.Errorf("broadcasting_frequencies_ghz, advertise_device_name, arp_proxy_enabled, and bss_transition_enabled are required for STANDARD broadcasts")
		}
	} else {
		if !plan.BroadcastingFrequenciesGHz.IsNull() || !plan.AdvertiseDeviceName.IsNull() || !plan.ARPProxyEnabled.IsNull() || !plan.BSSTransitionEnabled.IsNull() || !plan.BandSteeringEnabled.IsNull() {
			return fmt.Errorf("standard-only attributes must not be set for IOT_OPTIMIZED broadcasts")
		}
	}

	if !plan.BroadcastingDeviceFilter.IsNull() && !plan.BroadcastingDeviceFilter.IsUnknown() {
		var deviceFilter wifiBroadcastingDeviceFilterModel
		if diags := plan.BroadcastingDeviceFilter.As(ctx, &deviceFilter, basetypes.ObjectAsOptions{}); diags.HasError() {
			return fmt.Errorf("unable to decode broadcasting_device_filter block")
		}

		if deviceFilter.Type.IsNull() || deviceFilter.Type.ValueString() == "" {
			return fmt.Errorf("broadcasting_device_filter.type must not be empty")
		}

		if deviceFilter.Type.ValueString() == "DEVICE_TAGS" {
			if len(setToStrings(ctx, deviceFilter.DeviceTagIDs, "broadcasting_device_filter.device_tag_ids", &diag.Diagnostics{})) == 0 {
				return fmt.Errorf("broadcasting_device_filter.device_tag_ids must contain at least one device tag id when type is DEVICE_TAGS")
			}
		} else if !deviceFilter.DeviceTagIDs.IsNull() && !deviceFilter.DeviceTagIDs.IsUnknown() {
			return fmt.Errorf("broadcasting_device_filter.device_tag_ids is only supported when broadcasting_device_filter.type is DEVICE_TAGS")
		}
	}

	return nil
}

func (r *wifiBroadcastResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, broadcast *client.WifiBroadcast) {
	network, diagnostics := flattenWifiNetwork(ctx, broadcast.Network)
	diags.Append(diagnostics...)
	securityConfiguration, diagnostics := flattenWifiSecurityConfiguration(ctx, broadcast.SecurityConfiguration)
	diags.Append(diagnostics...)
	broadcastingFrequencies, diagnostics := float64SetValue(ctx, broadcast.BroadcastingFrequenciesGHz)
	diags.Append(diagnostics...)
	broadcastingDeviceFilter, diagnostics := flattenWifiBroadcastingDeviceFilter(ctx, broadcast.BroadcastingDeviceFilter)
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	model := wifiBroadcastResourceModel{
		ID:                                  types.StringValue(broadcast.ID),
		SiteID:                              siteID,
		Type:                                types.StringValue(broadcast.Type),
		Name:                                types.StringValue(broadcast.Name),
		Enabled:                             types.BoolValue(broadcast.Enabled),
		Network:                             network,
		SecurityConfiguration:               securityConfiguration,
		ClientIsolationEnabled:              types.BoolValue(broadcast.ClientIsolationEnabled),
		HideName:                            types.BoolValue(broadcast.HideName),
		UAPSDEnabled:                        types.BoolValue(broadcast.UAPSDEnabled),
		MulticastToUnicastConversionEnabled: types.BoolValue(broadcast.MulticastToUnicastConversionEnabled),
		BroadcastingFrequenciesGHz:          broadcastingFrequencies,
		BroadcastingDeviceFilter:            broadcastingDeviceFilter,
		AdvertiseDeviceName:                 nullableBool(broadcast.AdvertiseDeviceName),
		ARPProxyEnabled:                     nullableBool(broadcast.ARPProxyEnabled),
		BandSteeringEnabled:                 nullableBool(broadcast.BandSteeringEnabled),
		BSSTransitionEnabled:                nullableBool(broadcast.BSSTransitionEnabled),
	}

	diags.Append(state.Set(ctx, &model)...)
}

func flattenWifiNetwork(ctx context.Context, network *client.WifiNetworkReference) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if network == nil {
		return types.ObjectNull(wifiNetworkAttrTypes()), diagnostics
	}

	model := wifiNetworkModel{
		Type:      types.StringValue(network.Type),
		NetworkID: types.StringNull(),
	}
	if network.NetworkID != "" {
		model.NetworkID = types.StringValue(network.NetworkID)
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, wifiNetworkAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}

func flattenWifiSecurityConfiguration(ctx context.Context, security *client.WifiSecurityConfiguration) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if security == nil {
		return types.ObjectNull(wifiSecurityConfigurationAttrTypes()), diagnostics
	}

	saeConfiguration := types.ObjectNull(wifiSAEConfigurationAttrTypes())
	if security.SAEConfiguration != nil {
		saeModel := wifiSAEConfigurationModel{
			AnticloggingThresholdSeconds: types.Int64Value(security.SAEConfiguration.AnticloggingThresholdSeconds),
			SyncTimeSeconds:              types.Int64Value(security.SAEConfiguration.SyncTimeSeconds),
		}
		value, diagnosticsObject := types.ObjectValueFrom(ctx, wifiSAEConfigurationAttrTypes(), saeModel)
		diagnostics.Append(diagnosticsObject...)
		saeConfiguration = value
	}

	model := wifiSecurityConfigurationModel{
		Type:                      types.StringValue(security.Type),
		Passphrase:                nullableString(security.Passphrase),
		PMFMode:                   nullableString(security.PMFMode),
		FastRoamingEnabled:        nullableBool(security.FastRoamingEnabled),
		GroupRekeyIntervalSeconds: nullableInt64(security.GroupRekeyIntervalSeconds),
		WPA3FastRoamingEnabled:    nullableBool(security.WPA3FastRoamingEnabled),
		SAEConfiguration:          saeConfiguration,
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, wifiSecurityConfigurationAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}

func flattenWifiBroadcastingDeviceFilter(ctx context.Context, filter *client.WifiBroadcastingDeviceFilter) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if filter == nil {
		return types.ObjectNull(wifiBroadcastingDeviceFilterAttrTypes()), diagnostics
	}

	deviceTagIDs, diagnosticsSet := stringSetValue(ctx, filter.DeviceTagIDs)
	diagnostics.Append(diagnosticsSet...)

	model := wifiBroadcastingDeviceFilterModel{
		Type:         types.StringValue(filter.Type),
		DeviceTagIDs: deviceTagIDs,
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, wifiBroadcastingDeviceFilterAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}
