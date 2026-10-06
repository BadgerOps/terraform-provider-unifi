package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPortForwardSourceValidator(t *testing.T) {
	for _, value := range []types.String{
		types.StringNull(), types.StringUnknown(), types.StringValue("any"),
		types.StringValue("198.51.100.1"), types.StringValue("198.51.100.0/24"),
		types.StringValue("0.0.0.0/0"), types.StringValue("2001:db8::1"), types.StringValue("2001:db8::/64"),
	} {
		var response validator.StringResponse
		portForwardSourceValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, &response)
		if response.Diagnostics.HasError() {
			t.Errorf("valid source %s rejected: %v", value, response.Diagnostics)
		}
	}
}
