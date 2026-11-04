package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

const (
	ntpSettingKey = "ntp"
	ntpMaxServers = 4
)

var (
	_ resource.Resource                   = (*ntpResource)(nil)
	_ resource.ResourceWithConfigure      = (*ntpResource)(nil)
	_ resource.ResourceWithImportState    = (*ntpResource)(nil)
	_ resource.ResourceWithValidateConfig = (*ntpResource)(nil)
)

type ntpResource struct{ providerData *providerData }

type ntpModel struct {
	ID      types.String `tfsdk:"id"`
	SiteID  types.String `tfsdk:"site_id"`
	Mode    types.String `tfsdk:"mode"`
	Servers types.List   `tfsdk:"servers"`
}

func NewNTPResource() resource.Resource { return &ntpResource{} }

func (r *ntpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting_ntp"
}

func (r *ntpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := siteSettingIDAttributes()
	attributes["mode"] = schema.StringAttribute{Required: true, Validators: []validator.String{stringOneOf{"auto", "manual"}}, MarkdownDescription: "`auto` uses Ubiquiti's NTP pool; `manual` uses `servers`."}
	attributes["servers"] = schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Up to four NTP servers (host names or IP addresses), in order. Required for `mode = \"manual\"`. With `auto` the controller keeps them but doesn't use them. Keeps the controller's value when not set."}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the NTP servers a site's UniFi devices use, the legacy local Network API setting `ntp`. The setting exists once per site: the resource takes over the existing document, or creates it on a site that has never saved it, and changes only the mode and the servers. Each write sends the document back whole, to preserve the fields it doesn't model, so a field that was absent can become empty but never absent again. Import with `<site_id>`. Destroying the resource only removes it from Terraform state.",
		Attributes:          attributes,
	}
}

func (r *ntpResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ntpModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Servers.IsUnknown() {
		return
	}
	var servers []types.String
	if !config.Servers.IsNull() {
		resp.Diagnostics.Append(config.Servers.ElementsAs(ctx, &servers, false)...)
	}
	if len(servers) > ntpMaxServers {
		resp.Diagnostics.AddAttributeError(path.Root("servers"), "Too many NTP servers", fmt.Sprintf("The controller stores at most %d servers; got %d.", ntpMaxServers, len(servers)))
	}
	for i, server := range servers {
		if !server.IsUnknown() && server.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(path.Root("servers").AtListIndex(i), "Empty NTP server", "Remove the entry instead of leaving it empty.")
		}
	}
	if config.Mode.ValueString() == "manual" && !config.Servers.IsNull() && len(servers) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("servers"), "Missing NTP servers", "mode = \"manual\" needs at least one server.")
	}
}

func (r *ntpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureResource(req, resp)
}

func (r *ntpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *ntpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *ntpResource) apply(ctx context.Context, planned tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var plan ntpModel
	diags.Append(planned.Get(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	siteID := plan.SiteID.ValueString()
	fields := map[string]any{"setting_preference": plan.Mode.ValueString()}
	if !plan.Servers.IsNull() && !plan.Servers.IsUnknown() {
		var servers []string
		diags.Append(plan.Servers.ElementsAs(ctx, &servers, false)...)
		if diags.HasError() {
			return
		}
		for i := 0; i < ntpMaxServers; i++ {
			value := ""
			if i < len(servers) {
				value = servers[i]
			}
			fields[fmt.Sprintf("ntp_server_%d", i+1)] = value
		}
	}
	// Before the write: manual mode without a server is rejected by the controller
	// with a less helpful message.
	if _, configured := fields["ntp_server_1"]; plan.Mode.ValueString() == "manual" && !configured {
		current := readSiteSetting(ctx, r.providerData, siteID, ntpSettingKey, diags)
		if diags.HasError() {
			return
		}
		if len(ntpServers(current, diags)) == 0 {
			diags.AddAttributeError(path.Root("servers"), "Missing NTP servers", "mode = \"manual\" needs at least one server, and the controller has none.")
			return
		}
	}
	setting := updateSiteSetting(ctx, r.providerData, siteID, ntpSettingKey, fields, diags)
	if setting == nil {
		return
	}
	model := flattenNTP(ctx, plan.SiteID, setting, diags)
	diags.Append(state.Set(ctx, &model)...)
}

func (r *ntpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ntpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setting := readSiteSetting(ctx, r.providerData, state.SiteID.ValueString(), ntpSettingKey, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if setting == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model := flattenNTP(ctx, state.SiteID, setting, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete leaves the NTP servers as they are: the setting can't be removed.
func (r *ntpResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func (r *ntpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importSiteSetting(ctx, req, resp)
}

// flattenNTP reads the servers in order and skips empty slots, which is how
// the UI stores fewer than four.
func flattenNTP(ctx context.Context, siteID types.String, setting client.SiteSetting, diags *diag.Diagnostics) ntpModel {
	list, d := types.ListValueFrom(ctx, types.StringType, ntpServers(setting, diags))
	diags.Append(d...)
	return ntpModel{ID: siteID, SiteID: siteID, Mode: types.StringValue(decodeSettingString(setting, "setting_preference", diags)), Servers: list}
}

func ntpServers(setting client.SiteSetting, diags *diag.Diagnostics) []string {
	servers := []string{}
	for i := 1; i <= ntpMaxServers; i++ {
		if server := decodeSettingString(setting, fmt.Sprintf("ntp_server_%d", i), diags); server != "" {
			servers = append(servers, server)
		}
	}
	return servers
}
