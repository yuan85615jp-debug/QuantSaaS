package quant

import "math"

// SigmoidParams are the dynamic-balance knobs (often part of the genome).
type SigmoidParams struct {
	Beta                 float64
	Gamma                float64
	MarketBetaMultiplier float64
}

func DefaultSigmoidParams() SigmoidParams {
	return SigmoidParams{Beta: 1.0, Gamma: 0.5, MarketBetaMultiplier: 1.0}
}

// TargetWeight computes Sigmoid dynamic-balance target weight in [0,1].
// Signal > 0 → tendency to reduce weight; Signal < 0 → increase weight.
func TargetWeight(currentWeight, signal float64, p SigmoidParams) float64 {
	cw := Clamp(currentWeight, 0, 1)
	effBeta := p.Beta * p.MarketBetaMultiplier
	if effBeta < 0.01 {
		effBeta = 0.01
	}
	bias := cw - 0.5
	exp := effBeta*signal + p.Gamma*bias
	if exp > 60 {
		return 0
	}
	if exp < -60 {
		return 1
	}
	tw := 1.0 / (1.0 + math.Exp(exp))
	return Clamp(tw, 0, 1)
}

func TheoreticalCNY(targetWeight, currentWeight, totalEquity float64) float64 {
	return (targetWeight - currentWeight) * totalEquity
}

// PDESignal builds a linear signal from dimensionless pos/vel/acc features.
func PDESignal(pos, vel, acc, kp, kv, ka float64) float64 {
	return kp*pos + kv*vel + ka*acc
}
