package provider

import (
	"context"
	"math"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCanonicalizeWifiFrequency(t *testing.T) {
	t.Parallel()

	exact24 := 2.4
	drifted24 := math.Nextafter(2.4, 0)
	if drifted24 == exact24 {
		t.Fatal("expected Nextafter to produce a distinct float64 near 2.4")
	}

	testCases := []struct {
		name   string
		input  float64
		want   float64
		wantOK bool
	}{
		{name: "exact 2.4", input: exact24, want: 2.4, wantOK: true},
		{name: "drifted 2.4", input: drifted24, want: 2.4, wantOK: true},
		{name: "exact 5", input: 5, want: 5, wantOK: true},
		{name: "exact 6", input: 6, want: 6, wantOK: true},
		{name: "unknown", input: 3.5, want: 3.5, wantOK: false},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := canonicalizeWifiFrequency(testCase.input)
			if ok != testCase.wantOK {
				t.Fatalf("canonicalizeWifiFrequency(%v) ok = %v, want %v", testCase.input, ok, testCase.wantOK)
			}
			if got != testCase.want {
				t.Fatalf("canonicalizeWifiFrequency(%v) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestWifiFrequenciesSemanticallyEqual(t *testing.T) {
	t.Parallel()

	drifted24 := math.Nextafter(2.4, math.MaxFloat64)
	if !wifiFrequenciesSemanticallyEqual([]float64{drifted24, 5}, []float64{5, 2.4}) {
		t.Fatalf("expected drifted 2.4 + 5 to equal canonical [5, 2.4] regardless of order")
	}
	if wifiFrequenciesSemanticallyEqual([]float64{2.4}, []float64{2.4, 5}) {
		t.Fatalf("different band sets must not be semantically equal")
	}
	if !wifiFrequenciesSemanticallyEqual(nil, nil) {
		t.Fatalf("nil frequency sets should be equal")
	}
}

func TestWifiFrequenciesPlanModifierKeepsStateOnDriftedBits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drifted := math.Nextafter(2.4, math.MaxFloat64)
	configSet := types.SetValueMust(types.Float64Type, []attr.Value{
		types.Float64Value(2.4),
		types.Float64Value(5),
	})
	stateSet := types.SetValueMust(types.Float64Type, []attr.Value{
		types.Float64Value(drifted),
		types.Float64Value(5),
	})

	var response planmodifier.SetResponse
	response.PlanValue = configSet
	wifiFrequenciesSuppressDrift().PlanModifySet(ctx, planmodifier.SetRequest{
		ConfigValue: configSet,
		StateValue:  stateSet,
		PlanValue:   configSet,
	}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("PlanModifySet diagnostics: %v", response.Diagnostics)
	}
	if !response.PlanValue.Equal(stateSet) {
		t.Fatalf("PlanModifySet() plan = %v, want state %v", response.PlanValue, stateSet)
	}
}
