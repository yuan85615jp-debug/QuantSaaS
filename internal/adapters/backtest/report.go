package backtest

import (
	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
)

// BenchmarkReport compares a strategy Result against Ghost DCA on the same path.
type BenchmarkReport struct {
	StrategyROI    float64 `json:"strategy_roi"`
	StrategyMaxDD  float64 `json:"strategy_max_dd"`
	StrategyTrades int     `json:"strategy_trades"`
	FrictionCost   float64 `json:"friction_cost"`
	DCAROI         float64 `json:"dca_roi"`
	DCAMaxDD       float64 `json:"dca_max_dd"`
	// ExcessROI = strategy ROI - DCA ROI (positive means beat passive DCA)
	ExcessROI float64 `json:"excess_roi"`
	// BeatDCA is true when strategy ROI strictly exceeds DCA ROI
	BeatDCA bool `json:"beat_dca"`
}

// Report builds a benchmark summary. times may be nil (no monthly inject boundaries).
func Report(res Result, closes []float64, times []int64, spawn quant.SpawnPoint) BenchmarkReport {
	var bounds []bool
	if times != nil {
		bounds = quant.MonthBoundaries(times)
	}
	dca := quant.GhostDCA(closes, spawn, bounds)
	rep := BenchmarkReport{
		StrategyROI:    res.ROI,
		StrategyMaxDD:  res.MaxDD,
		StrategyTrades: res.TradeCount,
		FrictionCost:   res.FrictionCost,
		DCAROI:         dca.ROI,
		DCAMaxDD:       dca.MaxDD,
		ExcessROI:      res.ROI - dca.ROI,
		BeatDCA:        res.ROI > dca.ROI,
	}
	return rep
}
