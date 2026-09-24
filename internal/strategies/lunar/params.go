package lunar

import "github.com/yuan85615jp-debug/QuantSaaS/internal/quant"

const StrategyID = "lunar"

// Params is the evolvable chromosome (macro old-farmer + micro PDE/Sigmoid).
type Params struct {
	MaxDCAMonths        float64 `json:"max_dca_months"`
	BetaThreshold       float64 `json:"beta_threshold"`
	MoonPhasePressure   float64 `json:"moon_phase_pressure"`
	DCAFraction         float64 `json:"dca_fraction"`
	EMAPeriod           int     `json:"ema_period"`
	Kp                  float64 `json:"kp"`
	Kv                  float64 `json:"kv"`
	Ka                  float64 `json:"ka"`
	MinTradeThreshold   float64 `json:"min_trade_threshold"`
	MicroReservePct     float64 `json:"micro_reserve_pct"`
	Beta                float64 `json:"beta"`
	Gamma               float64 `json:"gamma"`
	ReleaseAccThreshold float64 `json:"release_acc_threshold"`
	ReleaseFraction     float64 `json:"release_fraction"`
}

func DefaultParams() Params {
	return Params{
		MaxDCAMonths: 36, BetaThreshold: 0.08, MoonPhasePressure: 1.0,
		DCAFraction: 0.15, EMAPeriod: 60,
		Kp: 1.0, Kv: 0.5, Ka: 0.3,
		MinTradeThreshold: 500, MicroReservePct: 0.10,
		Beta: 1.2, Gamma: 0.4,
		ReleaseAccThreshold: 0.02, ReleaseFraction: 0.1,
	}
}

func (p Params) Clamp() Params {
	p.MaxDCAMonths = quant.Clamp(p.MaxDCAMonths, 6, 120)
	p.BetaThreshold = quant.Clamp(p.BetaThreshold, 0.01, 0.5)
	p.MoonPhasePressure = quant.Clamp(p.MoonPhasePressure, 0.2, 3)
	p.DCAFraction = quant.Clamp(p.DCAFraction, 0.02, 0.5)
	if p.EMAPeriod < 5 {
		p.EMAPeriod = 5
	}
	if p.EMAPeriod > 500 {
		p.EMAPeriod = 500
	}
	p.Kp = quant.Clamp(p.Kp, -5, 5)
	p.Kv = quant.Clamp(p.Kv, -5, 5)
	p.Ka = quant.Clamp(p.Ka, -5, 5)
	p.MinTradeThreshold = quant.Clamp(p.MinTradeThreshold, 50, 50_000)
	p.MicroReservePct = quant.Clamp(p.MicroReservePct, 0.02, 0.30)
	p.Beta = quant.Clamp(p.Beta, 0.1, 10)
	p.Gamma = quant.Clamp(p.Gamma, 0, 5)
	p.ReleaseAccThreshold = quant.Clamp(p.ReleaseAccThreshold, 0.001, 0.2)
	p.ReleaseFraction = quant.Clamp(p.ReleaseFraction, 0.01, 0.5)
	return p
}

func (p Params) Sigmoid() quant.SigmoidParams {
	return quant.SigmoidParams{Beta: p.Beta, Gamma: p.Gamma, MarketBetaMultiplier: 1.0}
}
