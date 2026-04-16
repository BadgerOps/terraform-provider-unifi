package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*firewallPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*firewallPolicyResource)(nil)
)

type firewallPolicyResource struct {
	providerData *providerData
}

func NewFirewallPolicyResource() resource.Resource {
	return &firewallPolicyResource{}
}

func (r *firewallPolicyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_policy"
}

func (r *firewallPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi firewall policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"site_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Site UUID that owns the firewall policy.",
			},
			"enabled":     schema.BoolAttribute{Required: true},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"action": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Policy action. Supported values: `ALLOW`, `BLOCK`, `REJECT`.",
			},
			"allow_return_traffic": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Required when `action` is `ALLOW`. Creates the derived reverse policy to allow return traffic. Set this explicitly to `true` or `false` because current UniFi controller builds may reject `ALLOW` rules when the field is omitted.",
			},
			"source_zone_id":      schema.StringAttribute{Required: true},
			"source_filter":       firewallPolicyEndpointFilterSchema("source"),
			"destination_zone_id": schema.StringAttribute{Required: true},
			"destination_filter":  firewallPolicyEndpointFilterSchema("destination"),
			"ip_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "IP version scope. Supported values: `IPV4`, `IPV6`, `IPV4_AND_IPV6`.",
			},
			"protocol_filter": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optional protocol filter nested under `ip_version`.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Supported values: `NAMED_PROTOCOL`, `PROTOCOL_NUMBER`, `PRESET`.",
					},
					"named_protocol": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Named protocol when `type` is `NAMED_PROTOCOL`. Current UniFi controller builds reliably accept `ICMP` here. For TCP/UDP service rules, prefer `type = PRESET` with `preset_name = \"TCP_UDP\"` plus a nested destination `port_filter`.",
					},
					"match_opposite": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "When true, match all protocols except the selected one. Used by `NAMED_PROTOCOL` and `PROTOCOL_NUMBER`.",
					},
					"protocol_number": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "IANA protocol number when `type` is `PROTOCOL_NUMBER`.",
					},
					"preset_name": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Preset name when `type` is `PRESET`. Current controller preset is `TCP_UDP`.",
					},
				},
			},
			"connection_state_filter": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional connection state filter values: `NEW`, `INVALID`, `ESTABLISHED`, `RELATED`.",
			},
			"ipsec_filter": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional IPsec traffic filter. Supported values: `MATCH_ENCRYPTED`, `MATCH_NOT_ENCRYPTED`.",
			},
			"logging_enabled": schema.BoolAttribute{Required: true},
			"schedule": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optional schedule configuration.",
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Supported values: `EVERY_DAY`, `EVERY_WEEK`, `ONE_TIME_ONLY`, `CUSTOM`.",
					},
					"start_time": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Start time in `HH:MM` format.",
					},
					"stop_time": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Stop time in `HH:MM` format.",
					},
					"repeat_on_days": schema.SetAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Days used by `EVERY_WEEK` and `CUSTOM` modes.",
					},
					"date": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Date in `YYYY-MM-DD` format for `ONE_TIME_ONLY` mode.",
					},
					"start_date": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Start date in `YYYY-MM-DD` format for `CUSTOM` mode.",
					},
					"stop_date": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Stop date in `YYYY-MM-DD` format for `CUSTOM` mode.",
					},
				},
			},
			"index": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Controller-assigned firewall policy ordering index.",
			},
		},
	}
}

func firewallPolicyEndpointFilterSchema(side string) schema.Attribute {
	return schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: fmt.Sprintf("Optional %s traffic filter.", side),
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: firewallPolicyEndpointTypeDescription(side),
			},
			"network_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"network_match_opposite": schema.BoolAttribute{
				Optional: true,
			},
			"mac_address": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Additional source MAC address selector. Only valid for source `NETWORK`, `IP_ADDRESS`, and `IPV6_IID` filters.",
			},
			"mac_addresses": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "MAC address set for `MAC_ADDRESS` source filters.",
			},
			"ip_addresses": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IP addresses, CIDR subnets, or address ranges for `IP_ADDRESS` filters.",
			},
			"ip_address_match_opposite": schema.BoolAttribute{
				Optional: true,
			},
			"ip_address_traffic_matching_list_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Traffic matching list UUID for `IP_ADDRESS` filters.",
			},
			"ipv6_iid": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "IPv6 IID filter value for `IPV6_IID` filters.",
			},
			"ipv6_iid_match_opposite": schema.BoolAttribute{
				Optional: true,
			},
			"region_codes": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "ISO 3166-1 alpha-2 country codes for `REGION` filters.",
			},
			"vpn_server_ids": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "VPN server UUIDs for `VPN_SERVER` filters.",
			},
			"vpn_server_match_opposite": schema.BoolAttribute{
				Optional: true,
			},
			"site_to_site_vpn_tunnel_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Site-to-site VPN tunnel UUID for `SITE_TO_SITE_VPN_TUNNEL` filters.",
			},
			"domains": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Domain list for destination `DOMAIN` filters.",
			},
			"application_ids": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "DPI application IDs for destination `APPLICATION` filters.",
			},
			"application_category_ids": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "DPI category IDs for destination `APPLICATION_CATEGORY` filters.",
			},
			"port_filter": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optional nested port filter. `PORT` endpoint filters require this block.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Supported values: `PORTS`, `TRAFFIC_MATCHING_LIST`.",
					},
					"match_opposite": schema.BoolAttribute{
						Optional: true,
					},
					"ports": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"traffic_matching_list_id": schema.StringAttribute{
						Optional: true,
					},
				},
			},
		},
	}
}

func firewallPolicyEndpointFilterComputedSchema(side string) schema.Attribute {
	return schema.SingleNestedAttribute{
		Computed:            true,
		MarkdownDescription: fmt.Sprintf("Resolved %s traffic filter.", side),
		Attributes: map[string]schema.Attribute{
			"type":                                schema.StringAttribute{Computed: true},
			"network_ids":                         schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"network_match_opposite":              schema.BoolAttribute{Computed: true},
			"mac_address":                         schema.StringAttribute{Computed: true},
			"mac_addresses":                       schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"ip_addresses":                        schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"ip_address_match_opposite":           schema.BoolAttribute{Computed: true},
			"ip_address_traffic_matching_list_id": schema.StringAttribute{Computed: true},
			"ipv6_iid":                            schema.StringAttribute{Computed: true},
			"ipv6_iid_match_opposite":             schema.BoolAttribute{Computed: true},
			"region_codes":                        schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"vpn_server_ids":                      schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"vpn_server_match_opposite":           schema.BoolAttribute{Computed: true},
			"site_to_site_vpn_tunnel_id":          schema.StringAttribute{Computed: true},
			"domains":                             schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"application_ids":                     schema.SetAttribute{Computed: true, ElementType: types.Int64Type},
			"application_category_ids":            schema.SetAttribute{Computed: true, ElementType: types.Int64Type},
			"port_filter": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type":                     schema.StringAttribute{Computed: true},
					"match_opposite":           schema.BoolAttribute{Computed: true},
					"ports":                    schema.SetAttribute{Computed: true, ElementType: types.StringType},
					"traffic_matching_list_id": schema.StringAttribute{Computed: true},
				},
			},
		},
	}
}

func firewallPolicyEndpointTypeDescription(side string) string {
	if side == "source" {
		return "Supported values: `PORT`, `NETWORK`, `MAC_ADDRESS`, `IP_ADDRESS`, `IPV6_IID`, `REGION`, `VPN_SERVER`, `SITE_TO_SITE_VPN_TUNNEL`."
	}
	return "Supported values: `PORT`, `NETWORK`, `IP_ADDRESS`, `IPV6_IID`, `REGION`, `VPN_SERVER`, `SITE_TO_SITE_VPN_TUNNEL`, `DOMAIN`, `APPLICATION`, `APPLICATION_CATEGORY`."
}

func (r *firewallPolicyResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *firewallPolicyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan firewallPolicyModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := expandFirewallPolicy(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateFirewallPolicy(ctx, plan.SiteID.ValueString(), apiPolicy)
	if err != nil {
		response.Diagnostics.AddError("Unable to create firewall policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *firewallPolicyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state firewallPolicyModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallPolicy, err := r.providerData.client.GetFirewallPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read firewall policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, firewallPolicy)
}

func (r *firewallPolicyResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan firewallPolicyModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state firewallPolicyModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := expandFirewallPolicy(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateFirewallPolicy(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiPolicy)
	if err != nil {
		response.Diagnostics.AddError("Unable to update firewall policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *firewallPolicyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state firewallPolicyModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteFirewallPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete firewall policy", err.Error())
	}
}

func (r *firewallPolicyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func expandFirewallPolicy(ctx context.Context, plan firewallPolicyModel, diags *diag.Diagnostics) client.FirewallPolicy {
	policy := client.FirewallPolicy{
		Enabled:               plan.Enabled.ValueBool(),
		Name:                  plan.Name.ValueString(),
		Description:           stringPointerValue(plan.Description),
		LoggingEnabled:        plan.LoggingEnabled.ValueBool(),
		ConnectionStateFilter: nil,
		Action: &client.FirewallPolicyAction{
			Type:               plan.Action.ValueString(),
			AllowReturnTraffic: boolPointerValue(plan.AllowReturnTraffic),
		},
		Source: &client.FirewallPolicyEndpoint{
			ZoneID: plan.SourceZoneID.ValueString(),
		},
		Destination: &client.FirewallPolicyEndpoint{
			ZoneID: plan.DestinationZoneID.ValueString(),
		},
		IPProtocolScope: &client.FirewallPolicyIPProtocolScope{
			IPVersion: plan.IPVersion.ValueString(),
		},
		IPsecFilter: stringPointerValue(plan.IPsecFilter),
	}

	if err := validateFirewallPolicyBase(plan); err != nil {
		diags.AddError("Invalid firewall policy configuration", err.Error())
		return policy
	}

	if !plan.ConnectionStateFilter.IsNull() {
		policy.ConnectionStateFilter = setToStrings(ctx, plan.ConnectionStateFilter, "connection_state_filter", diags)
	}

	policy.Source.TrafficFilter = expandFirewallPolicyEndpointFilter(ctx, "source", plan.SourceFilter, diags)
	policy.Destination.TrafficFilter = expandFirewallPolicyEndpointFilter(ctx, "destination", plan.DestinationFilter, diags)
	policy.IPProtocolScope.ProtocolFilter = expandFirewallPolicyProtocolFilter(ctx, plan.ProtocolFilter, diags)
	policy.Schedule = expandFirewallPolicySchedule(ctx, plan.Schedule, diags)

	return policy
}

func validateFirewallPolicyBase(plan firewallPolicyModel) error {
	switch plan.Action.ValueString() {
	case "ALLOW", "BLOCK", "REJECT":
	default:
		return fmt.Errorf("action must be one of ALLOW, BLOCK, or REJECT")
	}

	if plan.Action.ValueString() != "ALLOW" && !plan.AllowReturnTraffic.IsNull() && !plan.AllowReturnTraffic.IsUnknown() {
		return fmt.Errorf("allow_return_traffic is only valid when action is ALLOW")
	}
	if plan.Action.ValueString() == "ALLOW" && (plan.AllowReturnTraffic.IsNull() || plan.AllowReturnTraffic.IsUnknown()) {
		return fmt.Errorf("allow_return_traffic must be set explicitly when action is ALLOW")
	}

	switch plan.IPVersion.ValueString() {
	case "IPV4", "IPV6", "IPV4_AND_IPV6":
	default:
		return fmt.Errorf("ip_version must be one of IPV4, IPV6, or IPV4_AND_IPV6")
	}

	if !plan.IPsecFilter.IsNull() {
		switch plan.IPsecFilter.ValueString() {
		case "MATCH_ENCRYPTED", "MATCH_NOT_ENCRYPTED":
		default:
			return fmt.Errorf("ipsec_filter must be one of MATCH_ENCRYPTED or MATCH_NOT_ENCRYPTED")
		}
	}

	return nil
}

func expandFirewallPolicyEndpointFilter(ctx context.Context, side string, value types.Object, diags *diag.Diagnostics) *client.FirewallPolicyTrafficFilter {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	model, err := decodeFirewallPolicyEndpointFilter(ctx, value, side+"_filter")
	if err != nil {
		diags.AddError("Invalid firewall policy filter", err.Error())
		return nil
	}

	filter := &client.FirewallPolicyTrafficFilter{Type: model.Type.ValueString()}
	portFilter := expandFirewallPolicyPortFilter(ctx, model.PortFilter, side+"_filter.port_filter", diags)
	if diags.HasError() {
		return nil
	}

	switch model.Type.ValueString() {
	case "PORT":
		if portFilter == nil {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.port_filter is required when %s_filter.type is PORT", side, side))
			return nil
		}
		filter.PortFilter = portFilter
	case "NETWORK":
		networkIDs := setToStrings(ctx, model.NetworkIDs, side+"_filter.network_ids", diags)
		if len(networkIDs) == 0 {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.network_ids must contain at least one value when %s_filter.type is NETWORK", side, side))
			return nil
		}
		filter.NetworkFilter = &client.FirewallPolicyNetworkFilter{
			NetworkIDs:    networkIDs,
			MatchOpposite: boolValueOrFalse(model.NetworkMatchOpposite),
		}
		filter.PortFilter = portFilter
		if side == "source" {
			filter.MACAddress = stringPointerValue(model.MACAddress)
		}
	case "MAC_ADDRESS":
		if side != "source" {
			diags.AddError("Invalid firewall policy filter", "destination_filter.type cannot be MAC_ADDRESS")
			return nil
		}
		macAddresses := setToStrings(ctx, model.MACAddresses, side+"_filter.mac_addresses", diags)
		if len(macAddresses) == 0 {
			diags.AddError("Invalid firewall policy filter", "source_filter.mac_addresses must contain at least one value when source_filter.type is MAC_ADDRESS")
			return nil
		}
		filter.MACAddressFilter = &client.FirewallPolicyMACAddressListFilter{MacAddresses: macAddresses}
		filter.PortFilter = portFilter
	case "IP_ADDRESS":
		ipAddressFilter := expandFirewallPolicyIPAddressFilter(ctx, model, side, diags)
		if diags.HasError() {
			return nil
		}
		filter.IPAddressFilter = ipAddressFilter
		filter.PortFilter = portFilter
		if side == "source" {
			filter.MACAddress = stringPointerValue(model.MACAddress)
		}
	case "IPV6_IID":
		if model.IPv6IID.IsNull() || model.IPv6IID.ValueString() == "" {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.ipv6_iid is required when %s_filter.type is IPV6_IID", side, side))
			return nil
		}
		filter.IPv6IIDFilter = &client.FirewallPolicyIPv6IIDFilter{
			IPv6IID:       model.IPv6IID.ValueString(),
			MatchOpposite: boolValueOrFalse(model.IPv6IIDMatchOpposite),
		}
		filter.PortFilter = portFilter
		if side == "source" {
			filter.MACAddress = stringPointerValue(model.MACAddress)
		}
	case "REGION":
		regionCodes := setToStrings(ctx, model.RegionCodes, side+"_filter.region_codes", diags)
		if len(regionCodes) == 0 {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.region_codes must contain at least one value when %s_filter.type is REGION", side, side))
			return nil
		}
		filter.RegionFilter = &client.FirewallPolicyRegionFilter{Regions: regionCodes}
		filter.PortFilter = portFilter
	case "VPN_SERVER":
		vpnServerIDs := setToStrings(ctx, model.VPNServerIDs, side+"_filter.vpn_server_ids", diags)
		if len(vpnServerIDs) == 0 {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.vpn_server_ids must contain at least one value when %s_filter.type is VPN_SERVER", side, side))
			return nil
		}
		filter.VPNServerFilter = &client.FirewallPolicyVPNServerFilter{
			VPNServerIDs:  vpnServerIDs,
			MatchOpposite: boolValueOrFalse(model.VPNServerMatchOpposite),
		}
		filter.PortFilter = portFilter
	case "SITE_TO_SITE_VPN_TUNNEL":
		if model.SiteToSiteVPNTunnelID.IsNull() || model.SiteToSiteVPNTunnelID.ValueString() == "" {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.site_to_site_vpn_tunnel_id is required when %s_filter.type is SITE_TO_SITE_VPN_TUNNEL", side, side))
			return nil
		}
		filter.SiteToSiteVPNTunnelFilter = &client.FirewallPolicySiteToSiteVPNTunnelFilter{
			SiteToSiteVPNTunnelID: model.SiteToSiteVPNTunnelID.ValueString(),
		}
		filter.PortFilter = portFilter
	case "DOMAIN":
		if side != "destination" {
			diags.AddError("Invalid firewall policy filter", "source_filter.type cannot be DOMAIN")
			return nil
		}
		domains := setToStrings(ctx, model.Domains, side+"_filter.domains", diags)
		if len(domains) == 0 {
			diags.AddError("Invalid firewall policy filter", "destination_filter.domains must contain at least one value when destination_filter.type is DOMAIN")
			return nil
		}
		filter.DomainFilter = &client.FirewallPolicyDomainFilter{
			Type:    "DOMAINS",
			Domains: domains,
		}
		filter.PortFilter = portFilter
	case "APPLICATION":
		if side != "destination" {
			diags.AddError("Invalid firewall policy filter", "source_filter.type cannot be APPLICATION")
			return nil
		}
		applicationIDs := setToInt64s(ctx, model.ApplicationIDs, side+"_filter.application_ids", diags)
		if len(applicationIDs) == 0 {
			diags.AddError("Invalid firewall policy filter", "destination_filter.application_ids must contain at least one value when destination_filter.type is APPLICATION")
			return nil
		}
		filter.ApplicationFilter = &client.FirewallPolicyApplicationFilter{ApplicationIDs: applicationIDs}
		filter.PortFilter = portFilter
	case "APPLICATION_CATEGORY":
		if side != "destination" {
			diags.AddError("Invalid firewall policy filter", "source_filter.type cannot be APPLICATION_CATEGORY")
			return nil
		}
		categoryIDs := setToInt64s(ctx, model.ApplicationCategoryIDs, side+"_filter.application_category_ids", diags)
		if len(categoryIDs) == 0 {
			diags.AddError("Invalid firewall policy filter", "destination_filter.application_category_ids must contain at least one value when destination_filter.type is APPLICATION_CATEGORY")
			return nil
		}
		filter.ApplicationCategoryFilter = &client.FirewallPolicyApplicationCategoryFilter{ApplicationCategoryIDs: categoryIDs}
		filter.PortFilter = portFilter
	default:
		diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.type %q is not supported", side, model.Type.ValueString()))
		return nil
	}

	return filter
}

func expandFirewallPolicyIPAddressFilter(ctx context.Context, model firewallPolicyEndpointFilterModel, side string, diags *diag.Diagnostics) *client.FirewallPolicyIPAddressFilter {
	ipAddresses := setToStrings(ctx, model.IPAddresses, side+"_filter.ip_addresses", diags)
	trafficMatchingListID := stringPointerValue(model.IPAddressTrafficMatchingListID)

	if len(ipAddresses) == 0 && trafficMatchingListID == nil {
		diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.ip_addresses or %s_filter.ip_address_traffic_matching_list_id must be set when %s_filter.type is IP_ADDRESS", side, side, side))
		return nil
	}
	if len(ipAddresses) > 0 && trafficMatchingListID != nil {
		diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.ip_addresses and %s_filter.ip_address_traffic_matching_list_id cannot both be set", side, side))
		return nil
	}

	filter := &client.FirewallPolicyIPAddressFilter{
		MatchOpposite: boolValueOrFalse(model.IPAddressMatchOpposite),
	}

	if len(ipAddresses) > 0 {
		items, err := expandGenericIPMatches(ipAddresses)
		if err != nil {
			diags.AddError("Invalid firewall policy filter", fmt.Sprintf("%s_filter.ip_addresses: %s", side, err.Error()))
			return nil
		}
		filter.Type = "IP_ADDRESSES"
		filter.Items = items
		return filter
	}

	filter.Type = "TRAFFIC_MATCHING_LIST"
	filter.TrafficMatchingListID = trafficMatchingListID
	return filter
}

func expandFirewallPolicyProtocolFilter(ctx context.Context, value types.Object, diags *diag.Diagnostics) *client.FirewallPolicyProtocolFilter {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	model, err := decodeFirewallPolicyProtocolFilter(ctx, value, "protocol_filter")
	if err != nil {
		diags.AddError("Invalid firewall policy protocol filter", err.Error())
		return nil
	}

	filter := &client.FirewallPolicyProtocolFilter{Type: model.Type.ValueString()}

	switch model.Type.ValueString() {
	case "NAMED_PROTOCOL":
		if model.NamedProtocol.IsNull() || model.NamedProtocol.ValueString() == "" {
			diags.AddError("Invalid firewall policy protocol filter", "protocol_filter.named_protocol is required when protocol_filter.type is NAMED_PROTOCOL")
			return nil
		}
		normalizedProtocol := strings.ToUpper(strings.TrimSpace(model.NamedProtocol.ValueString()))
		if normalizedProtocol != "ICMP" {
			diags.AddError("Invalid firewall policy protocol filter", "protocol_filter.named_protocol currently supports only ICMP on live UniFi controller builds; use protocol_filter = { type = \"PRESET\", preset_name = \"TCP_UDP\" } for TCP/UDP service rules")
			return nil
		}
		filter.Protocol = &client.FirewallPolicyNamedProtocol{Name: normalizedProtocol}
		matchOpposite := boolValueOrFalse(model.MatchOpposite)
		filter.MatchOpposite = &matchOpposite
	case "PROTOCOL_NUMBER":
		if model.ProtocolNumber.IsNull() || model.ProtocolNumber.IsUnknown() {
			diags.AddError("Invalid firewall policy protocol filter", "protocol_filter.protocol_number is required when protocol_filter.type is PROTOCOL_NUMBER")
			return nil
		}
		if value := model.ProtocolNumber.ValueInt64(); value < 0 || value > 255 {
			diags.AddError("Invalid firewall policy protocol filter", "protocol_filter.protocol_number must be between 0 and 255")
			return nil
		}
		filter.ProtocolNumber = int64PointerValue(model.ProtocolNumber)
		matchOpposite := boolValueOrFalse(model.MatchOpposite)
		filter.MatchOpposite = &matchOpposite
	case "PRESET":
		if model.PresetName.IsNull() || model.PresetName.ValueString() == "" {
			diags.AddError("Invalid firewall policy protocol filter", "protocol_filter.preset_name is required when protocol_filter.type is PRESET")
			return nil
		}
		filter.Preset = &client.FirewallPolicyProtocolPreset{Name: model.PresetName.ValueString()}
	default:
		diags.AddError("Invalid firewall policy protocol filter", "protocol_filter.type must be one of NAMED_PROTOCOL, PROTOCOL_NUMBER, or PRESET")
		return nil
	}

	return filter
}

func expandFirewallPolicySchedule(ctx context.Context, value types.Object, diags *diag.Diagnostics) *client.FirewallSchedule {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	model, err := decodeFirewallPolicySchedule(ctx, value, "schedule")
	if err != nil {
		diags.AddError("Invalid firewall policy schedule", err.Error())
		return nil
	}

	schedule := &client.FirewallSchedule{
		Mode:         model.Mode.ValueString(),
		Date:         stringPointerValue(model.Date),
		StartDate:    stringPointerValue(model.StartDate),
		StopDate:     stringPointerValue(model.StopDate),
		RepeatOnDays: setToStrings(ctx, model.RepeatOnDays, "schedule.repeat_on_days", diags),
	}
	if diags.HasError() {
		return nil
	}

	hasStart := !model.StartTime.IsNull() && model.StartTime.ValueString() != ""
	hasStop := !model.StopTime.IsNull() && model.StopTime.ValueString() != ""
	if hasStart != hasStop {
		diags.AddError("Invalid firewall policy schedule", "schedule.start_time and schedule.stop_time must be set together")
		return nil
	}
	if hasStart {
		schedule.TimeFilter = &client.FirewallScheduleTime{
			StartTime: model.StartTime.ValueString(),
			StopTime:  model.StopTime.ValueString(),
		}
	}

	switch schedule.Mode {
	case "EVERY_DAY":
		if schedule.TimeFilter == nil {
			diags.AddError("Invalid firewall policy schedule", "schedule.start_time and schedule.stop_time are required when schedule.mode is EVERY_DAY")
			return nil
		}
	case "EVERY_WEEK":
		if len(schedule.RepeatOnDays) == 0 {
			diags.AddError("Invalid firewall policy schedule", "schedule.repeat_on_days is required when schedule.mode is EVERY_WEEK")
			return nil
		}
	case "ONE_TIME_ONLY":
		if schedule.Date == nil || *schedule.Date == "" {
			diags.AddError("Invalid firewall policy schedule", "schedule.date is required when schedule.mode is ONE_TIME_ONLY")
			return nil
		}
		if schedule.TimeFilter == nil {
			diags.AddError("Invalid firewall policy schedule", "schedule.start_time and schedule.stop_time are required when schedule.mode is ONE_TIME_ONLY")
			return nil
		}
	case "CUSTOM":
		if len(schedule.RepeatOnDays) == 0 {
			diags.AddError("Invalid firewall policy schedule", "schedule.repeat_on_days is required when schedule.mode is CUSTOM")
			return nil
		}
		if schedule.StartDate == nil || *schedule.StartDate == "" || schedule.StopDate == nil || *schedule.StopDate == "" {
			diags.AddError("Invalid firewall policy schedule", "schedule.start_date and schedule.stop_date are required when schedule.mode is CUSTOM")
			return nil
		}
	default:
		diags.AddError("Invalid firewall policy schedule", "schedule.mode must be one of EVERY_DAY, EVERY_WEEK, ONE_TIME_ONLY, or CUSTOM")
		return nil
	}

	return schedule
}

func (r *firewallPolicyResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, firewallPolicy *client.FirewallPolicy) {
	model, diagnostics := buildFirewallPolicyStateModel(ctx, siteID, firewallPolicy)
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	diags.Append(state.Set(ctx, &model)...)
}
