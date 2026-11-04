package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

const deviceManagementSettingKey = "mgmt"

var (
	_ resource.Resource                   = (*deviceManagementResource)(nil)
	_ resource.ResourceWithConfigure      = (*deviceManagementResource)(nil)
	_ resource.ResourceWithImportState    = (*deviceManagementResource)(nil)
	_ resource.ResourceWithValidateConfig = (*deviceManagementResource)(nil)
)

type deviceManagementResource struct{ providerData *providerData }

type deviceManagementModel struct {
	ID                     types.String `tfsdk:"id"`
	SiteID                 types.String `tfsdk:"site_id"`
	SSHEnabled             types.Bool   `tfsdk:"ssh_enabled"`
	SSHPasswordAuthEnabled types.Bool   `tfsdk:"ssh_password_auth_enabled"`
	AutoUpgrade            types.Bool   `tfsdk:"auto_upgrade"`
	AutoUpgradeHour        types.Int64  `tfsdk:"auto_upgrade_hour"`
}

func NewDeviceManagementResource() resource.Resource { return &deviceManagementResource{} }

func (r *deviceManagementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting_device_management"
}

func (r *deviceManagementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := siteSettingIDAttributes()
	attributes["ssh_enabled"] = schema.BoolAttribute{Required: true, MarkdownDescription: "SSH access to the site's UniFi devices (device authentication)."}
	attributes["ssh_password_auth_enabled"] = optionalComputedBool("Allow SSH logins to the devices with the password; with `false`, only the configured SSH keys work.")
	attributes["auto_upgrade"] = optionalComputedBool("Update the devices' firmware automatically.")
	attributes["auto_upgrade_hour"] = schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, MarkdownDescription: "Hour of the day (0 to 23, in the site's time zone) for automatic firmware updates. Keeps the controller's value when not set."}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the toggles of a site's device management settings, the legacy local Network API setting `mgmt`: whether the UniFi devices accept SSH, whether they accept the SSH password, and automatic firmware updates. The SSH username, password, password hash and keys, and the controller's API and management keys, are deliberately not modelled: they never reach Terraform state, and every write sends them back unchanged, except that each write to `mgmt` makes the controller regenerate its `x_api_token`, even when no value changes, so every apply that changes this resource rotates that token. Set them in the UI. The setting exists once per site: the resource takes over the existing document, or creates it on a site that has never saved it, and changes only the configured fields. Each write sends the document back whole, to preserve the fields it doesn't model, so a field that was absent can become empty but never absent again. Import with `<site_id>`. Destroying the resource only removes it from Terraform state.",
		Attributes:          attributes,
	}
}

func (r *deviceManagementResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config deviceManagementModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.AutoUpgradeHour.IsNull() && !config.AutoUpgradeHour.IsUnknown() && (config.AutoUpgradeHour.ValueInt64() < 0 || config.AutoUpgradeHour.ValueInt64() > 23) {
		resp.Diagnostics.AddAttributeError(path.Root("auto_upgrade_hour"), "Invalid hour", fmt.Sprintf("auto_upgrade_hour must be between 0 and 23; got %d.", config.AutoUpgradeHour.ValueInt64()))
	}
}

func (r *deviceManagementResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *deviceManagementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *deviceManagementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *deviceManagementResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan deviceManagementModel
	diags.Append(planned.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	fields := map[string]any{"x_ssh_enabled": plan.SSHEnabled.ValueBool()}
	setKnownBool(fields, "x_ssh_auth_password_enabled", plan.SSHPasswordAuthEnabled)
	setKnownBool(fields, "auto_upgrade", plan.AutoUpgrade)
	if !plan.AutoUpgradeHour.IsNull() && !plan.AutoUpgradeHour.IsUnknown() {
		fields["auto_upgrade_hour"] = plan.AutoUpgradeHour.ValueInt64()
	}
	setting := updateSiteSetting(ctx, r.providerData, plan.SiteID.ValueString(), deviceManagementSettingKey, fields, diags)
	if setting == nil {
		return
	}
	model := flattenDeviceManagement(plan.SiteID, setting, diags)
	diags.Append(state.Set(ctx, &model)...)
}

func (r *deviceManagementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceManagementModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setting := readSiteSetting(ctx, r.providerData, state.SiteID.ValueString(), deviceManagementSettingKey, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if setting == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model := flattenDeviceManagement(state.SiteID, setting, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves device access as it is: the setting can't be removed, and
// switching SSH off on destroy could lock the operator out.
func (r *deviceManagementResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *deviceManagementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importSiteSetting(ctx, req, resp)
}

func flattenDeviceManagement(siteID types.String, setting client.SiteSetting, diags *diag.Diagnostics) deviceManagementModel {
	model := deviceManagementModel{
		ID:                     siteID,
		SiteID:                 siteID,
		SSHEnabled:             decodeSettingBool(setting, "x_ssh_enabled", diags),
		SSHPasswordAuthEnabled: decodeSettingBool(setting, "x_ssh_auth_password_enabled", diags),
		AutoUpgrade:            decodeSettingBool(setting, "auto_upgrade", diags),
		AutoUpgradeHour:        types.Int64Null(),
	}
	var hour int64
	found, err := setting.Decode("auto_upgrade_hour", &hour)
	if err != nil {
		diags.AddError("Unexpected setting value", err.Error())
	} else if found {
		model.AutoUpgradeHour = types.Int64Value(hour)
	}
	return model
}
