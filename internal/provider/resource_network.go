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
	_ resource.Resource                = (*networkResource)(nil)
	_ resource.ResourceWithConfigure   = (*networkResource)(nil)
	_ resource.ResourceWithImportState = (*networkResource)(nil)
)

type networkResource struct {
	providerData *providerData
}

type networkResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	SiteID                types.String `tfsdk:"site_id"`
	Management            types.String `tfsdk:"management"`
	Name                  types.String `tfsdk:"name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	VLANID                types.Int64  `tfsdk:"vlan_id"`
	IsolationEnabled      types.Bool   `tfsdk:"isolation_enabled"`
	CellularBackupEnabled types.Bool   `tfsdk:"cellular_backup_enabled"`
	ZoneID                types.String `tfsdk:"zone_id"`
	DeviceID              types.String `tfsdk:"device_id"`
	InternetAccessEnabled types.Bool   `tfsdk:"internet_access_enabled"`
	MDNSForwardingEnabled types.Bool   `tfsdk:"mdns_forwarding_enabled"`
	DHCPGuarding          types.Object `tfsdk:"dhcp_guarding"`
	IPv4Configuration     types.Object `tfsdk:"ipv4_configuration"`
	Default               types.Bool   `tfsdk:"default"`
}

type networkDHCPGuardingModel struct {
	TrustedDHCPServerIPAddresses types.Set `tfsdk:"trusted_dhcp_server_ip_addresses"`
}

type networkIPv4DHCPConfigurationModel struct {
	Mode                         types.String `tfsdk:"mode"`
	StartIPAddress               types.String `tfsdk:"start_ip_address"`
	EndIPAddress                 types.String `tfsdk:"end_ip_address"`
	DHCPServerIPAddresses        types.Set    `tfsdk:"dhcp_server_ip_addresses"`
	LeaseTimeSeconds             types.Int64  `tfsdk:"lease_time_seconds"`
	DNSServerIPAddresses         types.Set    `tfsdk:"dns_server_ip_addresses"`
	PingConflictDetectionEnabled types.Bool   `tfsdk:"ping_conflict_detection_enabled"`
	GatewayIPAddressOverride     types.String `tfsdk:"gateway_ip_address_override"`
	DomainName                   types.String `tfsdk:"domain_name"`
}

type networkIPv4ConfigurationModel struct {
	AutoScaleEnabled        types.Bool   `tfsdk:"auto_scale_enabled"`
	HostIPAddress           types.String `tfsdk:"host_ip_address"`
	PrefixLength            types.Int64  `tfsdk:"prefix_length"`
	AdditionalHostIPSubnets types.Set    `tfsdk:"additional_host_ip_subnets"`
	DHCPConfiguration       types.Object `tfsdk:"dhcp_configuration"`
}

func networkDHCPGuardingAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"trusted_dhcp_server_ip_addresses": types.SetType{ElemType: types.StringType},
	}
}

func networkIPv4DHCPConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":                            types.StringType,
		"start_ip_address":                types.StringType,
		"end_ip_address":                  types.StringType,
		"dhcp_server_ip_addresses":        types.SetType{ElemType: types.StringType},
		"lease_time_seconds":              types.Int64Type,
		"dns_server_ip_addresses":         types.SetType{ElemType: types.StringType},
		"ping_conflict_detection_enabled": types.BoolType,
		"gateway_ip_address_override":     types.StringType,
		"domain_name":                     types.StringType,
	}
}

func networkIPv4ConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"auto_scale_enabled":         types.BoolType,
		"host_ip_address":            types.StringType,
		"prefix_length":              types.Int64Type,
		"additional_host_ip_subnets": types.SetType{ElemType: types.StringType},
		"dhcp_configuration":         types.ObjectType{AttrTypes: networkIPv4DHCPConfigurationAttrTypes()},
	}
}

func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

func (r *networkResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_network"
}

func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi network using the current integration API network model.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"management": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Network management mode. Supported values: `UNMANAGED`, `GATEWAY`, `SWITCH`.",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"vlan_id": schema.Int64Attribute{
				Required: true,
			},
			"isolation_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"cellular_backup_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"zone_id": schema.StringAttribute{
				Optional: true,
			},
			"device_id": schema.StringAttribute{
				Optional: true,
			},
			"internet_access_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"mdns_forwarding_enabled": schema.BoolAttribute{
				Optional: true,
			},
			"dhcp_guarding": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"trusted_dhcp_server_ip_addresses": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
				},
			},
			"ipv4_configuration": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"auto_scale_enabled": schema.BoolAttribute{
						Required: true,
					},
					"host_ip_address": schema.StringAttribute{
						Required: true,
					},
					"prefix_length": schema.Int64Attribute{
						Required: true,
					},
					"additional_host_ip_subnets": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"dhcp_configuration": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"mode": schema.StringAttribute{
								Required: true,
							},
							"start_ip_address": schema.StringAttribute{
								Optional: true,
							},
							"end_ip_address": schema.StringAttribute{
								Optional: true,
							},
							"dhcp_server_ip_addresses": schema.SetAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"lease_time_seconds": schema.Int64Attribute{
								Optional: true,
							},
							"dns_server_ip_addresses": schema.SetAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"ping_conflict_detection_enabled": schema.BoolAttribute{
								Optional: true,
							},
							"gateway_ip_address_override": schema.StringAttribute{
								Optional: true,
							},
							"domain_name": schema.StringAttribute{
								Optional: true,
							},
						},
					},
				},
			},
			"default": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (r *networkResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *networkResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan networkResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiNetwork := r.expandNetwork(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateNetwork(ctx, plan.SiteID.ValueString(), apiNetwork)
	if err != nil {
		response.Diagnostics.AddError("Unable to create network", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *networkResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state networkResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	network, err := r.providerData.client.GetNetwork(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read network", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, network)
}

func (r *networkResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan networkResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiNetwork := r.expandNetwork(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateNetwork(ctx, plan.SiteID.ValueString(), plan.ID.ValueString(), apiNetwork)
	if err != nil {
		response.Diagnostics.AddError("Unable to update network", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *networkResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state networkResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteNetwork(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete network", err.Error())
	}
}

func (r *networkResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func (r *networkResource) expandNetwork(ctx context.Context, plan networkResourceModel, diags *diag.Diagnostics) client.Network {
	network := client.Network{
		Management: plan.Management.ValueString(),
		Name:       plan.Name.ValueString(),
		Enabled:    plan.Enabled.ValueBool(),
		VLANID:     plan.VLANID.ValueInt64(),
	}

	if err := validateNetworkModel(ctx, plan); err != nil {
		diags.AddError("Invalid network configuration", err.Error())
		return network
	}

	if !plan.DHCPGuarding.IsNull() && !plan.DHCPGuarding.IsUnknown() {
		var guarding networkDHCPGuardingModel
		diags.Append(plan.DHCPGuarding.As(ctx, &guarding, basetypes.ObjectAsOptions{})...)
		network.DHCPGuarding = &client.DHCPGuarding{
			TrustedDHCPServerIPAddresses: setToStrings(ctx, guarding.TrustedDHCPServerIPAddresses, "dhcp_guarding.trusted_dhcp_server_ip_addresses", diags),
		}
	}

	switch plan.Management.ValueString() {
	case "UNMANAGED":
		return network
	case "GATEWAY", "SWITCH":
		network.IsolationEnabled = boolPointerValue(plan.IsolationEnabled)
		network.CellularBackupEnabled = boolPointerValue(plan.CellularBackupEnabled)

		if !plan.IPv4Configuration.IsNull() && !plan.IPv4Configuration.IsUnknown() {
			var ipv4Configuration networkIPv4ConfigurationModel
			diags.Append(plan.IPv4Configuration.As(ctx, &ipv4Configuration, basetypes.ObjectAsOptions{})...)
			network.IPv4Configuration = expandNetworkIPv4Configuration(ctx, ipv4Configuration, diags)
		}

		if plan.Management.ValueString() == "GATEWAY" {
			network.InternetAccessEnabled = boolPointerValue(plan.InternetAccessEnabled)
			network.MDNSForwardingEnabled = boolPointerValue(plan.MDNSForwardingEnabled)
			network.ZoneID = stringPointerValue(plan.ZoneID)
		}

		if plan.Management.ValueString() == "SWITCH" {
			network.DeviceID = stringPointerValue(plan.DeviceID)
		}
	}

	return network
}

func expandNetworkIPv4Configuration(ctx context.Context, model networkIPv4ConfigurationModel, diags *diag.Diagnostics) *client.IPv4Configuration {
	configuration := &client.IPv4Configuration{
		AutoScaleEnabled:        model.AutoScaleEnabled.ValueBool(),
		HostIPAddress:           model.HostIPAddress.ValueString(),
		PrefixLength:            model.PrefixLength.ValueInt64(),
		AdditionalHostIPSubnets: setToStrings(ctx, model.AdditionalHostIPSubnets, "ipv4_configuration.additional_host_ip_subnets", diags),
	}

	if model.DHCPConfiguration.IsNull() || model.DHCPConfiguration.IsUnknown() {
		return configuration
	}

	var dhcpConfiguration networkIPv4DHCPConfigurationModel
	diags.Append(model.DHCPConfiguration.As(ctx, &dhcpConfiguration, basetypes.ObjectAsOptions{})...)

	configuration.DHCPConfiguration = &client.IPv4DHCPConfiguration{
		Mode:                         dhcpConfiguration.Mode.ValueString(),
		DHCPServerIPAddresses:        setToStrings(ctx, dhcpConfiguration.DHCPServerIPAddresses, "ipv4_configuration.dhcp_configuration.dhcp_server_ip_addresses", diags),
		DNSServerIPAddressesOverride: setToStrings(ctx, dhcpConfiguration.DNSServerIPAddresses, "ipv4_configuration.dhcp_configuration.dns_server_ip_addresses", diags),
		LeaseTimeSeconds:             int64PointerValue(dhcpConfiguration.LeaseTimeSeconds),
		PingConflictDetectionEnabled: boolPointerValue(dhcpConfiguration.PingConflictDetectionEnabled),
		GatewayIPAddressOverride:     stringPointerValue(dhcpConfiguration.GatewayIPAddressOverride),
		DomainName:                   stringPointerValue(dhcpConfiguration.DomainName),
	}

	if !dhcpConfiguration.StartIPAddress.IsNull() && !dhcpConfiguration.EndIPAddress.IsNull() {
		configuration.DHCPConfiguration.IPAddressRange = &client.IPAddressRange{
			Start: dhcpConfiguration.StartIPAddress.ValueString(),
			Stop:  dhcpConfiguration.EndIPAddress.ValueString(),
		}
	}

	return configuration
}

func validateNetworkModel(ctx context.Context, plan networkResourceModel) error {
	management := plan.Management.ValueString()
	if management != "UNMANAGED" && management != "GATEWAY" && management != "SWITCH" {
		return fmt.Errorf("management must be one of UNMANAGED, GATEWAY, or SWITCH")
	}

	switch management {
	case "UNMANAGED":
		if !plan.IPv4Configuration.IsNull() {
			return fmt.Errorf("ipv4_configuration is not supported for UNMANAGED networks")
		}
		if !plan.IsolationEnabled.IsNull() || !plan.CellularBackupEnabled.IsNull() || !plan.ZoneID.IsNull() || !plan.DeviceID.IsNull() || !plan.InternetAccessEnabled.IsNull() || !plan.MDNSForwardingEnabled.IsNull() {
			return fmt.Errorf("management-specific attributes must not be set for UNMANAGED networks")
		}
	case "GATEWAY":
		if plan.IPv4Configuration.IsNull() {
			return fmt.Errorf("ipv4_configuration is required for GATEWAY networks")
		}
		if plan.IsolationEnabled.IsNull() || plan.CellularBackupEnabled.IsNull() || plan.InternetAccessEnabled.IsNull() || plan.MDNSForwardingEnabled.IsNull() {
			return fmt.Errorf("isolation_enabled, cellular_backup_enabled, internet_access_enabled, and mdns_forwarding_enabled are required for GATEWAY networks")
		}
		if !plan.DeviceID.IsNull() {
			return fmt.Errorf("device_id is only valid for SWITCH networks")
		}
	case "SWITCH":
		if plan.IPv4Configuration.IsNull() {
			return fmt.Errorf("ipv4_configuration is required for SWITCH networks")
		}
		if plan.IsolationEnabled.IsNull() || plan.CellularBackupEnabled.IsNull() || plan.DeviceID.IsNull() {
			return fmt.Errorf("isolation_enabled, cellular_backup_enabled, and device_id are required for SWITCH networks")
		}
		if !plan.ZoneID.IsNull() || !plan.InternetAccessEnabled.IsNull() || !plan.MDNSForwardingEnabled.IsNull() {
			return fmt.Errorf("zone_id, internet_access_enabled, and mdns_forwarding_enabled are only valid for GATEWAY networks")
		}
	}

	if plan.IPv4Configuration.IsNull() || plan.IPv4Configuration.IsUnknown() {
		return nil
	}

	var ipv4Configuration networkIPv4ConfigurationModel
	if diags := plan.IPv4Configuration.As(ctx, &ipv4Configuration, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode ipv4_configuration")
	}

	if ipv4Configuration.DHCPConfiguration.IsNull() || ipv4Configuration.DHCPConfiguration.IsUnknown() {
		return nil
	}

	var dhcpConfiguration networkIPv4DHCPConfigurationModel
	if diags := ipv4Configuration.DHCPConfiguration.As(ctx, &dhcpConfiguration, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode ipv4_configuration.dhcp_configuration")
	}

	switch dhcpConfiguration.Mode.ValueString() {
	case "SERVER":
		if dhcpConfiguration.StartIPAddress.IsNull() || dhcpConfiguration.EndIPAddress.IsNull() || dhcpConfiguration.LeaseTimeSeconds.IsNull() {
			return fmt.Errorf("start_ip_address, end_ip_address, and lease_time_seconds are required when dhcp_configuration.mode is SERVER")
		}
	case "RELAY":
		if dhcpConfiguration.DHCPServerIPAddresses.IsNull() {
			return fmt.Errorf("dhcp_server_ip_addresses is required when dhcp_configuration.mode is RELAY")
		}
	default:
		return fmt.Errorf("dhcp_configuration.mode must be SERVER or RELAY")
	}

	return nil
}

func (r *networkResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, network *client.Network) {
	dhcpGuarding := types.ObjectNull(networkDHCPGuardingAttrTypes())
	if network.DHCPGuarding != nil {
		guardingModel := networkDHCPGuardingModel{}
		guardingModel.TrustedDHCPServerIPAddresses, _ = stringSetValue(ctx, network.DHCPGuarding.TrustedDHCPServerIPAddresses)
		value, diagnostics := types.ObjectValueFrom(ctx, networkDHCPGuardingAttrTypes(), guardingModel)
		diags.Append(diagnostics...)
		dhcpGuarding = value
	}

	ipv4Configuration := types.ObjectNull(networkIPv4ConfigurationAttrTypes())
	if network.IPv4Configuration != nil {
		value, diagnostics := flattenNetworkIPv4Configuration(ctx, network.IPv4Configuration)
		diags.Append(diagnostics...)
		ipv4Configuration = value
	}

	model := networkResourceModel{
		ID:                    types.StringValue(network.ID),
		SiteID:                siteID,
		Management:            types.StringValue(network.Management),
		Name:                  types.StringValue(network.Name),
		Enabled:               types.BoolValue(network.Enabled),
		VLANID:                types.Int64Value(network.VLANID),
		IsolationEnabled:      nullableBool(network.IsolationEnabled),
		CellularBackupEnabled: nullableBool(network.CellularBackupEnabled),
		ZoneID:                nullableString(network.ZoneID),
		DeviceID:              nullableString(network.DeviceID),
		InternetAccessEnabled: nullableBool(network.InternetAccessEnabled),
		MDNSForwardingEnabled: nullableBool(network.MDNSForwardingEnabled),
		DHCPGuarding:          dhcpGuarding,
		IPv4Configuration:     ipv4Configuration,
		Default:               types.BoolValue(network.Default),
	}

	diags.Append(state.Set(ctx, &model)...)
}

func flattenNetworkIPv4Configuration(ctx context.Context, ipv4Configuration *client.IPv4Configuration) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	dhcpConfiguration := types.ObjectNull(networkIPv4DHCPConfigurationAttrTypes())
	if ipv4Configuration.DHCPConfiguration != nil {
		dnsServerIPAddresses, diagnosticsSet := stringSetValue(ctx, ipv4Configuration.DHCPConfiguration.DNSServerIPAddressesOverride)
		diagnostics.Append(diagnosticsSet...)
		dhcpServerIPAddresses, diagnosticsSet := stringSetValue(ctx, ipv4Configuration.DHCPConfiguration.DHCPServerIPAddresses)
		diagnostics.Append(diagnosticsSet...)

		dhcpModel := networkIPv4DHCPConfigurationModel{
			Mode:                         types.StringValue(ipv4Configuration.DHCPConfiguration.Mode),
			StartIPAddress:               types.StringNull(),
			EndIPAddress:                 types.StringNull(),
			DHCPServerIPAddresses:        dhcpServerIPAddresses,
			LeaseTimeSeconds:             nullableInt64(ipv4Configuration.DHCPConfiguration.LeaseTimeSeconds),
			DNSServerIPAddresses:         dnsServerIPAddresses,
			PingConflictDetectionEnabled: nullableBool(ipv4Configuration.DHCPConfiguration.PingConflictDetectionEnabled),
			GatewayIPAddressOverride:     nullableString(ipv4Configuration.DHCPConfiguration.GatewayIPAddressOverride),
			DomainName:                   nullableString(ipv4Configuration.DHCPConfiguration.DomainName),
		}

		if ipv4Configuration.DHCPConfiguration.IPAddressRange != nil {
			dhcpModel.StartIPAddress = types.StringValue(ipv4Configuration.DHCPConfiguration.IPAddressRange.Start)
			dhcpModel.EndIPAddress = types.StringValue(ipv4Configuration.DHCPConfiguration.IPAddressRange.Stop)
		}

		value, diagnosticsObject := types.ObjectValueFrom(ctx, networkIPv4DHCPConfigurationAttrTypes(), dhcpModel)
		diagnostics.Append(diagnosticsObject...)
		dhcpConfiguration = value
	}

	additionalHostSubnets, diagnosticsSet := stringSetValue(ctx, ipv4Configuration.AdditionalHostIPSubnets)
	diagnostics.Append(diagnosticsSet...)

	ipv4Model := networkIPv4ConfigurationModel{
		AutoScaleEnabled:        types.BoolValue(ipv4Configuration.AutoScaleEnabled),
		HostIPAddress:           types.StringValue(ipv4Configuration.HostIPAddress),
		PrefixLength:            types.Int64Value(ipv4Configuration.PrefixLength),
		AdditionalHostIPSubnets: additionalHostSubnets,
		DHCPConfiguration:       dhcpConfiguration,
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, networkIPv4ConfigurationAttrTypes(), ipv4Model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}
