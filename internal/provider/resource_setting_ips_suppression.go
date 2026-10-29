package provider

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                   = (*ipsSuppressionResource)(nil)
	_ resource.ResourceWithConfigure      = (*ipsSuppressionResource)(nil)
	_ resource.ResourceWithImportState    = (*ipsSuppressionResource)(nil)
	_ resource.ResourceWithValidateConfig = (*ipsSuppressionResource)(nil)
)

type ipsSuppressionResource struct{ providerData *providerData }

type ipsSuppressionModel struct {
	ID     types.String               `tfsdk:"id"`
	SiteID types.String               `tfsdk:"site_id"`
	Alerts []ipsSuppressionAlertModel `tfsdk:"alerts"`
}

type ipsSuppressionAlertModel struct {
	SignatureID types.Int64                   `tfsdk:"signature_id"`
	GID         types.Int64                   `tfsdk:"gid"`
	Category    types.String                  `tfsdk:"category"`
	Signature   types.String                  `tfsdk:"signature"`
	Type        types.String                  `tfsdk:"type"`
	Tracking    []ipsSuppressionTrackingModel `tfsdk:"tracking"`
}

type ipsSuppressionTrackingModel struct {
	Mode      types.String `tfsdk:"mode"`
	Value     types.String `tfsdk:"value"`
	Direction types.String `tfsdk:"direction"`
}

func NewIPSSuppressionResource() resource.Resource { return &ipsSuppressionResource{} }

func (r *ipsSuppressionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting_ips_suppression"
}

func (r *ipsSuppressionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the UniFi IPS (intrusion prevention) alert suppressions of a site, the list shown under CyberSecure > Suppressions, using the legacy local Network API setting `ips_suppression`. The resource is authoritative for the alert list: suppressions not in the configuration are removed on apply. The suppression whitelist and other setting fields are left as they are. There is one setting per site; import with `<site_id>`. Destroying the resource only removes it from Terraform state and leaves the suppressions on the controller.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Synthetic resource ID equal to `site_id`."},
			"site_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Integration site UUID."},
			"alerts": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Suppressed alerts, in controller order. Use an empty list to remove every suppression.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"signature_id": schema.Int64Attribute{Required: true, MarkdownDescription: "Signature ID (SID) of the suppressed rule, for example `2016642`."},
					"gid":          schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1), MarkdownDescription: "Generator ID of the rule. Defaults to `1`, the ID of standard rule signatures."},
					"category":     schema.StringAttribute{Required: true, MarkdownDescription: "Category of the signature as the controller names it, for example `emerging-scan` or `emerging-webserver`, not the ruleset file name (`emerging-web_server`). The controller rejects a name it does not know on apply with `api.err.UnsupportedIpsCategories`; the category does not have to be enabled on the site, and suppressions can be written while IPS is disabled."},
					"signature":    schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Signature description shown in the UI, for example `ET SCAN Potential SSH Scan OUTBOUND`. Kept from the controller when not set."},
					"type":         schema.StringAttribute{Required: true, Validators: []validator.String{stringOneOf{"all", "track"}}, MarkdownDescription: "`all` suppresses the signature for every address; `track` only for the `tracking` entries."},
					"tracking": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Addresses the suppression applies to. Required for `type = \"track\"`, not allowed for `type = \"all\"`.",
						NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
							"mode":      schema.StringAttribute{Required: true, Validators: []validator.String{stringOneOf{"ip", "subnet"}}, MarkdownDescription: "`ip` for one IPv4 address, `subnet` for an IPv4 CIDR prefix."},
							"value":     schema.StringAttribute{Required: true, MarkdownDescription: "The IPv4 address or CIDR prefix."},
							"direction": schema.StringAttribute{Required: true, Validators: []validator.String{stringOneOf{"src", "dest", "both"}}, MarkdownDescription: "Match the address as the source (`src`), the destination (`dest`), or either (`both`)."},
						}},
					},
				}},
			},
		},
	}
}

func (r *ipsSuppressionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ipsSuppressionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateIPSSuppressionAlerts(config.Alerts)...)
}

func validateIPSSuppressionAlerts(alerts []ipsSuppressionAlertModel) diag.Diagnostics {
	var diags diag.Diagnostics
	seen := map[[2]int64]int{}
	for i, alert := range alerts {
		alertPath := path.Root("alerts").AtListIndex(i)
		if !alert.SignatureID.IsNull() && !alert.SignatureID.IsUnknown() && !alert.GID.IsUnknown() {
			gid := int64(1)
			if !alert.GID.IsNull() {
				gid = alert.GID.ValueInt64()
			}
			key := [2]int64{gid, alert.SignatureID.ValueInt64()}
			if first, ok := seen[key]; ok {
				diags.AddAttributeError(alertPath.AtName("signature_id"), "Duplicate suppression", fmt.Sprintf("Signature %d:%d is already suppressed by alerts[%d]; merge the tracking entries into one alert.", key[0], key[1], first))
			}
			seen[key] = i
		}
		if alert.Type.IsUnknown() {
			continue
		}
		switch alert.Type.ValueString() {
		case "track":
			if len(alert.Tracking) == 0 {
				diags.AddAttributeError(alertPath.AtName("tracking"), "Missing tracking", "type = \"track\" needs at least one tracking entry.")
			}
		case "all":
			if len(alert.Tracking) > 0 {
				diags.AddAttributeError(alertPath.AtName("tracking"), "Unexpected tracking", "type = \"all\" suppresses the signature everywhere; remove tracking or use type = \"track\".")
			}
		}
		for j, entry := range alert.Tracking {
			if entry.Mode.IsUnknown() || entry.Value.IsUnknown() {
				continue
			}
			if !validIPSSuppressionTrackingValue(entry.Mode.ValueString(), entry.Value.ValueString()) {
				diags.AddAttributeError(alertPath.AtName("tracking").AtListIndex(j).AtName("value"), "Invalid tracking value", fmt.Sprintf("mode %q needs %s; got %q.", entry.Mode.ValueString(), map[string]string{"ip": "an IPv4 address", "subnet": "an IPv4 CIDR prefix with no host bits set"}[entry.Mode.ValueString()], entry.Value.ValueString()))
			}
		}
	}
	return diags
}

func validIPSSuppressionTrackingValue(mode, value string) bool {
	switch mode {
	case "ip":
		address, err := netip.ParseAddr(value)
		return err == nil && address.Is4()
	case "subnet":
		prefix, err := netip.ParsePrefix(value)
		return err == nil && prefix.Addr().Is4() && prefix == prefix.Masked()
	}
	return true // the mode validator reports unknown modes
}

func (r *ipsSuppressionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *ipsSuppressionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *ipsSuppressionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *ipsSuppressionResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan ipsSuppressionModel
	diags.Append(planned.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	siteID := plan.SiteID.ValueString()
	current, err := r.providerData.client.GetIPSSuppression(ctx, siteID)
	if err != nil {
		diags.AddError("Unable to read IPS suppression", err.Error())
		return
	}
	setting, err := r.providerData.client.SetIPSSuppression(ctx, siteID, expandIPSSuppressionAlerts(plan.Alerts, current.Alerts))
	if err != nil {
		diags.AddError("Unable to update IPS suppression", err.Error())
		return
	}
	model := flattenIPSSuppression(plan.SiteID, setting)
	diags.Append(state.Set(ctx, &model)...)
}

func (r *ipsSuppressionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipsSuppressionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setting, err := r.providerData.client.GetIPSSuppression(ctx, state.SiteID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read IPS suppression", err.Error())
		return
	}
	model := flattenIPSSuppression(state.SiteID, setting)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves the controller's suppressions in place: the setting always
// exists, and silently removing every suppression on destroy would re-enable
// blocking for signatures someone deliberately silenced.
func (r *ipsSuppressionResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *ipsSuppressionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import identifier", "Import with the site ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// expandIPSSuppressionAlerts keeps the controller's signature description for
// alerts whose configuration leaves it unset.
func expandIPSSuppressionAlerts(alerts []ipsSuppressionAlertModel, current []client.IPSSuppressionAlert) []client.IPSSuppressionAlert {
	descriptions := map[[2]int64]string{}
	for _, alert := range current {
		descriptions[[2]int64{alert.GID, alert.ID}] = alert.Signature
	}
	out := make([]client.IPSSuppressionAlert, 0, len(alerts))
	for _, alert := range alerts {
		gid := alert.GID.ValueInt64()
		if alert.GID.IsNull() || alert.GID.IsUnknown() {
			gid = 1
		}
		signature := alert.Signature.ValueString()
		if alert.Signature.IsNull() || alert.Signature.IsUnknown() {
			signature = descriptions[[2]int64{gid, alert.SignatureID.ValueInt64()}]
		}
		tracking := make([]client.IPSSuppressionTracking, 0, len(alert.Tracking))
		for _, entry := range alert.Tracking {
			tracking = append(tracking, client.IPSSuppressionTracking{Mode: entry.Mode.ValueString(), Value: entry.Value.ValueString(), Direction: entry.Direction.ValueString()})
		}
		out = append(out, client.IPSSuppressionAlert{GID: gid, ID: alert.SignatureID.ValueInt64(), Category: alert.Category.ValueString(), Signature: signature, Type: alert.Type.ValueString(), Tracking: tracking})
	}
	return out
}

func flattenIPSSuppression(siteID types.String, setting *client.IPSSuppression) ipsSuppressionModel {
	alerts := make([]ipsSuppressionAlertModel, 0, len(setting.Alerts))
	for _, alert := range setting.Alerts {
		var tracking []ipsSuppressionTrackingModel
		for _, entry := range alert.Tracking {
			tracking = append(tracking, ipsSuppressionTrackingModel{Mode: types.StringValue(entry.Mode), Value: types.StringValue(entry.Value), Direction: types.StringValue(entry.Direction)})
		}
		alerts = append(alerts, ipsSuppressionAlertModel{SignatureID: types.Int64Value(alert.ID), GID: types.Int64Value(alert.GID), Category: types.StringValue(alert.Category), Signature: types.StringValue(alert.Signature), Type: types.StringValue(alert.Type), Tracking: tracking})
	}
	return ipsSuppressionModel{ID: siteID, SiteID: siteID, Alerts: alerts}
}
