package quant

import "math"

// SpawnPoint is frozen at Epoch start and shared by the whole population.
// It does not enter the genome fingerprint / crossover.
type SpawnPoint struct {
	InitialCapital   float64 `json:"initial_capital"`
	MonthlyInject    float64 `json:"monthly_inject"`
	CommissionRate   float64 `json:"commission_rate"`
	StampTaxRate     float64 `json:"stamp_tax_rate"`
	LotStep          float64 `json:"lot_step"`
	LotMin           float64 `json:"lot_min"`
	DeadReserveRatio float64 `json:"dead_reserve_ratio"`
	GlobalStopLoss   float64 `json:"global_stop_loss"`
	MaxLeverage      float64 `json:"max_leverage"`
}

func DefaultSpawnPoint() SpawnPoint {
	return SpawnPoint{
		InitialCapital:   100_000,
		MonthlyInject:    0,
		CommissionRate:   0.0003,
		StampTaxRate:     0.001,
		LotStep:          100,
		LotMin:           100,
		DeadReserveRatio: 0.05,
		GlobalStopLoss:   0,
		MaxLeverage:      1,
	}
}

func (s SpawnPoint) RoundTripCostRate() float64 {
	return 2*s.CommissionRate + s.StampTaxRate
}

func (s SpawnPoint) NormalizeQty(qty float64) float64 {
	if qty <= 0 {
		return 0
	}
	if s.LotStep <= 0 {
		if qty < s.LotMin {
			return 0
		}
		return qty
	}
	q := math.Floor(qty/s.LotStep) * s.LotStep
	if q < s.LotMin {
		return 0
	}
	return q
}
