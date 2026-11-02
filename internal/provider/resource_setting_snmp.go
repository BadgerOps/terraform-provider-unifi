package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

const snmpSettingKey = "snmp"

var (
	_ resource.Resource                = (*snmpResource)(nil)
	_ resource.ResourceWithConfigure   = (*snmpResource)(nil)
	_ resource.ResourceWithImportState = (*snmpResource)(nil)
)

type snmpResource struct{ providerData *providerData }

type snmpModel struct {
	ID        types.String `tfsdk:"id"`
	SiteID    types.String `tfsdk:"site_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	EnabledV3 types.Bool   `tfsdk:"enabled_v3"`
}

func NewSNMPResource() resource.Resource { return &snmpResource{} }

func (r *snmpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting_snmp"
}

func (r *snmpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := siteSettingIDAttributes()
	attributes["enabled"] = schema.BoolAttribute{Required: true, MarkdownDescription: "SNMP v1/v2c on the site's devices."}
	attributes["enabled_v3"] = optionalComputedBool("SNMP v3 on the site's devices.")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Turn SNMP on a site's UniFi devices on or off, the legacy local Network API setting `snmp`. Only the two toggles are managed: the community string and the v3 credentials are secrets and stay as they were set in the UI, so they never reach Terraform state. The setting exists once per site: the resource takes over the existing document, or creates it on a site that has never saved it. Each write sends the document back whole, to preserve the fields it doesn't model, so a field that was absent can become empty but never absent again. Import with `<site_id>`. Destroying the resource only removes it from Terraform state.",
		Attributes:          attributes,
	}
}

func (r *snmpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *snmpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *snmpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *snmpResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan snmpModel
	diags.Append(planned.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	fields := map[string]any{"enabled": plan.Enabled.ValueBool()}
	setKnownBool(fields, "enabledV3", plan.EnabledV3)
	setting := updateSiteSetting(ctx, r.providerData, plan.SiteID.ValueString(), snmpSettingKey, fields, diags)
	if setting == nil {
		return
	}
	model := flattenSNMP(plan.SiteID, setting, diags)
	diags.Append(state.Set(ctx, &model)...)
}

func (r *snmpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state snmpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setting := readSiteSetting(ctx, r.providerData, state.SiteID.ValueString(), snmpSettingKey, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if setting == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model := flattenSNMP(state.SiteID, setting, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves SNMP as it is: the setting can't be removed.
func (r *snmpResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *snmpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importSiteSetting(ctx, req, resp)
}

func flattenSNMP(siteID types.String, setting client.SiteSetting, diags *diag.Diagnostics) snmpModel {
	return snmpModel{ID: siteID, SiteID: siteID, Enabled: decodeSettingBool(setting, "enabled", diags), EnabledV3: decodeSettingBool(setting, "enabledV3", diags)}
}
