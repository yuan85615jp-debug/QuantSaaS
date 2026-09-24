package lunar

import (
	"math"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategy"
)

type Strategy struct {
	P Params
}

func New(p Params) *Strategy {
	return &Strategy{P: p.Clamp()}
}

func NewDefault() *Strategy {
	return New(DefaultParams())
}

func (s *Strategy) Manifest() strategy.Manifest {
	return strategy.Manifest{
		ID: StrategyID, Name: "Lunar Macro-DCA + Micro-PDE", Version: "0.1.0", IsSpot: true,
	}
}

// Step is the single pure decision entry for backtest and live.
func (s *Strategy) Step(in quant.StrategyInput) quant.StrategyOutput {
	p := s.P.Clamp()
	out := quant.StrategyOutput{Runtime: map[string]float64{}}
	if in.Price <= 0 || len(in.Closes) == 0 {
		return out
	}

	closes := in.Closes
	n := len(closes)
	price := in.Price
	if n > 0 {
		price = closes[n-1]
	}
	port := in.Portfolio

eq := port.TotalEquity(price)
	if eq <= 0 {
	
eq = port.Cash
	}

	emaPeriod := p.EMAPeriod
	if emaPeriod > n {
		emaPeriod = n
	}
	ema := quant.EMA(closes, emaPeriod)
	emaNow := ema[n-1]
	pos := 0.0
	if emaNow > 0 && price > 0 {
		pos = math.Log(price / emaNow)
	}
	vel := 0.0
	acc := 0.0
	if n >= 2 && closes[n-2] > 0 {
		vel = quant.LogReturn(closes[n-2], closes[n-1])
	}
	if n >= 3 && closes[n-3] > 0 && closes[n-2] > 0 {
		v1 := quant.LogReturn(closes[n-3], closes[n-2])
		v2 := quant.LogReturn(closes[n-2], closes[n-1])
		acc = v2 - v1
	}

	signal := quant.PDESignal(pos, vel, acc, p.Kp, p.Kv, p.Ka)
	out.Signal = signal
	out.Orders = append(out.Orders, macroDCA(p, port, price, pos, in.Spawn)...)

	if port.DeadHold > 0 && math.Abs(acc) >= p.ReleaseAccThreshold {
		qty := in.Spawn.NormalizeQty(port.DeadHold * p.ReleaseFraction)
		if qty > 0 && qty <= port.DeadHold {
			out.Releases = append(out.Releases, quant.ReleaseIntent{Qty: qty, Note: "acc_release"})
		}
	}

	cw := port.CurrentMicroWeight(price)
	tw := quant.TargetWeight(cw, signal, p.Sigmoid())
	out.TargetW = tw
	theo := quant.TheoreticalCNY(tw, cw, eq)
	minNotional := p.MinTradeThreshold
	if in.MinTradeThreshold > 0 {
		minNotional = in.MinTradeThreshold
	}
	if math.Abs(theo) >= minNotional && price > 0 {
		rawShares := math.Abs(theo) / price
		qty := in.Spawn.NormalizeQty(rawShares)
		if qty > 0 {
			side := quant.SideBuy
			if theo < 0 {
				side = quant.SideSell
				if qty > port.FloatHold {
					qty = in.Spawn.NormalizeQty(port.FloatHold)
				}
			} else {
				spend := port.SpendableCNY(price, p.MicroReservePct)
				maxShares := spend / price
				if qty > maxShares {
					qty = in.Spawn.NormalizeQty(maxShares)
				}
			}
			if qty > 0 {
				out.Orders = append(out.Orders, quant.OrderIntent{
					Side: side, Engine: "MICRO", Qty: qty, Note: "sigmoid",
				})
			}
		}
	}

	out.Runtime["pos"] = pos
	out.Runtime["vel"] = vel
	out.Runtime["acc"] = acc
	out.Runtime["signal"] = signal
	out.Runtime["target_w"] = tw
	out.Runtime["micro_w"] = cw
	return out
}

func macroDCA(p Params, port quant.Portfolio, price, pos float64, spawn quant.SpawnPoint) []quant.OrderIntent {
	if price <= 0 {
		return nil
	}
	if pos > -p.BetaThreshold {
		return nil
	}
	spend := port.SpendableCNY(price, p.MicroReservePct)
	if spend <= 0 {
		return nil
	}
	frac := quant.Clamp(p.DCAFraction*p.MoonPhasePressure, 0.01, 0.5)
	notional := spend * frac
	if notional < p.MinTradeThreshold {
		return nil
	}
	qty := spawn.NormalizeQty(notional / price)
	if qty <= 0 {
		return nil
	}
	return []quant.OrderIntent{{
		Side: quant.SideBuy, Engine: "MACRO", Qty: qty, Note: "dca_dead",
	}}
}

var _ strategy.Strategy = (*Strategy)(nil)
