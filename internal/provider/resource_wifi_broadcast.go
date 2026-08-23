package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	Channel2GLockedTo6                  types.Bool   `tfsdk:"channel_2g_locked_to_6"`
	DTIMPeriod2GLockedTo3               types.Bool   `tfsdk:"dtim_period_2g_locked_to_3"`
	DNSAssistanceConfiguration          types.Object `tfsdk:"dns_assistance_configuration"`
}

type wifiNetworkModel struct {
	Type      types.String `tfsdk:"type"`
	NetworkID types.String `tfsdk:"network_id"`
}

type wifiSAEConfigurationModel struct {
	AnticloggingThresholdSeconds types.Int64 `tfsdk:"anticlogging_threshold_seconds"`
	SyncTimeSeconds              types.Int64 `tfsdk:"sync_time_seconds"`
}

type wifiNASIDConfigurationModel struct {
	Type   types.String `tfsdk:"type"`
	Source types.String `tfsdk:"source"`
	Value  types.String `tfsdk:"value"`
}

type wifiRadiusMACAuthenticationConfigurationModel struct {
	MACAddressFormat types.String `tfsdk:"mac_address_format"`
}

type wifiRadiusConfigurationModel struct {
	ProfileID                      types.String `tfsdk:"profile_id"`
	NASID                          types.Object `tfsdk:"nas_id"`
	MACAuthenticationConfiguration types.Object `tfsdk:"mac_authentication_configuration"`
}

type wifiPresharedKeyModel struct {
	Passphrase types.String `tfsdk:"passphrase"`
	Network    types.Object `tfsdk:"network"`
}

type wifiSecurityConfigurationModel struct {
	Type                      types.String `tfsdk:"type"`
	Passphrase                types.String `tfsdk:"passphrase"`
	Encryption                types.String `tfsdk:"encryption"`
	PMFMode                   types.String `tfsdk:"pmf_mode"`
	FastRoamingEnabled        types.Bool   `tfsdk:"fast_roaming_enabled"`
	GroupRekeyIntervalSeconds types.Int64  `tfsdk:"group_rekey_interval_seconds"`
	WPA3FastRoamingEnabled    types.Bool   `tfsdk:"wpa3_fast_roaming_enabled"`
	SAEConfiguration          types.Object `tfsdk:"sae_configuration"`
	RadiusConfiguration       types.Object `tfsdk:"radius_configuration"`
	CoAEnabled                types.Bool   `tfsdk:"coa_enabled"`
	SecurityMode              types.String `tfsdk:"security_mode"`
	PresharedKeys             types.List   `tfsdk:"preshared_keys"`
}

type wifiDNSAssistanceConfigurationModel struct {
	Mode    types.String `tfsdk:"mode"`
	Servers types.List   `tfsdk:"servers"`
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

func wifiNASIDConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":   types.StringType,
		"source": types.StringType,
		"value":  types.StringType,
	}
}

func wifiRadiusMACAuthenticationConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{"mac_address_format": types.StringType}
}

func wifiRadiusConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"profile_id":                       types.StringType,
		"nas_id":                           types.ObjectType{AttrTypes: wifiNASIDConfigurationAttrTypes()},
		"mac_authentication_configuration": types.ObjectType{AttrTypes: wifiRadiusMACAuthenticationConfigurationAttrTypes()},
	}
}

func wifiPresharedKeyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"passphrase": types.StringType,
		"network":    types.ObjectType{AttrTypes: wifiNetworkAttrTypes()},
	}
}

func wifiSecurityConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":                         types.StringType,
		"passphrase":                   types.StringType,
		"encryption":                   types.StringType,
		"pmf_mode":                     types.StringType,
		"fast_roaming_enabled":         types.BoolType,
		"group_rekey_interval_seconds": types.Int64Type,
		"wpa3_fast_roaming_enabled":    types.BoolType,
		"sae_configuration":            types.ObjectType{AttrTypes: wifiSAEConfigurationAttrTypes()},
		"radius_configuration":         types.ObjectType{AttrTypes: wifiRadiusConfigurationAttrTypes()},
		"coa_enabled":                  types.BoolType,
		"security_mode":                types.StringType,
		"preshared_keys":               types.ListType{ElemType: types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}},
	}
}

func wifiDNSAssistanceConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":    types.StringType,
		"servers": types.ListType{ElemType: types.StringType},
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
		MarkdownDescription: "Manage a UniFi WiFi broadcast. Enterprise security references an existing RADIUS profile; `unifi_radius_profile` is a data source because the UniFi Integration API exposes RADIUS profiles as supporting/read-only resources. WPA2 PPSK and Enterprise authentication are separate security modes and must not be assumed to coexist on one broadcast. UniFi responses may omit PPSK passphrases; known state secrets are preserved during refresh, while imported PPSK broadcasts require the passphrases to be supplied in configuration.",
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
						MarkdownDescription: "Security mode. Supported values: `OPEN`, `WPA2_PERSONAL`, `WPA3_PERSONAL`, `WPA2_WPA3_PERSONAL`, `WPA2_ENTERPRISE`, `WPA2_WPA3_ENTERPRISE`, `WPA3_ENTERPRISE`.",
					},
					"passphrase": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
					},
					"encryption": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Open security encryption mode. Supported values for `OPEN` security: `ENHANCED_OPEN`, `ENHANCED_OPEN_WITH_TRANSITION`. Leave unset for plain open WiFi.",
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
					"radius_configuration": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"profile_id": schema.StringAttribute{Required: true},
							"nas_id": schema.SingleNestedAttribute{
								Required: true,
								Attributes: map[string]schema.Attribute{
									"type":   schema.StringAttribute{Required: true},
									"source": schema.StringAttribute{Optional: true},
									"value":  schema.StringAttribute{Optional: true},
								},
							},
							"mac_authentication_configuration": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"mac_address_format": schema.StringAttribute{Required: true},
								},
							},
						},
					},
					"coa_enabled": schema.BoolAttribute{Optional: true},
					"security_mode": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "WPA3 Enterprise security mode. Supported values: `DEFAULT`, `HIGH_SECURITY_192_BIT`.",
					},
					"preshared_keys": schema.ListNestedAttribute{
						Optional: true,
						NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
							"passphrase": schema.StringAttribute{Required: true, Sensitive: true},
							"network": schema.SingleNestedAttribute{
								Required: true,
								Attributes: map[string]schema.Attribute{
									"type":       schema.StringAttribute{Required: true},
									"network_id": schema.StringAttribute{Optional: true},
								},
							},
						}},
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
			"channel_2g_locked_to_6": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Locks the 2.4 GHz radio channel to 6 on all broadcasting devices. Requires UniFi Network `10.6` or newer; older controllers do not report this field.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dtim_period_2g_locked_to_3": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Locks the DTIM period to 3 for the 2.4 GHz radio. Requires UniFi Network `10.6` or newer; older controllers do not report this field.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_assistance_configuration": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "DNS assistance configuration for `STANDARD` WiFi broadcasts. Supported modes: `AUTO`, `MANUAL`.",
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Required: true,
					},
					"servers": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
				},
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
	preserveWifiSecuritySecrets(ctx, created.SecurityConfiguration, plan.SecurityConfiguration, &response.Diagnostics)

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
	preserveWifiSecuritySecrets(ctx, broadcast.SecurityConfiguration, state.SecurityConfiguration, &response.Diagnostics)

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
	preserveWifiSecuritySecrets(ctx, updated.SecurityConfiguration, plan.SecurityConfiguration, &response.Diagnostics)

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
		Channel2GLockedTo6:                  boolPointerValue(plan.Channel2GLockedTo6),
		DTIMPeriod2GLockedTo3:               boolPointerValue(plan.DTIMPeriod2GLockedTo3),
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

	if !plan.DNSAssistanceConfiguration.IsNull() && !plan.DNSAssistanceConfiguration.IsUnknown() {
		var dnsAssistance wifiDNSAssistanceConfigurationModel
		diags.Append(plan.DNSAssistanceConfiguration.As(ctx, &dnsAssistance, basetypes.ObjectAsOptions{})...)
		configuration := &client.WifiDNSAssistanceConfiguration{
			Mode: dnsAssistance.Mode.ValueString(),
		}
		if !dnsAssistance.Servers.IsNull() && !dnsAssistance.Servers.IsUnknown() {
			servers := listToStrings(ctx, dnsAssistance.Servers, "dns_assistance_configuration.servers", diags)
			configuration.Servers = &servers
		}
		broadcast.DNSAssistanceConfiguration = configuration
	}

	return broadcast
}

func expandWifiSecurityConfiguration(ctx context.Context, model wifiSecurityConfigurationModel, diags *diag.Diagnostics) *client.WifiSecurityConfiguration {
	configuration := &client.WifiSecurityConfiguration{
		Type: model.Type.ValueString(),
	}

	switch configuration.Type {
	case "OPEN":
		configuration.Encryption = stringPointerValue(model.Encryption)
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
	case "WPA2_PERSONAL":
		configuration.Passphrase = stringPointerValue(model.Passphrase)
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
		configuration.PMFMode = stringPointerValue(model.PMFMode)
		configuration.FastRoamingEnabled = boolPointerValue(model.FastRoamingEnabled)
		configuration.GroupRekeyIntervalSeconds = int64PointerValue(model.GroupRekeyIntervalSeconds)
		configuration.PresharedKeys = expandWifiPresharedKeys(ctx, model.PresharedKeys, diags)
	case "WPA3_PERSONAL":
		configuration.Passphrase = stringPointerValue(model.Passphrase)
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
		configuration.PMFMode = stringPointerValue(model.PMFMode)
		configuration.FastRoamingEnabled = boolPointerValue(model.FastRoamingEnabled)
		configuration.GroupRekeyIntervalSeconds = int64PointerValue(model.GroupRekeyIntervalSeconds)
	case "WPA2_WPA3_PERSONAL":
		configuration.Passphrase = stringPointerValue(model.Passphrase)
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
		configuration.PMFMode = stringPointerValue(model.PMFMode)
		configuration.FastRoamingEnabled = boolPointerValue(model.FastRoamingEnabled)
		configuration.GroupRekeyIntervalSeconds = int64PointerValue(model.GroupRekeyIntervalSeconds)
		configuration.WPA3FastRoamingEnabled = boolPointerValue(model.WPA3FastRoamingEnabled)
	case "WPA2_ENTERPRISE":
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
		configuration.CoAEnabled = boolPointerValue(model.CoAEnabled)
		configuration.PMFMode = stringPointerValue(model.PMFMode)
		configuration.FastRoamingEnabled = boolPointerValue(model.FastRoamingEnabled)
		configuration.GroupRekeyIntervalSeconds = int64PointerValue(model.GroupRekeyIntervalSeconds)
	case "WPA2_WPA3_ENTERPRISE":
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
		configuration.CoAEnabled = boolPointerValue(model.CoAEnabled)
		configuration.PMFMode = stringPointerValue(model.PMFMode)
		configuration.FastRoamingEnabled = boolPointerValue(model.FastRoamingEnabled)
		configuration.GroupRekeyIntervalSeconds = int64PointerValue(model.GroupRekeyIntervalSeconds)
		configuration.WPA3FastRoamingEnabled = boolPointerValue(model.WPA3FastRoamingEnabled)
	case "WPA3_ENTERPRISE":
		configuration.RadiusConfiguration = expandWifiRadiusConfiguration(ctx, model.RadiusConfiguration, diags)
		configuration.CoAEnabled = boolPointerValue(model.CoAEnabled)
		configuration.SecurityMode = stringPointerValue(model.SecurityMode)
		configuration.FastRoamingEnabled = boolPointerValue(model.FastRoamingEnabled)
		configuration.GroupRekeyIntervalSeconds = int64PointerValue(model.GroupRekeyIntervalSeconds)
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

func expandWifiRadiusConfiguration(ctx context.Context, value types.Object, diags *diag.Diagnostics) *client.WifiRadiusConfiguration {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var model wifiRadiusConfigurationModel
	diags.Append(value.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	var nasID wifiNASIDConfigurationModel
	diags.Append(model.NASID.As(ctx, &nasID, basetypes.ObjectAsOptions{})...)
	configuration := &client.WifiRadiusConfiguration{
		ProfileID: model.ProfileID.ValueString(),
		NASID: client.WifiNASIDConfiguration{
			Type:   nasID.Type.ValueString(),
			Source: stringPointerValue(nasID.Source),
			Value:  stringPointerValue(nasID.Value),
		},
	}
	if !model.MACAuthenticationConfiguration.IsNull() && !model.MACAuthenticationConfiguration.IsUnknown() {
		var macAuth wifiRadiusMACAuthenticationConfigurationModel
		diags.Append(model.MACAuthenticationConfiguration.As(ctx, &macAuth, basetypes.ObjectAsOptions{})...)
		configuration.MACAuthenticationConfiguration = &client.WifiRadiusMACAuthenticationConfiguration{MACAddressFormat: macAuth.MACAddressFormat.ValueString()}
	}
	return configuration
}

func expandWifiPresharedKeys(ctx context.Context, value types.List, diags *diag.Diagnostics) []client.WifiPresharedKey {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var models []wifiPresharedKeyModel
	diags.Append(value.ElementsAs(ctx, &models, false)...)
	keys := make([]client.WifiPresharedKey, 0, len(models))
	for _, model := range models {
		var network wifiNetworkModel
		diags.Append(model.Network.As(ctx, &network, basetypes.ObjectAsOptions{})...)
		keys = append(keys, client.WifiPresharedKey{
			Passphrase: stringPointerValue(model.Passphrase),
			Network:    client.WifiNetworkReference{Type: network.Type.ValueString(), NetworkID: network.NetworkID.ValueString()},
		})
	}
	return keys
}

func preserveWifiSecuritySecrets(ctx context.Context, security *client.WifiSecurityConfiguration, priorValue types.Object, diags *diag.Diagnostics) {
	if security == nil || priorValue.IsNull() || priorValue.IsUnknown() {
		return
	}
	var prior wifiSecurityConfigurationModel
	diags.Append(priorValue.As(ctx, &prior, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return
	}
	if security.Passphrase == nil && !prior.Passphrase.IsNull() && !prior.Passphrase.IsUnknown() {
		security.Passphrase = stringPointerValue(prior.Passphrase)
	}
	if len(security.PresharedKeys) == 0 || prior.PresharedKeys.IsNull() || prior.PresharedKeys.IsUnknown() {
		return
	}
	var priorKeys []wifiPresharedKeyModel
	diags.Append(prior.PresharedKeys.ElementsAs(ctx, &priorKeys, false)...)
	usedPriorKeys := make([]bool, len(priorKeys))
	for index := range security.PresharedKeys {
		if security.PresharedKeys[index].Passphrase != nil {
			continue
		}
		for priorIndex, priorKey := range priorKeys {
			if usedPriorKeys[priorIndex] {
				continue
			}
			var network wifiNetworkModel
			diags.Append(priorKey.Network.As(ctx, &network, basetypes.ObjectAsOptions{})...)
			if network.Type.ValueString() == security.PresharedKeys[index].Network.Type && network.NetworkID.ValueString() == security.PresharedKeys[index].Network.NetworkID {
				security.PresharedKeys[index].Passphrase = stringPointerValue(priorKey.Passphrase)
				usedPriorKeys[priorIndex] = true
				break
			}
		}
	}
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
		if err := rejectSecurityFields(security, "OPEN", "passphrase", "pmf_mode", "fast_roaming_enabled", "group_rekey_interval_seconds", "wpa3_fast_roaming_enabled", "sae_configuration", "coa_enabled", "security_mode", "preshared_keys"); err != nil {
			return err
		}
	case "WPA2_PERSONAL":
		if security.Passphrase.IsNull() && security.PresharedKeys.IsNull() && security.RadiusConfiguration.IsNull() {
			return fmt.Errorf("security_configuration.passphrase, preshared_keys, or radius_configuration is required for WPA2_PERSONAL")
		}
		if err := rejectSecurityFields(security, "WPA2_PERSONAL", "encryption", "wpa3_fast_roaming_enabled", "sae_configuration", "coa_enabled", "security_mode"); err != nil {
			return err
		}
		if err := validateWifiPresharedKeys(ctx, security.PresharedKeys); err != nil {
			return err
		}
	case "WPA3_PERSONAL":
		if security.Passphrase.IsNull() || security.SAEConfiguration.IsNull() {
			return fmt.Errorf("security_configuration.passphrase and security_configuration.sae_configuration are required for WPA3_PERSONAL")
		}
		if err := rejectSecurityFields(security, "WPA3_PERSONAL", "encryption", "wpa3_fast_roaming_enabled", "coa_enabled", "security_mode", "preshared_keys"); err != nil {
			return err
		}
	case "WPA2_WPA3_PERSONAL":
		if security.Passphrase.IsNull() || security.PMFMode.IsNull() || security.SAEConfiguration.IsNull() || security.WPA3FastRoamingEnabled.IsNull() {
			return fmt.Errorf("security_configuration.passphrase, pmf_mode, sae_configuration, and wpa3_fast_roaming_enabled are required for WPA2_WPA3_PERSONAL")
		}
		if err := rejectSecurityFields(security, "WPA2_WPA3_PERSONAL", "encryption", "coa_enabled", "security_mode", "preshared_keys"); err != nil {
			return err
		}
	case "WPA2_ENTERPRISE":
		if security.RadiusConfiguration.IsNull() || security.CoAEnabled.IsNull() {
			return fmt.Errorf("security_configuration.radius_configuration and security_configuration.coa_enabled are required for WPA2_ENTERPRISE")
		}
		if err := rejectSecurityFields(security, "WPA2_ENTERPRISE", "passphrase", "encryption", "wpa3_fast_roaming_enabled", "sae_configuration", "security_mode", "preshared_keys"); err != nil {
			return err
		}
	case "WPA2_WPA3_ENTERPRISE":
		if security.RadiusConfiguration.IsNull() || security.CoAEnabled.IsNull() || security.PMFMode.IsNull() || security.WPA3FastRoamingEnabled.IsNull() {
			return fmt.Errorf("security_configuration.radius_configuration, coa_enabled, pmf_mode, and wpa3_fast_roaming_enabled are required for WPA2_WPA3_ENTERPRISE")
		}
		if err := rejectSecurityFields(security, "WPA2_WPA3_ENTERPRISE", "passphrase", "encryption", "sae_configuration", "security_mode", "preshared_keys"); err != nil {
			return err
		}
	case "WPA3_ENTERPRISE":
		if security.RadiusConfiguration.IsNull() || security.CoAEnabled.IsNull() || security.SecurityMode.IsNull() {
			return fmt.Errorf("security_configuration.radius_configuration, coa_enabled, and security_mode are required for WPA3_ENTERPRISE")
		}
		if err := rejectSecurityFields(security, "WPA3_ENTERPRISE", "passphrase", "encryption", "pmf_mode", "wpa3_fast_roaming_enabled", "sae_configuration", "preshared_keys"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("security_configuration.type must be OPEN, WPA2_PERSONAL, WPA3_PERSONAL, WPA2_WPA3_PERSONAL, WPA2_ENTERPRISE, WPA2_WPA3_ENTERPRISE, or WPA3_ENTERPRISE")
	}

	if !security.RadiusConfiguration.IsNull() {
		if err := validateWifiRadiusConfiguration(ctx, security.RadiusConfiguration); err != nil {
			return err
		}
		if isNonEnterpriseSecurityType(security.Type.ValueString()) {
			var radius wifiRadiusConfigurationModel
			if diags := security.RadiusConfiguration.As(ctx, &radius, basetypes.ObjectAsOptions{}); diags.HasError() || radius.MACAuthenticationConfiguration.IsNull() {
				return fmt.Errorf("security_configuration.radius_configuration.mac_authentication_configuration is required for non-Enterprise security")
			}
		}
	}
	if err := validateWifiSecurityMode(security.SecurityMode); err != nil {
		return err
	}

	if !security.Encryption.IsNull() {
		if security.Type.ValueString() != "OPEN" {
			return fmt.Errorf("security_configuration.encryption is only valid when security_configuration.type is OPEN")
		}
		switch security.Encryption.ValueString() {
		case "ENHANCED_OPEN", "ENHANCED_OPEN_WITH_TRANSITION":
		default:
			return fmt.Errorf("security_configuration.encryption must be ENHANCED_OPEN or ENHANCED_OPEN_WITH_TRANSITION")
		}
	}

	if security.PMFMode.ValueString() != "" && security.PMFMode.ValueString() != "OPTIONAL" && security.PMFMode.ValueString() != "REQUIRED" {
		return fmt.Errorf("security_configuration.pmf_mode must be OPTIONAL or REQUIRED")
	}

	if broadcastType == "STANDARD" {
		if plan.BroadcastingFrequenciesGHz.IsNull() || plan.AdvertiseDeviceName.IsNull() || plan.ARPProxyEnabled.IsNull() || plan.BSSTransitionEnabled.IsNull() {
			return fmt.Errorf("broadcasting_frequencies_ghz, advertise_device_name, arp_proxy_enabled, and bss_transition_enabled are required for STANDARD broadcasts")
		}
	} else {
		if !plan.BroadcastingFrequenciesGHz.IsNull() || !plan.AdvertiseDeviceName.IsNull() || !plan.ARPProxyEnabled.IsNull() || !plan.BSSTransitionEnabled.IsNull() || !plan.BandSteeringEnabled.IsNull() || !plan.DNSAssistanceConfiguration.IsNull() {
			return fmt.Errorf("standard-only attributes must not be set for IOT_OPTIMIZED broadcasts")
		}
	}

	if !plan.DNSAssistanceConfiguration.IsNull() && !plan.DNSAssistanceConfiguration.IsUnknown() {
		var dnsAssistance wifiDNSAssistanceConfigurationModel
		if diags := plan.DNSAssistanceConfiguration.As(ctx, &dnsAssistance, basetypes.ObjectAsOptions{}); diags.HasError() {
			return fmt.Errorf("unable to decode dns_assistance_configuration block")
		}

		switch dnsAssistance.Mode.ValueString() {
		case "AUTO":
			if !dnsAssistance.Servers.IsNull() {
				return fmt.Errorf("dns_assistance_configuration.servers must not be set when mode is AUTO")
			}
		case "MANUAL":
			if dnsAssistance.Servers.IsNull() {
				return fmt.Errorf("dns_assistance_configuration.servers is required when mode is MANUAL")
			}
			if len(listToStrings(ctx, dnsAssistance.Servers, "dns_assistance_configuration.servers", &diag.Diagnostics{})) > 2 {
				return fmt.Errorf("dns_assistance_configuration.servers supports at most two DNS servers")
			}
		default:
			return fmt.Errorf("dns_assistance_configuration.mode must be AUTO or MANUAL")
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

func validateWifiSecurityMode(value types.String) error {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "DEFAULT" || value.ValueString() == "HIGH_SECURITY_192_BIT" {
		return nil
	}
	return fmt.Errorf("security_configuration.security_mode must be DEFAULT or HIGH_SECURITY_192_BIT")
}

func isNonEnterpriseSecurityType(value string) bool {
	switch value {
	case "OPEN", "WPA2_PERSONAL", "WPA3_PERSONAL", "WPA2_WPA3_PERSONAL":
		return true
	default:
		return false
	}
}

func rejectSecurityFields(security wifiSecurityConfigurationModel, securityType string, fields ...string) error {
	set := map[string]bool{
		"passphrase": !security.Passphrase.IsNull(), "encryption": !security.Encryption.IsNull(), "pmf_mode": !security.PMFMode.IsNull(),
		"fast_roaming_enabled": !security.FastRoamingEnabled.IsNull(), "group_rekey_interval_seconds": !security.GroupRekeyIntervalSeconds.IsNull(),
		"wpa3_fast_roaming_enabled": !security.WPA3FastRoamingEnabled.IsNull(), "sae_configuration": !security.SAEConfiguration.IsNull(),
		"radius_configuration": !security.RadiusConfiguration.IsNull(), "coa_enabled": !security.CoAEnabled.IsNull(), "security_mode": !security.SecurityMode.IsNull(),
		"preshared_keys": !security.PresharedKeys.IsNull(),
	}
	for _, field := range fields {
		if set[field] {
			return fmt.Errorf("security_configuration.%s is not valid for %s", field, securityType)
		}
	}
	return nil
}

func validateWifiRadiusConfiguration(ctx context.Context, value types.Object) error {
	var radius wifiRadiusConfigurationModel
	if diags := value.As(ctx, &radius, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode security_configuration.radius_configuration")
	}
	var nasID wifiNASIDConfigurationModel
	if diags := radius.NASID.As(ctx, &nasID, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode security_configuration.radius_configuration.nas_id")
	}
	switch nasID.Type.ValueString() {
	case "DERIVED":
		if nasID.Source.IsNull() || !nasID.Value.IsNull() {
			return fmt.Errorf("derived NAS-ID requires source and forbids value")
		}
		switch nasID.Source.ValueString() {
		case "DEVICE_MAC_ADDRESS", "DEVICE_NAME", "SITE_NAME", "BSSID":
		default:
			return fmt.Errorf("NAS-ID source must be DEVICE_MAC_ADDRESS, DEVICE_NAME, SITE_NAME, or BSSID")
		}
	case "USER_DEFINED":
		if nasID.Value.IsNull() || !nasID.Source.IsNull() {
			return fmt.Errorf("user-defined NAS-ID requires value and forbids source")
		}
	default:
		return fmt.Errorf("NAS-ID type must be DERIVED or USER_DEFINED")
	}
	if !radius.MACAuthenticationConfiguration.IsNull() {
		var macAuth wifiRadiusMACAuthenticationConfigurationModel
		if diags := radius.MACAuthenticationConfiguration.As(ctx, &macAuth, basetypes.ObjectAsOptions{}); diags.HasError() {
			return fmt.Errorf("unable to decode MAC authentication configuration")
		}
		switch macAuth.MACAddressFormat.ValueString() {
		case "UPPERCASE_NOT_SEPARATED", "UPPERCASE_DASH_SEPARATED", "UPPERCASE_COLON_SEPARATED", "LOWERCASE_NOT_SEPARATED", "LOWERCASE_COLON_SEPARATED", "LOWERCASE_DASH_SEPARATED":
		default:
			return fmt.Errorf("unsupported MAC authentication address format")
		}
	}
	return nil
}

func validateWifiPresharedKeys(ctx context.Context, value types.List) error {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	if len(value.Elements()) == 0 {
		return fmt.Errorf("security_configuration.preshared_keys must contain at least one key")
	}
	var keys []wifiPresharedKeyModel
	if diags := value.ElementsAs(ctx, &keys, false); diags.HasError() {
		return fmt.Errorf("unable to decode security_configuration.preshared_keys")
	}
	for i, key := range keys {
		var network wifiNetworkModel
		if diags := key.Network.As(ctx, &network, basetypes.ObjectAsOptions{}); diags.HasError() {
			return fmt.Errorf("unable to decode preshared key %d network", i)
		}
		switch network.Type.ValueString() {
		case "NATIVE":
			if !network.NetworkID.IsNull() {
				return fmt.Errorf("preshared key %d network_id must not be set for NATIVE", i)
			}
		case "SPECIFIC":
			if network.NetworkID.IsNull() {
				return fmt.Errorf("preshared key %d network_id is required for SPECIFIC", i)
			}
		default:
			return fmt.Errorf("preshared key %d network type must be NATIVE or SPECIFIC", i)
		}
	}
	return nil
}

func (r *wifiBroadcastResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, broadcast *client.WifiBroadcast) {
	model, diagnostics := buildWifiBroadcastStateModel(ctx, siteID, broadcast)
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	diags.Append(state.Set(ctx, &model)...)
}

func buildWifiBroadcastStateModel(ctx context.Context, siteID types.String, broadcast *client.WifiBroadcast) (wifiBroadcastResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	network, networkDiagnostics := flattenWifiNetwork(ctx, broadcast.Network)
	diagnostics.Append(networkDiagnostics...)
	securityConfiguration, securityDiagnostics := flattenWifiSecurityConfiguration(ctx, broadcast.SecurityConfiguration)
	diagnostics.Append(securityDiagnostics...)
	broadcastingFrequencies, frequenciesDiagnostics := float64SetValue(ctx, broadcast.BroadcastingFrequenciesGHz)
	diagnostics.Append(frequenciesDiagnostics...)
	broadcastingDeviceFilter, deviceFilterDiagnostics := flattenWifiBroadcastingDeviceFilter(ctx, broadcast.BroadcastingDeviceFilter)
	diagnostics.Append(deviceFilterDiagnostics...)
	dnsAssistanceConfiguration, dnsAssistanceDiagnostics := flattenWifiDNSAssistanceConfiguration(ctx, broadcast.DNSAssistanceConfiguration)
	diagnostics.Append(dnsAssistanceDiagnostics...)

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
		Channel2GLockedTo6:                  nullableBool(broadcast.Channel2GLockedTo6),
		DTIMPeriod2GLockedTo3:               nullableBool(broadcast.DTIMPeriod2GLockedTo3),
		DNSAssistanceConfiguration:          dnsAssistanceConfiguration,
	}

	return model, diagnostics
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
	radiusConfiguration, radiusDiagnostics := flattenWifiRadiusConfiguration(ctx, security.RadiusConfiguration)
	diagnostics.Append(radiusDiagnostics...)
	presharedKeys, pskDiagnostics := flattenWifiPresharedKeys(ctx, security.PresharedKeys)
	diagnostics.Append(pskDiagnostics...)

	model := wifiSecurityConfigurationModel{
		Type:                      types.StringValue(security.Type),
		Passphrase:                nullableString(security.Passphrase),
		Encryption:                nullableString(security.Encryption),
		PMFMode:                   nullableString(security.PMFMode),
		FastRoamingEnabled:        nullableBool(security.FastRoamingEnabled),
		GroupRekeyIntervalSeconds: nullableInt64(security.GroupRekeyIntervalSeconds),
		WPA3FastRoamingEnabled:    nullableBool(security.WPA3FastRoamingEnabled),
		SAEConfiguration:          saeConfiguration,
		RadiusConfiguration:       radiusConfiguration,
		CoAEnabled:                nullableBool(security.CoAEnabled),
		SecurityMode:              nullableString(security.SecurityMode),
		PresharedKeys:             presharedKeys,
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, wifiSecurityConfigurationAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}

func flattenWifiRadiusConfiguration(ctx context.Context, configuration *client.WifiRadiusConfiguration) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if configuration == nil {
		return types.ObjectNull(wifiRadiusConfigurationAttrTypes()), diagnostics
	}
	nasID, nasDiagnostics := types.ObjectValueFrom(ctx, wifiNASIDConfigurationAttrTypes(), wifiNASIDConfigurationModel{
		Type: types.StringValue(configuration.NASID.Type), Source: nullableString(configuration.NASID.Source), Value: nullableString(configuration.NASID.Value),
	})
	diagnostics.Append(nasDiagnostics...)
	macAuth := types.ObjectNull(wifiRadiusMACAuthenticationConfigurationAttrTypes())
	if configuration.MACAuthenticationConfiguration != nil {
		var macDiagnostics diag.Diagnostics
		macAuth, macDiagnostics = types.ObjectValueFrom(ctx, wifiRadiusMACAuthenticationConfigurationAttrTypes(), wifiRadiusMACAuthenticationConfigurationModel{
			MACAddressFormat: types.StringValue(configuration.MACAuthenticationConfiguration.MACAddressFormat),
		})
		diagnostics.Append(macDiagnostics...)
	}
	value, objectDiagnostics := types.ObjectValueFrom(ctx, wifiRadiusConfigurationAttrTypes(), wifiRadiusConfigurationModel{
		ProfileID: types.StringValue(configuration.ProfileID), NASID: nasID, MACAuthenticationConfiguration: macAuth,
	})
	diagnostics.Append(objectDiagnostics...)
	return value, diagnostics
}

func flattenWifiPresharedKeys(ctx context.Context, keys []client.WifiPresharedKey) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if keys == nil {
		return types.ListNull(types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}), diagnostics
	}
	models := make([]wifiPresharedKeyModel, 0, len(keys))
	for _, key := range keys {
		network, networkDiagnostics := flattenWifiNetwork(ctx, &key.Network)
		diagnostics.Append(networkDiagnostics...)
		models = append(models, wifiPresharedKeyModel{Passphrase: nullableString(key.Passphrase), Network: network})
	}
	value, listDiagnostics := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: wifiPresharedKeyAttrTypes()}, models)
	diagnostics.Append(listDiagnostics...)
	return value, diagnostics
}

func flattenWifiDNSAssistanceConfiguration(ctx context.Context, configuration *client.WifiDNSAssistanceConfiguration) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if configuration == nil {
		return types.ObjectNull(wifiDNSAssistanceConfigurationAttrTypes()), diagnostics
	}

	servers := types.ListNull(types.StringType)
	if configuration.Servers != nil {
		serverList, diagnosticsList := stringListValue(ctx, *configuration.Servers)
		diagnostics.Append(diagnosticsList...)
		servers = serverList
	}

	model := wifiDNSAssistanceConfigurationModel{
		Mode:    types.StringValue(configuration.Mode),
		Servers: servers,
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, wifiDNSAssistanceConfigurationAttrTypes(), model)
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
