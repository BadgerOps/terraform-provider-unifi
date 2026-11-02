package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

const remoteSyslogSettingKey = "rsyslogd"

var (
	_ resource.Resource                   = (*remoteSyslogResource)(nil)
	_ resource.ResourceWithConfigure      = (*remoteSyslogResource)(nil)
	_ resource.ResourceWithImportState    = (*remoteSyslogResource)(nil)
	_ resource.ResourceWithValidateConfig = (*remoteSyslogResource)(nil)
)

type remoteSyslogResource struct{ providerData *providerData }

type remoteSyslogModel struct {
	ID                          types.String `tfsdk:"id"`
	SiteID                      types.String `tfsdk:"site_id"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	IP                          types.String `tfsdk:"ip"`
	Port                        types.Int64  `tfsdk:"port"`
	Contents                    types.Set    `tfsdk:"contents"`
	LogAllContents              types.Bool   `tfsdk:"log_all_contents"`
	ThisController              types.Bool   `tfsdk:"this_controller"`
	ThisControllerEncryptedOnly types.Bool   `tfsdk:"this_controller_encrypted_only"`
	NetconsoleEnabled           types.Bool   `tfsdk:"netconsole_enabled"`
}

func NewRemoteSyslogResource() resource.Resource { return &remoteSyslogResource{} }

func (r *remoteSyslogResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting_remote_syslog"
}

func (r *remoteSyslogResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := siteSettingIDAttributes()
	attributes["enabled"] = schema.BoolAttribute{Required: true, MarkdownDescription: "Send the site's syslog (gateway, switches, access points) to a remote server."}
	attributes["ip"] = schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "IPv4 address of the syslog server. Required when `enabled` is true, unless the controller already has one. Keeps the controller's value when not set."}
	attributes["port"] = schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, MarkdownDescription: "UDP port of the syslog server, 1 to 65535. Keeps the controller's value when not set."}
	attributes["contents"] = schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Log categories to send, as the controller names them, for example `device`, `client`, `firewall_default_policy`, `security_detections`, `admin_activity`, `critical`, `triggers`, `updates` and `vpn`. Keeps the controller's value when not set."}
	attributes["log_all_contents"] = optionalComputedBool("Send every category (the UI's \"all\" option) instead of only `contents`.")
	attributes["this_controller"] = optionalComputedBool("Also send the logs to the console itself (controller field `this_controller`).")
	attributes["this_controller_encrypted_only"] = optionalComputedBool("Send logs to the console itself only over an encrypted connection (controller field `this_controller_encrypted_only`).")
	attributes["netconsole_enabled"] = optionalComputedBool("Send the devices' kernel netconsole output too.")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a site's remote syslog server, the legacy local Network API setting `rsyslogd`. The setting exists once per site: the resource takes over the existing document, or creates it on a site that has never saved it, and changes only the fields it models. Each write sends the document back whole, to preserve the fields it doesn't model, so a field that was absent can become empty but never absent again. Import with `<site_id>`. Destroying the resource only removes it from Terraform state; logging stays as it is.",
		Attributes:          attributes,
	}
}

func (r *remoteSyslogResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config remoteSyslogModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.IP.IsNull() && !config.IP.IsUnknown() {
		if !isIPv4Address(config.IP.ValueString()) {
			resp.Diagnostics.AddAttributeError(path.Root("ip"), "Invalid syslog server", fmt.Sprintf("ip needs an IPv4 address; got %q.", config.IP.ValueString()))
		}
	}
	if !config.Port.IsNull() && !config.Port.IsUnknown() && (config.Port.ValueInt64() < 1 || config.Port.ValueInt64() > 65535) {
		resp.Diagnostics.AddAttributeError(path.Root("port"), "Invalid syslog port", fmt.Sprintf("port must be between 1 and 65535; got %d.", config.Port.ValueInt64()))
	}
}

func (r *remoteSyslogResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *remoteSyslogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *remoteSyslogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *remoteSyslogResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan remoteSyslogModel
	diags.Append(planned.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	siteID := plan.SiteID.ValueString()
	current := readSiteSetting(ctx, r.providerData, siteID, remoteSyslogSettingKey, diags)
	if diags.HasError() {
		return
	}
	fields := map[string]any{"enabled": plan.Enabled.ValueBool()}
	if !plan.IP.IsNull() && !plan.IP.IsUnknown() {
		fields["ip"] = plan.IP.ValueString()
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		fields["port"] = remoteSyslogPortValue(current, plan.Port.ValueInt64())
	}
	if !plan.Contents.IsNull() && !plan.Contents.IsUnknown() {
		var contents []string
		diags.Append(plan.Contents.ElementsAs(ctx, &contents, false)...)
		sort.Strings(contents)
		fields["contents"] = contents
	}
	setKnownBool(fields, "log_all_contents", plan.LogAllContents)
	setKnownBool(fields, "this_controller", plan.ThisController)
	setKnownBool(fields, "this_controller_encrypted_only", plan.ThisControllerEncryptedOnly)
	setKnownBool(fields, "netconsole_enabled", plan.NetconsoleEnabled)
	if diags.HasError() {
		return
	}
	if plan.Enabled.ValueBool() {
		ip, _ := fields["ip"].(string)
		if ip == "" {
			ip = decodeSettingString(current, "ip", diags)
		}
		if ip == "" {
			diags.AddAttributeError(path.Root("ip"), "Missing syslog server", "enabled = true needs ip: the controller has no syslog server yet.")
			return
		}
	}
	setting := updateSiteSetting(ctx, r.providerData, siteID, remoteSyslogSettingKey, fields, diags)
	if setting == nil {
		return
	}
	model := flattenRemoteSyslog(ctx, plan.SiteID, setting, diags)
	diags.Append(state.Set(ctx, &model)...)
}

// remoteSyslogPortValue sends the port in the controller's own JSON type:
// current Network versions store it as a string ("514").
func remoteSyslogPortValue(current client.SiteSetting, port int64) any {
	if raw, ok := current["port"]; ok && len(raw) > 0 && raw[0] != '"' {
		return port
	}
	return strconv.FormatInt(port, 10)
}

func (r *remoteSyslogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state remoteSyslogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setting := readSiteSetting(ctx, r.providerData, state.SiteID.ValueString(), remoteSyslogSettingKey, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if setting == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model := flattenRemoteSyslog(ctx, state.SiteID, setting, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves logging as it is: the setting can't be removed, and turning
// remote syslog off on destroy would silently stop log collection.
func (r *remoteSyslogResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *remoteSyslogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importSiteSetting(ctx, req, resp)
}

func flattenRemoteSyslog(ctx context.Context, siteID types.String, setting client.SiteSetting, diags *diag.Diagnostics) remoteSyslogModel {
	model := remoteSyslogModel{
		ID:                          siteID,
		SiteID:                      siteID,
		Enabled:                     decodeSettingBool(setting, "enabled", diags),
		IP:                          types.StringValue(decodeSettingString(setting, "ip", diags)),
		Port:                        types.Int64Null(),
		LogAllContents:              decodeSettingBool(setting, "log_all_contents", diags),
		ThisController:              decodeSettingBool(setting, "this_controller", diags),
		ThisControllerEncryptedOnly: decodeSettingBool(setting, "this_controller_encrypted_only", diags),
		NetconsoleEnabled:           decodeSettingBool(setting, "netconsole_enabled", diags),
	}
	if raw, ok := setting["port"]; ok && string(raw) != "null" {
		port, err := parseRemoteSyslogPort(raw)
		if err != nil {
			diags.AddError("Unexpected setting value", err.Error())
		} else {
			model.Port = types.Int64Value(port)
		}
	}
	var contents []string
	if _, err := setting.Decode("contents", &contents); err != nil {
		diags.AddError("Unexpected setting value", err.Error())
	}
	if contents == nil {
		contents = []string{}
	}
	set, d := types.SetValueFrom(ctx, types.StringType, contents)
	diags.Append(d...)
	model.Contents = set
	return model
}

// parseRemoteSyslogPort accepts the port as a string ("514") or a number.
func parseRemoteSyslogPort(raw json.RawMessage) (int64, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		port, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("decode setting field port: %w", err)
		}
		return port, nil
	}
	var number int64
	if err := json.Unmarshal(raw, &number); err != nil {
		return 0, fmt.Errorf("decode setting field port: %w", err)
	}
	return number, nil
}
