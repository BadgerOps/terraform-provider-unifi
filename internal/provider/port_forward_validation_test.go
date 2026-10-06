package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runStringValidator(t *testing.T, v validator.String, valid, invalid []types.String) {
	t.Helper()
	for _, value := range valid {
		var response validator.StringResponse
		v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, &response)
		if response.Diagnostics.HasError() {
			t.Errorf("valid value %s rejected: %v", value, response.Diagnostics)
		}
	}
	for _, value := range invalid {
		var response validator.StringResponse
		v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, &response)
		if !response.Diagnostics.HasError() {
			t.Errorf("invalid value %s accepted", value)
		}
	}
}

func TestPortForwardSourceValidator(t *testing.T) {
	runStringValidator(t, portForwardSourceValidator{},
		[]types.String{
			types.StringNull(), types.StringUnknown(), types.StringValue("any"),
			types.StringValue("198.51.100.1"), types.StringValue("198.51.100.0/24"), types.StringValue("0.0.0.0/0"),
		},
		[]types.String{
			// The legacy port-forward src field is IPv4 only, and an unmasked
			// prefix is normalised by the controller into a perpetual diff.
			types.StringValue("2001:db8::1"), types.StringValue("2001:db8::/64"),
			types.StringValue("198.51.100.7/24"), types.StringValue("Any"), types.StringValue(""), types.StringValue("not-an-ip"),
		})
}

func TestPortForwardPortsValidator(t *testing.T) {
	runStringValidator(t, portForwardPortsValidator{},
		[]types.String{
			types.StringNull(), types.StringUnknown(),
			types.StringValue("3074"), types.StringValue("3000-3010"), types.StringValue("80,443,8080"), types.StringValue("1,1000-2000,65535"),
		},
		[]types.String{
			types.StringValue(""), types.StringValue("0"), types.StringValue("65536"), types.StringValue("80;443"),
			types.StringValue("3010-3000"), types.StringValue("80, 443"), types.StringValue("80,"), types.StringValue("http"), types.StringValue("1-2-3"),
		})
}

func TestPortForwardIPv4Validator(t *testing.T) {
	runStringValidator(t, ipv4AddressValidator{},
		[]types.String{types.StringNull(), types.StringUnknown(), types.StringValue("192.0.2.90")},
		[]types.String{types.StringValue(""), types.StringValue("example.lan"), types.StringValue("2001:db8::1"), types.StringValue("192.0.2.0/24")})
}
