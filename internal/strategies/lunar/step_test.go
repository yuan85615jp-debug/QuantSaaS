package lunar

import (
	"testing"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
)

func baseInput(closes []float64) quant.StrategyInput {
	n := len(closes)
	times := make([]int64, n)
	for i := range times {
		times[i] = int64(i) * 60_000
	}
	return quant.StrategyInput{
		NowMs: times[n-1], Price: closes[n-1], Closes: closes, Times: times,
		Portfolio:          quant.Portfolio{Cash: 100_000},
		Spawn:              quant.DefaultSpawnPoint(),
		MicroReservePct:    0.1,
		Sigmoid:            quant.DefaultSigmoidParams(),
		MinTradeThreshold:  100,
	}
}

func TestStepEmpty(t *testing.T) {
	out := NewDefault().Step(quant.StrategyInput{})
	if len(out.Orders) != 0 {
		t.Fatalf("expected no orders")
	}
}

func TestStepMacroDCAOnDip(t *testing.T) {
	closes := make([]float64, 80)
	for i := 0; i < 70; i++ {
		closes[i] = 100
	}
	for i := 70; i < 80; i++ {
		closes[i] = 100 - float64(i-69)*2
	}
	in := baseInput(closes)
	s := NewDefault()
	s.P.BetaThreshold = 0.05
	s.P.MinTradeThreshold = 50
	out := s.Step(in)
	macro := 0
	for _, o := range out.Orders {
		if o.Engine == "MACRO" && o.Side == quant.SideBuy {
			macro++
		}
	}
	if macro == 0 {
		t.Fatalf("expected MACRO buy, signal=%v orders=%v", out.Signal, out.Orders)
	}
}

func TestStepMicroSigmoid(t *testing.T) {
	closes := make([]float64, 50)
	for i := range closes {
		closes[i] = 100
	}
	in := baseInput(closes)
	in.Portfolio.FloatHold = 200
	in.Portfolio.Cash = 80_000
	out := NewDefault().Step(in)
	if out.TargetW < 0 || out.TargetW > 1 {
		t.Fatalf("target weight %v", out.TargetW)
	}
}

func TestManifest(t *testing.T) {
	m := NewDefault().Manifest()
	if m.ID != StrategyID || !m.IsSpot {
		t.Fatalf("%+v", m)
	}
}

func TestClamp(t *testing.T) {
	p := Params{Beta: 999, EMAPeriod: 1}.Clamp()
	if p.Beta > 10 || p.EMAPeriod < 5 {
		t.Fatalf("%+v", p)
	}
}
