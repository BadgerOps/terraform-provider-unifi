package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

// The site settings (`rsyslogd`, `ntp`, `snmp`) exist at most once per site
// and can't be deleted: these resources take over fields of the existing
// document, or create it on a site that has never saved the setting. Fields
// left out of the configuration keep the controller's value, so an import
// followed by a configuration that names only some fields shows no diff.

func siteSettingIDAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Synthetic resource ID equal to `site_id`."},
		"site_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, MarkdownDescription: "Integration site UUID."},
	}
}

func optionalComputedBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}, MarkdownDescription: description + " Keeps the controller's value when not set."}
}

// setKnownBool adds a field to a setting update only when the plan knows it:
// an unset optional attribute is unknown on create and import, and must not
// overwrite the controller's value with a zero.
func setKnownBool(fields map[string]any, field string, value types.Bool) {
	if !value.IsNull() && !value.IsUnknown() {
		fields[field] = value.ValueBool()
	}
}

func decodeSettingBool(setting client.SiteSetting, field string, diags *diag.Diagnostics) types.Bool {
	var value bool
	found, err := setting.Decode(field, &value)
	if err != nil {
		diags.AddError("Unexpected setting value", err.Error())
		return types.BoolNull()
	}
	if !found {
		return types.BoolValue(false)
	}
	return types.BoolValue(value)
}

func decodeSettingString(setting client.SiteSetting, field string, diags *diag.Diagnostics) string {
	var value string
	if _, err := setting.Decode(field, &value); err != nil {
		diags.AddError("Unexpected setting value", err.Error())
	}
	return value
}

// readSiteSetting returns nil without an error when the setting is gone, so
// Read can drop the resource from state.
func readSiteSetting(ctx context.Context, data *providerData, siteID, key string, diags *diag.Diagnostics) client.SiteSetting {
	setting, err := data.client.GetSiteSetting(ctx, siteID, key)
	if err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		diags.AddError(fmt.Sprintf("Unable to read setting %s", key), err.Error())
		return nil
	}
	return setting
}

func updateSiteSetting(ctx context.Context, data *providerData, siteID, key string, fields map[string]any, diags *diag.Diagnostics) client.SiteSetting {
	setting, err := data.client.UpdateSiteSetting(ctx, siteID, key, fields)
	if err != nil {
		diags.AddError(fmt.Sprintf("Unable to update setting %s", key), err.Error())
		return nil
	}
	return setting
}

func importSiteSetting(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import identifier", "Import with the site ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
