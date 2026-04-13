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
	_ resource.Resource                = (*aclRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*aclRuleResource)(nil)
	_ resource.ResourceWithImportState = (*aclRuleResource)(nil)
)

type aclRuleResource struct {
	providerData *providerData
}

type aclRuleResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	SiteID               types.String `tfsdk:"site_id"`
	Type                 types.String `tfsdk:"type"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Action               types.String `tfsdk:"action"`
	EnforcingDeviceIDs   types.Set    `tfsdk:"enforcing_device_ids"`
	ProtocolFilter       types.Set    `tfsdk:"protocol_filter"`
	NetworkIDFilter      types.String `tfsdk:"network_id_filter"`
	SourceIPFilter       types.Object `tfsdk:"source_ip_filter"`
	DestinationIPFilter  types.Object `tfsdk:"destination_ip_filter"`
	SourceMACFilter      types.Object `tfsdk:"source_mac_filter"`
	DestinationMACFilter types.Object `tfsdk:"destination_mac_filter"`
	Index                types.Int64  `tfsdk:"index"`
}

type aclIPFilterModel struct {
	Type                 types.String `tfsdk:"type"`
	IPAddressesOrSubnets types.Set    `tfsdk:"ip_addresses_or_subnets"`
	NetworkIDs           types.Set    `tfsdk:"network_ids"`
	Ports                types.Set    `tfsdk:"ports"`
}

type aclMACFilterModel struct {
	Type         types.String `tfsdk:"type"`
	MacAddresses types.Set    `tfsdk:"mac_addresses"`
	PrefixLength types.Int64  `tfsdk:"prefix_length"`
}

func aclIPFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":                    types.StringType,
		"ip_addresses_or_subnets": types.SetType{ElemType: types.StringType},
		"network_ids":             types.SetType{ElemType: types.StringType},
		"ports":                   types.SetType{ElemType: types.Int64Type},
	}
}

func aclMACFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":          types.StringType,
		"mac_addresses": types.SetType{ElemType: types.StringType},
		"prefix_length": types.Int64Type,
	}
}

func NewACLRuleResource() resource.Resource {
	return &aclRuleResource{}
}

func (r *aclRuleResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_acl_rule"
}

func (r *aclRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi ACL rule. The active nested filter blocks depend on `type`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ACL rule type. Supported values: `IPV4`, `MAC`.",
			},
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"action": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ACL rule action. Supported values: `ALLOW`, `BLOCK`.",
			},
			"enforcing_device_ids": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional list of switch-capable device IDs where the ACL rule is enforced. When omitted, the rule applies to all compatible switches.",
			},
			"protocol_filter": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional protocol filter for `IPV4` ACL rules. Supported values: `TCP`, `UDP`.",
			},
			"network_id_filter": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Required for `MAC` ACL rules. The network ID to which the rule applies.",
			},
			"source_ip_filter": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"ip_addresses_or_subnets": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"network_ids": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"ports": schema.SetAttribute{
						Optional:    true,
						ElementType: types.Int64Type,
					},
				},
			},
			"destination_ip_filter": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"ip_addresses_or_subnets": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"network_ids": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"ports": schema.SetAttribute{
						Optional:    true,
						ElementType: types.Int64Type,
					},
				},
			},
			"source_mac_filter": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"mac_addresses": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
					"prefix_length": schema.Int64Attribute{
						Optional: true,
					},
				},
			},
			"destination_mac_filter": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"mac_addresses": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
					"prefix_length": schema.Int64Attribute{
						Optional: true,
					},
				},
			},
			"index": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Controller-assigned ACL rule ordering index.",
			},
		},
	}
}

func (r *aclRuleResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *aclRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan aclRuleResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiRule := r.expandACLRule(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateACLRule(ctx, plan.SiteID.ValueString(), apiRule)
	if err != nil {
		response.Diagnostics.AddError("Unable to create ACL rule", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *aclRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state aclRuleResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	aclRule, err := r.providerData.client.GetACLRule(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read ACL rule", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, aclRule)
}

func (r *aclRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan aclRuleResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state aclRuleResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiRule := r.expandACLRule(ctx, plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateACLRule(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiRule)
	if err != nil {
		response.Diagnostics.AddError("Unable to update ACL rule", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *aclRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state aclRuleResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteACLRule(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete ACL rule", err.Error())
	}
}

func (r *aclRuleResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func (r *aclRuleResource) expandACLRule(ctx context.Context, plan aclRuleResourceModel, diags *diag.Diagnostics) client.ACLRule {
	if err := validateACLRuleModel(ctx, plan); err != nil {
		diags.AddError("Invalid ACL rule configuration", err.Error())
	}

	rule := client.ACLRule{
		Type:        plan.Type.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
		Name:        plan.Name.ValueString(),
		Description: stringPointerValue(plan.Description),
		Action:      plan.Action.ValueString(),
	}

	deviceIDs := setToStrings(ctx, plan.EnforcingDeviceIDs, "enforcing_device_ids", diags)
	if len(deviceIDs) > 0 {
		rule.EnforcingDeviceFilter = &client.ACLRuleDeviceFilter{
			Type:      "DEVICES",
			DeviceIDs: deviceIDs,
		}
	}

	switch plan.Type.ValueString() {
	case "IPV4":
		rule.ProtocolFilter = setToStrings(ctx, plan.ProtocolFilter, "protocol_filter", diags)
		rule.SourceFilter = expandACLIPFilter(ctx, plan.SourceIPFilter, "source_ip_filter", diags)
		rule.DestinationFilter = expandACLIPFilter(ctx, plan.DestinationIPFilter, "destination_ip_filter", diags)
	case "MAC":
		rule.NetworkIDFilter = stringPointerValue(plan.NetworkIDFilter)
		rule.SourceFilter = expandACLMACFilter(ctx, plan.SourceMACFilter, "source_mac_filter", diags)
		rule.DestinationFilter = expandACLMACFilter(ctx, plan.DestinationMACFilter, "destination_mac_filter", diags)
	}

	return rule
}

func validateACLRuleModel(ctx context.Context, plan aclRuleResourceModel) error {
	if action := plan.Action.ValueString(); action != "ALLOW" && action != "BLOCK" {
		return fmt.Errorf("action must be one of ALLOW or BLOCK")
	}

	switch plan.Type.ValueString() {
	case "IPV4":
		if !plan.NetworkIDFilter.IsNull() {
			return fmt.Errorf("network_id_filter is only valid for MAC ACL rules")
		}
		if !plan.SourceMACFilter.IsNull() || !plan.DestinationMACFilter.IsNull() {
			return fmt.Errorf("source_mac_filter and destination_mac_filter are only valid for MAC ACL rules")
		}

		protocols, err := decodeACLStringSet(ctx, plan.ProtocolFilter)
		if err != nil {
			return fmt.Errorf("unable to decode protocol_filter")
		}
		for _, protocol := range protocols {
			if protocol != "TCP" && protocol != "UDP" {
				return fmt.Errorf("protocol_filter values must be TCP or UDP")
			}
		}

		if err := validateACLIPFilterObject(ctx, plan.SourceIPFilter, "source_ip_filter"); err != nil {
			return err
		}
		if err := validateACLIPFilterObject(ctx, plan.DestinationIPFilter, "destination_ip_filter"); err != nil {
			return err
		}
	case "MAC":
		if !plan.ProtocolFilter.IsNull() {
			return fmt.Errorf("protocol_filter is only valid for IPV4 ACL rules")
		}
		if !plan.SourceIPFilter.IsNull() || !plan.DestinationIPFilter.IsNull() {
			return fmt.Errorf("source_ip_filter and destination_ip_filter are only valid for IPV4 ACL rules")
		}
		if plan.NetworkIDFilter.IsNull() || plan.NetworkIDFilter.ValueString() == "" {
			return fmt.Errorf("network_id_filter is required when type is MAC")
		}

		if err := validateACLMACFilterObject(ctx, plan.SourceMACFilter, "source_mac_filter"); err != nil {
			return err
		}
		if err := validateACLMACFilterObject(ctx, plan.DestinationMACFilter, "destination_mac_filter"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("type must be one of IPV4 or MAC")
	}

	return nil
}

func validateACLIPFilterObject(ctx context.Context, value types.Object, path string) error {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var model aclIPFilterModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode %s", path)
	}

	subnets, err := decodeACLStringSet(ctx, model.IPAddressesOrSubnets)
	if err != nil {
		return fmt.Errorf("unable to decode %s.ip_addresses_or_subnets", path)
	}
	networkIDs, err := decodeACLStringSet(ctx, model.NetworkIDs)
	if err != nil {
		return fmt.Errorf("unable to decode %s.network_ids", path)
	}
	ports, err := decodeACLIntSet(ctx, model.Ports)
	if err != nil {
		return fmt.Errorf("unable to decode %s.ports", path)
	}

	switch model.Type.ValueString() {
	case "IP_ADDRESSES_OR_SUBNETS":
		if len(subnets) == 0 {
			return fmt.Errorf("%s.ip_addresses_or_subnets is required when %s.type is IP_ADDRESSES_OR_SUBNETS", path, path)
		}
		if len(networkIDs) > 0 {
			return fmt.Errorf("%s.network_ids is not valid when %s.type is IP_ADDRESSES_OR_SUBNETS", path, path)
		}
	case "NETWORKS":
		if len(networkIDs) == 0 {
			return fmt.Errorf("%s.network_ids is required when %s.type is NETWORKS", path, path)
		}
		if len(subnets) > 0 {
			return fmt.Errorf("%s.ip_addresses_or_subnets is not valid when %s.type is NETWORKS", path, path)
		}
	case "PORTS":
		if len(ports) == 0 {
			return fmt.Errorf("%s.ports is required when %s.type is PORTS", path, path)
		}
		if len(subnets) > 0 || len(networkIDs) > 0 {
			return fmt.Errorf("%s.ip_addresses_or_subnets and %s.network_ids are not valid when %s.type is PORTS", path, path, path)
		}
	default:
		return fmt.Errorf("%s.type must be one of IP_ADDRESSES_OR_SUBNETS, NETWORKS, or PORTS", path)
	}

	return nil
}

func validateACLMACFilterObject(ctx context.Context, value types.Object, path string) error {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var model aclMACFilterModel
	if diags := value.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		return fmt.Errorf("unable to decode %s", path)
	}

	macAddresses, err := decodeACLStringSet(ctx, model.MacAddresses)
	if err != nil {
		return fmt.Errorf("unable to decode %s.mac_addresses", path)
	}

	if model.Type.ValueString() != "MAC_ADDRESSES" {
		return fmt.Errorf("%s.type must be MAC_ADDRESSES", path)
	}
	if len(macAddresses) == 0 {
		return fmt.Errorf("%s.mac_addresses must contain at least one value", path)
	}

	return nil
}

func decodeACLStringSet(ctx context.Context, value types.Set) ([]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var values []string
	if diags := value.ElementsAs(ctx, &values, false); diags.HasError() {
		return nil, fmt.Errorf("unable to decode set")
	}

	return values, nil
}

func decodeACLIntSet(ctx context.Context, value types.Set) ([]int64, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var values []int64
	if diags := value.ElementsAs(ctx, &values, false); diags.HasError() {
		return nil, fmt.Errorf("unable to decode set")
	}

	return values, nil
}

func expandACLIPFilter(ctx context.Context, value types.Object, path string, diags *diag.Diagnostics) *client.ACLRuleEndpointFilter {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var model aclIPFilterModel
	diags.Append(value.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	return &client.ACLRuleEndpointFilter{
		Type:                 model.Type.ValueString(),
		IPAddressesOrSubnets: setToStrings(ctx, model.IPAddressesOrSubnets, path+".ip_addresses_or_subnets", diags),
		NetworkIDs:           setToStrings(ctx, model.NetworkIDs, path+".network_ids", diags),
		PortFilter:           setToInt64s(ctx, model.Ports, path+".ports", diags),
	}
}

func expandACLMACFilter(ctx context.Context, value types.Object, path string, diags *diag.Diagnostics) *client.ACLRuleEndpointFilter {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var model aclMACFilterModel
	diags.Append(value.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	return &client.ACLRuleEndpointFilter{
		Type:         model.Type.ValueString(),
		MacAddresses: setToStrings(ctx, model.MacAddresses, path+".mac_addresses", diags),
		PrefixLength: int64PointerValue(model.PrefixLength),
	}
}

func (r *aclRuleResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, aclRule *client.ACLRule) {
	enforcingDeviceIDs := types.SetNull(types.StringType)
	if aclRule.EnforcingDeviceFilter != nil {
		var diagnostics diag.Diagnostics
		enforcingDeviceIDs, diagnostics = stringSetValue(ctx, aclRule.EnforcingDeviceFilter.DeviceIDs)
		diags.Append(diagnostics...)
	}

	protocolFilter := types.SetNull(types.StringType)
	if len(aclRule.ProtocolFilter) > 0 {
		var diagnostics diag.Diagnostics
		protocolFilter, diagnostics = stringSetValue(ctx, aclRule.ProtocolFilter)
		diags.Append(diagnostics...)
	}

	sourceIPFilter, diagnostics := flattenACLIPFilter(ctx, aclRule.Type, aclRule.SourceFilter)
	diags.Append(diagnostics...)
	destinationIPFilter, diagnostics := flattenACLIPFilter(ctx, aclRule.Type, aclRule.DestinationFilter)
	diags.Append(diagnostics...)
	sourceMACFilter, diagnostics := flattenACLMACFilter(ctx, aclRule.Type, aclRule.SourceFilter)
	diags.Append(diagnostics...)
	destinationMACFilter, diagnostics := flattenACLMACFilter(ctx, aclRule.Type, aclRule.DestinationFilter)
	diags.Append(diagnostics...)
	if diags.HasError() {
		return
	}

	model := aclRuleResourceModel{
		ID:                   types.StringValue(aclRule.ID),
		SiteID:               siteID,
		Type:                 types.StringValue(aclRule.Type),
		Enabled:              types.BoolValue(aclRule.Enabled),
		Name:                 types.StringValue(aclRule.Name),
		Description:          nullableString(aclRule.Description),
		Action:               types.StringValue(aclRule.Action),
		EnforcingDeviceIDs:   enforcingDeviceIDs,
		ProtocolFilter:       protocolFilter,
		NetworkIDFilter:      nullableString(aclRule.NetworkIDFilter),
		SourceIPFilter:       sourceIPFilter,
		DestinationIPFilter:  destinationIPFilter,
		SourceMACFilter:      sourceMACFilter,
		DestinationMACFilter: destinationMACFilter,
		Index:                types.Int64Value(aclRule.Index),
	}

	diags.Append(state.Set(ctx, &model)...)
}

func flattenACLIPFilter(ctx context.Context, ruleType string, filter *client.ACLRuleEndpointFilter) (types.Object, diag.Diagnostics) {
	if ruleType != "IPV4" || filter == nil {
		return types.ObjectNull(aclIPFilterAttrTypes()), nil
	}

	subnets, diagnostics := stringSetValue(ctx, filter.IPAddressesOrSubnets)
	if diagnostics.HasError() {
		return types.ObjectNull(aclIPFilterAttrTypes()), diagnostics
	}
	networkIDs, diagnosticsSet := stringSetValue(ctx, filter.NetworkIDs)
	diagnostics.Append(diagnosticsSet...)
	ports, diagnosticsSet := int64SetValue(ctx, filter.PortFilter)
	diagnostics.Append(diagnosticsSet...)

	model := aclIPFilterModel{
		Type:                 types.StringValue(filter.Type),
		IPAddressesOrSubnets: subnets,
		NetworkIDs:           networkIDs,
		Ports:                ports,
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, aclIPFilterAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}

func flattenACLMACFilter(ctx context.Context, ruleType string, filter *client.ACLRuleEndpointFilter) (types.Object, diag.Diagnostics) {
	if ruleType != "MAC" || filter == nil {
		return types.ObjectNull(aclMACFilterAttrTypes()), nil
	}

	macAddresses, diagnostics := stringSetValue(ctx, filter.MacAddresses)
	if diagnostics.HasError() {
		return types.ObjectNull(aclMACFilterAttrTypes()), diagnostics
	}

	model := aclMACFilterModel{
		Type:         types.StringValue(filter.Type),
		MacAddresses: macAddresses,
		PrefixLength: nullableInt64(filter.PrefixLength),
	}

	value, diagnosticsObject := types.ObjectValueFrom(ctx, aclMACFilterAttrTypes(), model)
	diagnostics.Append(diagnosticsObject...)
	return value, diagnostics
}
