package backtest

import (
	"math"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategy"
)

type Result struct {
	Equity       []float64
	ROI          float64
	MaxDD        float64
	TradeCount   int
	FrictionCost float64
	FinalCash    float64
	FinalShares  float64
}

// Run walks closes bar-by-bar, calling strat.Step and applying intents with fees/lots.
func Run(strat strategy.Strategy, closes []float64, times []int64, spawn quant.SpawnPoint, microReserve float64) Result {
	n := len(closes)
	res := Result{Equity: make([]float64, n)}
	if n == 0 {
		return res
	}
	port := quant.Portfolio{Cash: spawn.InitialCapital}
	rt := map[string]float64{}

	for i := 0; i < n; i++ {
		price := closes[i]
		if price <= 0 {
			res.Equity[i] = port.TotalEquity(math.Max(price, 0))
			continue
		}
		histEnd := i + 1
		in := quant.StrategyInput{
			Price: price, Closes: closes[:histEnd], Portfolio: port, Spawn: spawn,
			Runtime: rt, MicroReservePct: microReserve,
		}
		if times != nil && i < len(times) {
			in.NowMs = times[i]
			in.Times = times[:histEnd]
		}
		out := strat.Step(in)
		if out.Runtime != nil {
			rt = out.Runtime
		}
		for _, rel := range out.Releases {
			q := spawn.NormalizeQty(rel.Qty)
			if q <= 0 || q > port.DeadHold {
				continue
			}
			port.DeadHold -= q
			port.FloatHold += q
		}
		for _, o := range out.Orders {
			q := spawn.NormalizeQty(o.Qty)
			if q <= 0 {
				continue
			}
			notional := q * price
			fee := notional * spawn.CommissionRate
			if o.Side == quant.SideSell {
				fee += notional * spawn.StampTaxRate
			}
			switch o.Side {
			case quant.SideBuy:
				cost := notional + fee
				if cost > port.Cash {
					q = spawn.NormalizeQty(port.Cash / (price * (1 + spawn.CommissionRate)))
					if q <= 0 {
						continue
					}
					notional = q * price
					fee = notional * spawn.CommissionRate
					cost = notional + fee
				}
				port.Cash -= cost
				res.FrictionCost += fee
				res.TradeCount++
				if o.Engine == "MACRO" {
					port.DeadHold += q
				} else {
					port.FloatHold += q
				}
			case quant.SideSell:
				if q > port.FloatHold {
					q = spawn.NormalizeQty(port.FloatHold)
					if q <= 0 {
						continue
					}
					notional = q * price
					fee = notional * (spawn.CommissionRate + spawn.StampTaxRate)
				}
				port.FloatHold -= q
				port.Cash += notional - fee
				res.FrictionCost += fee
				res.TradeCount++
			}
		}
		res.Equity[i] = port.TotalEquity(price)
	}
	res.FinalCash = port.Cash
	res.FinalShares = port.DeadHold + port.FloatHold + port.ColdSealedHold
	if n > 0 {
		res.ROI = quant.ROI(spawn.InitialCapital, res.Equity[n-1])
		res.MaxDD = quant.MaxDrawdown(res.Equity)
	}
	return res
}
