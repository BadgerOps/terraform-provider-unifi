package provider

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// knownWifiFrequenciesGHz are the UniFi Integration API enum values for
// broadcastingFrequenciesGHz (2.4, 5, and 6 GHz).
var knownWifiFrequenciesGHz = []float64{2.4, 5, 6}

const wifiFrequencyMatchEpsilon = 0.05

// canonicalizeWifiFrequency maps a float to the nearest known WiFi band when
// it is within epsilon of one. Controllers and Terraform/OpenTofu can disagree
// on the exact float64 bits for 2.4, which otherwise makes set equality fail
// and produces perpetual no-op plans (#45).
func canonicalizeWifiFrequency(value float64) (float64, bool) {
	for _, known := range knownWifiFrequenciesGHz {
		if math.Abs(value-known) < wifiFrequencyMatchEpsilon {
			return known, true
		}
	}
	return value, false
}

func validateWifiFrequencies(ctx context.Context, value types.Set) error {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var frequencies []float64
	diags := value.ElementsAs(ctx, &frequencies, false)
	if diags.HasError() {
		return fmt.Errorf("unable to decode broadcasting_frequencies_ghz")
	}
	if len(frequencies) == 0 {
		return fmt.Errorf("broadcasting_frequencies_ghz must contain at least one band")
	}
	for _, frequency := range frequencies {
		if _, ok := canonicalizeWifiFrequency(frequency); !ok {
			return fmt.Errorf("broadcasting_frequencies_ghz values must be 2.4, 5, or 6")
		}
	}
	return nil
}

func canonicalizeWifiFrequencies(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]float64, len(values))
	for i, value := range values {
		if canonical, ok := canonicalizeWifiFrequency(value); ok {
			out[i] = canonical
		} else {
			out[i] = value
		}
	}
	return out
}

func wifiFrequenciesSemanticallyEqual(left, right []float64) bool {
	leftCanonical := canonicalizeWifiFrequencies(left)
	rightCanonical := canonicalizeWifiFrequencies(right)
	if len(leftCanonical) != len(rightCanonical) {
		return false
	}
	sort.Float64s(leftCanonical)
	sort.Float64s(rightCanonical)
	for i := range leftCanonical {
		if leftCanonical[i] != rightCanonical[i] {
			return false
		}
	}
	return true
}

// wifiFrequenciesSuppressDrift keeps state for broadcasting_frequencies_ghz
// when config and state name the same bands, even if float64 bit patterns differ.
func wifiFrequenciesSuppressDrift() planmodifier.Set {
	return wifiFrequenciesPlanModifier{}
}

type wifiFrequenciesPlanModifier struct{}

func (m wifiFrequenciesPlanModifier) Description(_ context.Context) string {
	return "Suppresses drift when broadcasting_frequencies_ghz lists the same WiFi bands with different float64 representations of 2.4 GHz."
}

func (m wifiFrequenciesPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m wifiFrequenciesPlanModifier) PlanModifySet(ctx context.Context, request planmodifier.SetRequest, response *planmodifier.SetResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() || request.StateValue.IsNull() || request.StateValue.IsUnknown() {
		return
	}

	var configValues []float64
	var stateValues []float64
	diags := request.ConfigValue.ElementsAs(ctx, &configValues, false)
	response.Diagnostics.Append(diags...)
	diags = request.StateValue.ElementsAs(ctx, &stateValues, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if wifiFrequenciesSemanticallyEqual(configValues, stateValues) {
		response.PlanValue = request.StateValue
	}
}

func wifiFrequencySetValue(ctx context.Context, values []float64) (types.Set, diag.Diagnostics) {
	return float64SetValue(ctx, canonicalizeWifiFrequencies(values))
}
