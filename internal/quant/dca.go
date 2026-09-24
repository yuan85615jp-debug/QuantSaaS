package quant

import "time"

// DCABaseline is the passive Ghost DCA equity path used as Alpha benchmark.
type DCABaseline struct {
	Equity []float64
	ROI    float64
	MaxDD  float64
}

// GhostDCA simulates lump-sum at first close plus optional monthly cash-in buys.
func GhostDCA(closes []float64, spawn SpawnPoint, monthBoundary []bool) DCABaseline {
	n := len(closes)
	out := DCABaseline{Equity: make([]float64, n)}
	if n == 0 || spawn.InitialCapital <= 0 {
		return out
	}
	cash := spawn.InitialCapital
	shares := 0.0
	if closes[0] > 0 {
		shares = cash / closes[0]
		cash = 0
	}
	out.Equity[0] = shares*closes[0] + cash

	for i := 1; i < n; i++ {
		if monthBoundary != nil && i < len(monthBoundary) && monthBoundary[i] && spawn.MonthlyInject > 0 && closes[i] > 0 {
			cash += spawn.MonthlyInject
			shares += cash / closes[i]
			cash = 0
		}
		out.Equity[i] = shares*closes[i] + cash
	}
	out.ROI = ROI(spawn.InitialCapital, out.Equity[n-1])
	out.MaxDD = MaxDrawdown(out.Equity)
	return out
}

// MonthBoundaries marks the first bar of each new calendar month (UTC).
func MonthBoundaries(times []int64) []bool {
	n := len(times)
	out := make([]bool, n)
	if n == 0 {
		return out
	}
	prev := time.UnixMilli(times[0]).UTC()
	prevYM := prev.Year()*12 + int(prev.Month())
	for i := 1; i < n; i++ {
		t := time.UnixMilli(times[i]).UTC()
		ym := t.Year()*12 + int(t.Month())
		if ym != prevYM {
			out[i] = true
			prevYM = ym
		}
	}
	return out
}
