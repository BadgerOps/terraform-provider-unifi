package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                   = (*wanSLAResource)(nil)
	_ resource.ResourceWithConfigure      = (*wanSLAResource)(nil)
	_ resource.ResourceWithImportState    = (*wanSLAResource)(nil)
	_ resource.ResourceWithValidateConfig = (*wanSLAResource)(nil)
)

var (
	wanSLAHealthTypes      = stringOneOf{"ALL", "ANY", "MAJORITY", "CUSTOM"}
	wanSLAMonitorTypes     = stringOneOf{"ICMP", "DNS"}
	wanSLAThresholdPolices = stringOneOf{"any", "all", "any-two", "loss-or-all-others", "latency-or-all-others", "jitter-or-all-others", "loss-plus-one", "latency-plus-one", "jitter-plus-one"}
)

type wanSLAResource struct{ providerData *providerData }

type wanSLAModel struct {
	ID                types.String `tfsdk:"id"`
	SiteID            types.String `tfsdk:"site_id"`
	Name              types.String `tfsdk:"name"`
	MonitorHealthType types.String `tfsdk:"monitor_health_type"`
	Monitors          types.List   `tfsdk:"monitors"`
}

type wanSLAMonitorModel struct {
	Type             types.String `tfsdk:"type"`
	Target           types.String `tfsdk:"target"`
	Domain           types.String `tfsdk:"domain"`
	Interval         types.Int64  `tfsdk:"interval"`
	TimePeriod       types.Int64  `tfsdk:"time_period"`
	LossThreshold    types.Int64  `tfsdk:"loss_threshold"`
	LatencyThreshold types.Int64  `tfsdk:"latency_threshold"`
	JitterThreshold  types.Int64  `tfsdk:"jitter_threshold"`
	ThresholdPolicy  types.String `tfsdk:"threshold_policy"`
}

var wanSLAMonitorAttrTypes = map[string]attr.Type{
	"type":              types.StringType,
	"target":            types.StringType,
	"domain":            types.StringType,
	"interval":          types.Int64Type,
	"time_period":       types.Int64Type,
	"loss_threshold":    types.Int64Type,
	"latency_threshold": types.Int64Type,
	"jitter_threshold":  types.Int64Type,
	"threshold_policy":  types.StringType,
}

func NewWANSLAResource() resource.Resource { return &wanSLAResource{} }

func (r *wanSLAResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wan_sla"
}

func (r *wanSLAResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a WAN SLA (Settings > Internet > WAN SLA): the monitors that decide whether a WAN counts as up for failover and load balancing. Uses the Network application's v2 API (`/v2/api/site/<site>/wan-slas`), which the integration API does not cover. A WAN refers to its SLA by ID (`wan_sla` on the WAN); this resource doesn't change which WAN uses which SLA. The monitor list is authoritative and numbered in order. Import with `<site_id>/<sla_id>`.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "The SLA's ID, as a WAN's `wan_sla` refers to it."},
			"site_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Integration site UUID."},
			"name":    schema.StringAttribute{Required: true, MarkdownDescription: "Name shown in the UI."},
			"monitor_health_type": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{wanSLAHealthTypes},
				MarkdownDescription: "How the monitors combine: `ALL` (the WAN is down when every monitor fails), `ANY` (when one fails), `MAJORITY` (three or more monitors) or `CUSTOM`. Keeps the controller's value when not set.",
			},
			"monitors": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "The probes, in order.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"type":              schema.StringAttribute{Required: true, Validators: []validator.String{wanSLAMonitorTypes}, MarkdownDescription: "`ICMP` pings `target`; `DNS` resolves `domain` through the server `target`."},
					"target":            schema.StringAttribute{Optional: true, MarkdownDescription: "Host to ping, or the DNS server to ask."},
					"domain":            schema.StringAttribute{Optional: true, MarkdownDescription: "Name to resolve, for `DNS` monitors."},
					"interval":          schema.Int64Attribute{Required: true, MarkdownDescription: "Seconds between probes."},
					"time_period":       schema.Int64Attribute{Required: true, MarkdownDescription: "Seconds of probes the thresholds are evaluated over."},
					"loss_threshold":    schema.Int64Attribute{Optional: true, MarkdownDescription: "Packet loss in percent (0-100) at which the monitor fails. Not set: not evaluated."},
					"latency_threshold": schema.Int64Attribute{Optional: true, MarkdownDescription: "Latency in milliseconds at which the monitor fails. Not set: not evaluated."},
					"jitter_threshold":  schema.Int64Attribute{Optional: true, MarkdownDescription: "Jitter in milliseconds at which the monitor fails. Not set: not evaluated."},
					"threshold_policy":  schema.StringAttribute{Required: true, Validators: []validator.String{wanSLAThresholdPolices}, MarkdownDescription: "Which exceeded thresholds fail the monitor, for example `any` (one is enough) or `all`."},
				}},
			},
		},
	}
}

func (r *wanSLAResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config wanSLAModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Monitors.IsNull() || config.Monitors.IsUnknown() {
		return
	}
	var monitors []wanSLAMonitorModel
	resp.Diagnostics.Append(config.Monitors.ElementsAs(ctx, &monitors, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(monitors) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("monitors"), "No monitors", "A WAN SLA needs at least one monitor.")
	}
	if config.MonitorHealthType.ValueString() == "MAJORITY" && len(monitors) < 3 {
		resp.Diagnostics.AddAttributeError(path.Root("monitor_health_type"), "MAJORITY needs three monitors", fmt.Sprintf("The controller offers MAJORITY only with three or more monitors; got %d.", len(monitors)))
	}
	for i, m := range monitors {
		at := path.Root("monitors").AtListIndex(i)
		positiveInt64(m.Interval, at.AtName("interval"), &resp.Diagnostics)
		positiveInt64(m.TimePeriod, at.AtName("time_period"), &resp.Diagnostics)
		positiveInt64(m.LatencyThreshold, at.AtName("latency_threshold"), &resp.Diagnostics)
		positiveInt64(m.JitterThreshold, at.AtName("jitter_threshold"), &resp.Diagnostics)
		if v := m.LossThreshold; knownInt64(v) && (v.ValueInt64() < 0 || v.ValueInt64() > 100) {
			resp.Diagnostics.AddAttributeError(at.AtName("loss_threshold"), "Invalid loss threshold", fmt.Sprintf("A percentage from 0 to 100; got %d.", v.ValueInt64()))
		}
		if knownInt64(m.Interval) && knownInt64(m.TimePeriod) && m.TimePeriod.ValueInt64() < m.Interval.ValueInt64() {
			resp.Diagnostics.AddAttributeError(at.AtName("time_period"), "Period shorter than the interval", "time_period must cover at least one probe.")
		}
		if m.Type.ValueString() == "ICMP" && m.Target.IsNull() {
			resp.Diagnostics.AddAttributeError(at.AtName("target"), "Missing target", "An ICMP monitor needs the host to ping.")
		}
		if m.Type.ValueString() == "DNS" && m.Domain.IsNull() {
			resp.Diagnostics.AddAttributeError(at.AtName("domain"), "Missing domain", "A DNS monitor needs the name to resolve.")
		}
		if m.Type.ValueString() == "ICMP" && !m.Domain.IsNull() {
			resp.Diagnostics.AddAttributeError(at.AtName("domain"), "Domain on an ICMP monitor", "Only DNS monitors resolve a domain.")
		}
		if m.LossThreshold.IsNull() && m.LatencyThreshold.IsNull() && m.JitterThreshold.IsNull() {
			resp.Diagnostics.AddAttributeError(at, "No threshold", "Set at least one of loss_threshold, latency_threshold or jitter_threshold, or the monitor never fails.")
		}
	}
}

func knownInt64(v types.Int64) bool { return !v.IsNull() && !v.IsUnknown() }

func positiveInt64(v types.Int64, at path.Path, diags *diag.Diagnostics) {
	if knownInt64(v) && v.ValueInt64() < 1 {
		diags.AddAttributeError(at, "Must be positive", fmt.Sprintf("Expected 1 or more; got %d.", v.ValueInt64()))
	}
}

func (r *wanSLAResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *wanSLAResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wanSLAModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sla := expandWANSLA(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.providerData.client.CreateWANSLA(ctx, plan.SiteID.ValueString(), sla)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create WAN SLA", err.Error())
		return
	}
	model := flattenWANSLA(plan.SiteID, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *wanSLAResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wanSLAModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sla, err := r.providerData.client.GetWANSLA(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read WAN SLA", err.Error())
		return
	}
	model := flattenWANSLA(state.SiteID, sla, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *wanSLAResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state wanSLAModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sla := expandWANSLA(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	sla.ID = state.ID.ValueString()
	updated, err := r.providerData.client.UpdateWANSLA(ctx, plan.SiteID.ValueString(), sla)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update WAN SLA", err.Error())
		return
	}
	model := flattenWANSLA(plan.SiteID, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *wanSLAResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wanSLAModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.providerData.client.DeleteWANSLA(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete WAN SLA", err.Error())
	}
}

func (r *wanSLAResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}

func expandWANSLA(ctx context.Context, plan wanSLAModel, diags *diag.Diagnostics) client.WANSLA {
	var monitors []wanSLAMonitorModel
	diags.Append(plan.Monitors.ElementsAs(ctx, &monitors, false)...)
	sla := client.WANSLA{Name: plan.Name.ValueString(), Monitors: make([]client.WANSLAMonitor, 0, len(monitors))}
	if !plan.MonitorHealthType.IsNull() && !plan.MonitorHealthType.IsUnknown() {
		sla.MonitorHealthType = plan.MonitorHealthType.ValueString()
	}
	for i, m := range monitors {
		sla.Monitors = append(sla.Monitors, client.WANSLAMonitor{
			MonitorID:  int64(i + 1),
			Type:       m.Type.ValueString(),
			Target:     m.Target.ValueString(),
			Domain:     m.Domain.ValueString(),
			Interval:   m.Interval.ValueInt64(),
			TimePeriod: m.TimePeriod.ValueInt64(),
			Alert: client.WANSLAAlert{
				LossThreshold:    m.LossThreshold.ValueInt64Pointer(),
				LatencyThreshold: m.LatencyThreshold.ValueInt64Pointer(),
				JitterThreshold:  m.JitterThreshold.ValueInt64Pointer(),
				ThresholdPolicy:  m.ThresholdPolicy.ValueString(),
			},
		})
	}
	return sla
}

func flattenWANSLA(siteID types.String, sla *client.WANSLA, diags *diag.Diagnostics) wanSLAModel {
	elements := make([]attr.Value, 0, len(sla.Monitors))
	for _, m := range sla.Monitors {
		value, d := types.ObjectValue(wanSLAMonitorAttrTypes, map[string]attr.Value{
			"type":              types.StringValue(m.Type),
			"target":            optionalString(m.Target),
			"domain":            optionalString(m.Domain),
			"interval":          types.Int64Value(m.Interval),
			"time_period":       types.Int64Value(m.TimePeriod),
			"loss_threshold":    types.Int64PointerValue(m.Alert.LossThreshold),
			"latency_threshold": types.Int64PointerValue(m.Alert.LatencyThreshold),
			"jitter_threshold":  types.Int64PointerValue(m.Alert.JitterThreshold),
			"threshold_policy":  types.StringValue(m.Alert.ThresholdPolicy),
		})
		diags.Append(d...)
		elements = append(elements, value)
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: wanSLAMonitorAttrTypes}, elements)
	diags.Append(d...)
	return wanSLAModel{
		ID:                types.StringValue(sla.ID),
		SiteID:            siteID,
		Name:              types.StringValue(sla.Name),
		MonitorHealthType: types.StringValue(sla.MonitorHealthType),
		Monitors:          list,
	}
}

// optionalString maps the controller's absent or empty string to null, so an
// unset optional attribute reads back unchanged.
func optionalString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
