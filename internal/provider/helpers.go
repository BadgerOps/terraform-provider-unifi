package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func configureResource(request resource.ConfigureRequest, response *resource.ConfigureResponse) *providerData {
	if request.ProviderData == nil {
		return nil
	}

	providerData, ok := request.ProviderData.(*providerData)
	if !ok {
		response.Diagnostics.AddError("Unexpected provider data type", "The provider data could not be converted to the expected client configuration.")
		return nil
	}

	return providerData
}

func parseCompositeImportID(raw string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import ID in the form <site_id>/<resource_id>")
	}

	return parts[0], parts[1], nil
}

func importCompositeID(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	siteID, resourceID, err := parseCompositeImportID(request.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid import identifier", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("site_id"), siteID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("id"), resourceID)...)
}

func stringSetValue(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetNull(types.StringType), nil
	}

	return types.SetValueFrom(ctx, types.StringType, values)
}

func float64SetValue(ctx context.Context, values []float64) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetNull(types.Float64Type), nil
	}

	return types.SetValueFrom(ctx, types.Float64Type, values)
}

func int64SetValue(ctx context.Context, values []int64) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetNull(types.Int64Type), nil
	}

	return types.SetValueFrom(ctx, types.Int64Type, values)
}

func setToStrings(ctx context.Context, value types.Set, path string, diags *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var values []string
	diags.Append(value.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		diags.AddError("Invalid set value", fmt.Sprintf("Unable to decode `%s` into a string slice.", path))
	}

	return values
}

func setToFloat64s(ctx context.Context, value types.Set, path string, diags *diag.Diagnostics) []float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var values []float64
	diags.Append(value.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		diags.AddError("Invalid set value", fmt.Sprintf("Unable to decode `%s` into a float set.", path))
	}

	return values
}

func setToInt64s(ctx context.Context, value types.Set, path string, diags *diag.Diagnostics) []int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var values []int64
	diags.Append(value.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		diags.AddError("Invalid set value", fmt.Sprintf("Unable to decode `%s` into an integer set.", path))
	}

	return values
}

func stringPointerValue(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	contents := value.ValueString()
	return &contents
}

func boolPointerValue(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	contents := value.ValueBool()
	return &contents
}

func int64PointerValue(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	contents := value.ValueInt64()
	return &contents
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func nullableBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func nullableInt64(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}
