package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*dnsPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*dnsPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*dnsPolicyResource)(nil)
)

type dnsPolicyResource struct {
	providerData *providerData
}

type dnsPolicyResourceModel struct {
	ID               types.String `tfsdk:"id"`
	SiteID           types.String `tfsdk:"site_id"`
	Type             types.String `tfsdk:"type"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Domain           types.String `tfsdk:"domain"`
	IPv4Address      types.String `tfsdk:"ipv4_address"`
	IPv6Address      types.String `tfsdk:"ipv6_address"`
	TargetDomain     types.String `tfsdk:"target_domain"`
	MailServerDomain types.String `tfsdk:"mail_server_domain"`
	Priority         types.Int64  `tfsdk:"priority"`
	Text             types.String `tfsdk:"text"`
	ServerDomain     types.String `tfsdk:"server_domain"`
	Service          types.String `tfsdk:"service"`
	Protocol         types.String `tfsdk:"protocol"`
	Port             types.Int64  `tfsdk:"port"`
	Weight           types.Int64  `tfsdk:"weight"`
	IPAddress        types.String `tfsdk:"ip_address"`
	TTLSeconds       types.Int64  `tfsdk:"ttl_seconds"`
}

func NewDNSPolicyResource() resource.Resource {
	return &dnsPolicyResource{}
}

func (r *dnsPolicyResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dns_policy"
}

func (r *dnsPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi DNS policy. The active type-specific attributes depend on `type`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "DNS policy type. Supported values: `A_RECORD`, `AAAA_RECORD`, `CNAME_RECORD`, `MX_RECORD`, `TXT_RECORD`, `SRV_RECORD`, `FORWARD_DOMAIN`.",
			},
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"domain": schema.StringAttribute{
				Required: true,
			},
			"ipv4_address": schema.StringAttribute{
				Optional: true,
			},
			"ipv6_address": schema.StringAttribute{
				Optional: true,
			},
			"target_domain": schema.StringAttribute{
				Optional: true,
			},
			"mail_server_domain": schema.StringAttribute{
				Optional: true,
			},
			"priority": schema.Int64Attribute{
				Optional: true,
			},
			"text": schema.StringAttribute{
				Optional: true,
			},
			"server_domain": schema.StringAttribute{
				Optional: true,
			},
			"service": schema.StringAttribute{
				Optional: true,
			},
			"protocol": schema.StringAttribute{
				Optional: true,
			},
			"port": schema.Int64Attribute{
				Optional: true,
			},
			"weight": schema.Int64Attribute{
				Optional: true,
			},
			"ip_address": schema.StringAttribute{
				Optional: true,
			},
			"ttl_seconds": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}

func (r *dnsPolicyResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.providerData = configureResource(request, response)
}

func (r *dnsPolicyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan dnsPolicyResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := expandDNSPolicy(plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	created, err := r.providerData.client.CreateDNSPolicy(ctx, plan.SiteID.ValueString(), apiPolicy)
	if err != nil {
		response.Diagnostics.AddError("Unable to create DNS policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, created)
}

func (r *dnsPolicyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state dnsPolicyResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	dnsPolicy, err := r.providerData.client.GetDNSPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, state.SiteID, dnsPolicy)
}

func (r *dnsPolicyResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan dnsPolicyResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	var state dnsPolicyResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiPolicy := expandDNSPolicy(plan, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	updated, err := r.providerData.client.UpdateDNSPolicy(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), apiPolicy)
	if err != nil {
		response.Diagnostics.AddError("Unable to update DNS policy", err.Error())
		return
	}

	r.writeState(ctx, &response.State, &response.Diagnostics, plan.SiteID, updated)
}

func (r *dnsPolicyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state dnsPolicyResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.providerData.client.DeleteDNSPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete DNS policy", err.Error())
	}
}

func (r *dnsPolicyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importCompositeID(ctx, request, response)
}

func expandDNSPolicy(model dnsPolicyResourceModel, diags *diag.Diagnostics) client.DNSPolicy {
	if err := validateDNSPolicyModel(model); err != nil {
		diags.AddError("Invalid DNS policy configuration", err.Error())
	}

	return client.DNSPolicy{
		Type:             model.Type.ValueString(),
		Enabled:          model.Enabled.ValueBool(),
		Domain:           stringPointerValue(model.Domain),
		IPv4Address:      stringPointerValue(model.IPv4Address),
		IPv6Address:      stringPointerValue(model.IPv6Address),
		TargetDomain:     stringPointerValue(model.TargetDomain),
		MailServerDomain: stringPointerValue(model.MailServerDomain),
		Priority:         int64PointerValue(model.Priority),
		Text:             stringPointerValue(model.Text),
		ServerDomain:     stringPointerValue(model.ServerDomain),
		Service:          stringPointerValue(model.Service),
		Protocol:         stringPointerValue(model.Protocol),
		Port:             int64PointerValue(model.Port),
		Weight:           int64PointerValue(model.Weight),
		IPAddress:        stringPointerValue(model.IPAddress),
		TTLSeconds:       int64PointerValue(model.TTLSeconds),
	}
}

func validateDNSPolicyModel(model dnsPolicyResourceModel) error {
	typeValue := model.Type.ValueString()
	if typeValue == "" {
		return fmt.Errorf("type must not be empty")
	}

	switch typeValue {
	case "A_RECORD":
		return validateDNSRecordFields(model, []string{"ipv4_address", "ttl_seconds"})
	case "AAAA_RECORD":
		return validateDNSRecordFields(model, []string{"ipv6_address", "ttl_seconds"})
	case "CNAME_RECORD":
		return validateDNSRecordFields(model, []string{"target_domain", "ttl_seconds"})
	case "MX_RECORD":
		return validateDNSRecordFields(model, []string{"mail_server_domain", "priority"})
	case "TXT_RECORD":
		return validateDNSRecordFields(model, []string{"text"})
	case "SRV_RECORD":
		return validateDNSRecordFields(model, []string{"server_domain", "service", "protocol", "port", "priority", "weight"})
	case "FORWARD_DOMAIN":
		return validateDNSRecordFields(model, []string{"ip_address"})
	default:
		return fmt.Errorf("type must be one of A_RECORD, AAAA_RECORD, CNAME_RECORD, MX_RECORD, TXT_RECORD, SRV_RECORD, or FORWARD_DOMAIN")
	}
}

func validateDNSRecordFields(model dnsPolicyResourceModel, required []string) error {
	fields := map[string]bool{
		"ipv4_address":       !model.IPv4Address.IsNull() && model.IPv4Address.ValueString() != "",
		"ipv6_address":       !model.IPv6Address.IsNull() && model.IPv6Address.ValueString() != "",
		"target_domain":      !model.TargetDomain.IsNull() && model.TargetDomain.ValueString() != "",
		"mail_server_domain": !model.MailServerDomain.IsNull() && model.MailServerDomain.ValueString() != "",
		"priority":           !model.Priority.IsNull(),
		"text":               !model.Text.IsNull() && model.Text.ValueString() != "",
		"server_domain":      !model.ServerDomain.IsNull() && model.ServerDomain.ValueString() != "",
		"service":            !model.Service.IsNull() && model.Service.ValueString() != "",
		"protocol":           !model.Protocol.IsNull() && model.Protocol.ValueString() != "",
		"port":               !model.Port.IsNull(),
		"weight":             !model.Weight.IsNull(),
		"ip_address":         !model.IPAddress.IsNull() && model.IPAddress.ValueString() != "",
		"ttl_seconds":        !model.TTLSeconds.IsNull(),
	}

	allowed := map[string]bool{}
	for _, field := range required {
		allowed[field] = true
		if !fields[field] {
			return fmt.Errorf("%s is required when type is %s", field, model.Type.ValueString())
		}
	}

	for field, present := range fields {
		if present && !allowed[field] {
			return fmt.Errorf("%s is not valid when type is %s", field, model.Type.ValueString())
		}
	}

	return nil
}

func (r *dnsPolicyResource) writeState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, dnsPolicy *client.DNSPolicy) {
	model := dnsPolicyResourceModel{
		ID:               types.StringValue(dnsPolicy.ID),
		SiteID:           siteID,
		Type:             types.StringValue(dnsPolicy.Type),
		Enabled:          types.BoolValue(dnsPolicy.Enabled),
		Domain:           nullableString(dnsPolicy.Domain),
		IPv4Address:      nullableString(dnsPolicy.IPv4Address),
		IPv6Address:      nullableString(dnsPolicy.IPv6Address),
		TargetDomain:     nullableString(dnsPolicy.TargetDomain),
		MailServerDomain: nullableString(dnsPolicy.MailServerDomain),
		Priority:         nullableInt64(dnsPolicy.Priority),
		Text:             nullableString(dnsPolicy.Text),
		ServerDomain:     nullableString(dnsPolicy.ServerDomain),
		Service:          nullableString(dnsPolicy.Service),
		Protocol:         nullableString(dnsPolicy.Protocol),
		Port:             nullableInt64(dnsPolicy.Port),
		Weight:           nullableInt64(dnsPolicy.Weight),
		IPAddress:        nullableString(dnsPolicy.IPAddress),
		TTLSeconds:       nullableInt64(dnsPolicy.TTLSeconds),
	}

	diags.Append(state.Set(ctx, &model)...)
}
