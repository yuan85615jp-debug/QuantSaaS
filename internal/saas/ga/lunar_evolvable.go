package ga

import (
	"math"
	"math/rand"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/adapters/backtest"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
)

type LunarEvolvable struct{}

func (LunarEvolvable) StrategyID() string { return lunar.StrategyID }
func (LunarEvolvable) Dim() int           { return 14 }

func (LunarEvolvable) Sample(rng *rand.Rand) []float64 {
	v := make([]float64, 14)
	v[0] = 6 + rng.Float64()*114
	v[1] = 0.01 + rng.Float64()*0.49
	v[2] = 0.2 + rng.Float64()*2.8
	v[3] = 0.02 + rng.Float64()*0.48
	v[4] = float64(5 + rng.Intn(496))
	v[5] = rng.NormFloat64()
	v[6] = rng.NormFloat64()
	v[7] = rng.NormFloat64()
	v[8] = 50 + rng.Float64()*5000
	v[9] = 0.02 + rng.Float64()*0.28
	v[10] = 0.1 + rng.Float64()*9.9
	v[11] = rng.Float64() * 5
	v[12] = 0.001 + rng.Float64()*0.199
	v[13] = 0.01 + rng.Float64()*0.49
	return lunar.FromSlice(v).ToSlice()
}

func (LunarEvolvable) Clamp(gene []float64) []float64 {
	return lunar.FromSlice(gene).ToSlice()
}

func (LunarEvolvable) Evaluate(gene []float64, closes []float64, times []int64, cfg Config) (float64, float64, [4]float64, int) {
	var wins [4]float64
	p := lunar.FromSlice(gene)
	strat := lunar.New(p)
	spawn := cfg.Spawn
	n := len(closes)
	if n < 10 {
		return -1e6, 1, wins, 0
	}
	type winSpec struct {
		bars   int
		weight float64
	}
	specs := []winSpec{
		{n, cfg.WAll},
		{minInt(n, 1825), cfg.W5y},
		{minInt(n, 730), cfg.W2y},
		{minInt(n, 183), cfg.W6m},
	}
	total := 0.0
	worstDD := 0.0
	trades := 0
	for _, wi := range []int{3, 2, 1, 0} {
		sp := specs[wi]
		if sp.weight == 0 || sp.bars < 5 {
			continue
		}
		start := n - sp.bars
		c := closes[start:]
		var t []int64
		if times != nil && len(times) == n {
			t = times[start:]
		}
		br := backtest.Run(strat, c, t, spawn, p.MicroReservePct)
		dca := quant.GhostDCA(c, spawn, quant.MonthBoundaries(t))
		alpha := br.ROI - dca.ROI
		ddPen := 1.5 * math.Max(0, br.MaxDD-dca.MaxDD)
		tradePen := 0.0001 * float64(br.TradeCount)
		slice := alpha - ddPen - tradePen
		if br.MaxDD >= cfg.FatalMaxDD {
			wins[wi] = -99999
			return -99999, br.MaxDD, wins, br.TradeCount
		}
		wins[wi] = slice
		total += sp.weight * slice
		if br.MaxDD > worstDD {
			worstDD = br.MaxDD
		}
		trades += br.TradeCount
	}
	return total, worstDD, wins, trades
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
