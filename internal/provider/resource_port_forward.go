package provider

import (
	"context"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = (*portForwardResource)(nil)
	_ resource.ResourceWithConfigure   = (*portForwardResource)(nil)
	_ resource.ResourceWithImportState = (*portForwardResource)(nil)
)

type portForwardResource struct{ providerData *providerData }

type portForwardModel struct {
	ID              types.String `tfsdk:"id"`
	SiteID          types.String `tfsdk:"site_id"`
	Name            types.String `tfsdk:"name"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	WANInterface    types.String `tfsdk:"wan_interface"`
	Protocol        types.String `tfsdk:"protocol"`
	Source          types.String `tfsdk:"source"`
	DestinationPort types.String `tfsdk:"destination_port"`
	ForwardIP       types.String `tfsdk:"forward_ip"`
	ForwardPort     types.String `tfsdk:"forward_port"`
	LoggingEnabled  types.Bool   `tfsdk:"logging_enabled"`
}

func NewPortForwardResource() resource.Resource { return &portForwardResource{} }

func (r *portForwardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forward"
}

func (r *portForwardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a UniFi WAN port forwarding rule using the legacy local Network API. Import with `<site_id>/<id>`, where `id` is the legacy rule `_id`. Updates preserve controller fields that are not exposed by this resource, including destination IP filters and source firewall groups. Existing source-group restrictions remain active when changing modelled settings. Configure IP or CIDR restrictions using `source`.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Legacy port forwarding rule ID (`_id`)."},
			"site_id":          schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Integration site UUID. Changing the site replaces the rule."},
			"name":             schema.StringAttribute{Required: true, MarkdownDescription: "Rule name."},
			"enabled":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), MarkdownDescription: "Whether the rule is enabled. Defaults to `true`."},
			"wan_interface":    schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("wan"), Validators: []validator.String{stringOneOf{"wan", "wan2", "both"}}, MarkdownDescription: "WAN interface: `wan`, `wan2`, or `both`. Defaults to `wan`."},
			"protocol":         schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("tcp_udp"), Validators: []validator.String{stringOneOf{"tcp", "udp", "tcp_udp"}}, MarkdownDescription: "Protocol: `tcp`, `udp`, or `tcp_udp`. Defaults to `tcp_udp`."},
			"source":           schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("any"), Validators: []validator.String{portForwardSourceValidator{}}, MarkdownDescription: "Allowed source IP, CIDR, or `any`. Defaults to `any`; a specific source enables source limiting."},
			"destination_port": schema.StringAttribute{Required: true, MarkdownDescription: "WAN destination port, range, or comma-separated list. Kept as a string, for example `3074`, `3000-3010`, or `80,443`."},
			"forward_ip":       schema.StringAttribute{Required: true, MarkdownDescription: "Internal IPv4 address to forward traffic to."},
			"forward_port":     schema.StringAttribute{Required: true, MarkdownDescription: "Internal destination port, range, or comma-separated list, expressed as a string."},
			"logging_enabled":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), MarkdownDescription: "Whether to log traffic matching the rule. Defaults to `false`."},
		},
	}
}

type portForwardSourceValidator struct{}

func (portForwardSourceValidator) Description(context.Context) string {
	return "Must be the literal any, an IP address, or a CIDR prefix."
}
func (v portForwardSourceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v portForwardSourceValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "any" {
		return
	}
	if address, err := netip.ParseAddr(value); err == nil && address.Zone() == "" {
		return
	}
	if _, err := netip.ParsePrefix(value); err == nil {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid source", v.Description(ctx))
}

func (r *portForwardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *portForwardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portForwardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.providerData.client.CreatePortForward(ctx, plan.SiteID.ValueString(), expandPortForward(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create port forward", err.Error())
		return
	}
	writePortForwardState(ctx, &resp.State, &resp.Diagnostics, plan.SiteID, rule)
}

func (r *portForwardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portForwardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.providerData.client.GetPortForward(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read port forward", err.Error())
		return
	}
	writePortForwardState(ctx, &resp.State, &resp.Diagnostics, state.SiteID, rule)
}

func (r *portForwardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state portForwardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.providerData.client.UpdatePortForward(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), expandPortForward(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update port forward", err.Error())
		return
	}
	writePortForwardState(ctx, &resp.State, &resp.Diagnostics, plan.SiteID, rule)
}

func (r *portForwardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state portForwardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.providerData.client.DeletePortForward(ctx, state.SiteID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete port forward", err.Error())
	}
}

func (r *portForwardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}

func expandPortForward(model portForwardModel) client.PortForward {
	return client.PortForward{Name: model.Name.ValueString(), Enabled: model.Enabled.ValueBool(), WANInterface: model.WANInterface.ValueString(), Protocol: model.Protocol.ValueString(), Source: model.Source.ValueString(), DestinationPort: model.DestinationPort.ValueString(), ForwardIP: model.ForwardIP.ValueString(), ForwardPort: model.ForwardPort.ValueString(), LoggingEnabled: model.LoggingEnabled.ValueBool()}
}

func flattenPortForward(siteID types.String, rule *client.PortForward) portForwardModel {
	return portForwardModel{ID: types.StringValue(rule.ID), SiteID: siteID, Name: types.StringValue(rule.Name), Enabled: types.BoolValue(rule.Enabled), WANInterface: types.StringValue(rule.WANInterface), Protocol: types.StringValue(rule.Protocol), Source: types.StringValue(rule.Source), DestinationPort: types.StringValue(rule.DestinationPort), ForwardIP: types.StringValue(rule.ForwardIP), ForwardPort: types.StringValue(rule.ForwardPort), LoggingEnabled: types.BoolValue(rule.LoggingEnabled)}
}

func writePortForwardState(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, siteID types.String, rule *client.PortForward) {
	model := flattenPortForward(siteID, rule)
	diags.Append(state.Set(ctx, &model)...)
}
