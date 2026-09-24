package quant

import (
	"math"
	"testing"
)

func TestTargetWeightNeutral(t *testing.T) {
	p := SigmoidParams{Beta: 1, Gamma: 0, MarketBetaMultiplier: 1}
	tw := TargetWeight(0.5, 0, p)
	if math.Abs(tw-0.5) > 1e-9 {
		t.Fatalf("got %v", tw)
	}
}

func TestTargetWeightBearishSignal(t *testing.T) {
	p := DefaultSigmoidParams()
	tw := TargetWeight(0.5, 2.0, p)
	if tw >= 0.5 {
		t.Fatalf("expected reduce weight, got %v", tw)
	}
}

func TestTheoreticalCNY(t *testing.T) {
	v := TheoreticalCNY(0.6, 0.4, 100_000)
	if math.Abs(v-20_000) > 1e-9 {
		t.Fatalf("got %v", v)
	}
}
