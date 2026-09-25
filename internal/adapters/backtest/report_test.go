package backtest

import (
	"testing"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
)

func TestReportBeatDCA(t *testing.T) {
	closes := []float64{100, 110, 120, 130}
	spawn := quant.SpawnPoint{InitialCapital: 10000, LotStep: 1, CommissionRate: 0}
	res := Result{
		Equity: []float64{10000, 11000, 12000, 13000},
		ROI:    quant.ROI(10000, 13000),
		MaxDD:  0,
	}
	rep := Report(res, closes, nil, spawn)
	if rep.DCAROI == 0 && closes[0] > 0 {
		t.Fatalf("dca roi zero: %+v", rep)
	}
}

func TestMedian(t *testing.T) {
	if median([]float64{3, 1, 2}) != 2 {
		t.Fatal("median")
	}
	if median([]float64{1, 2, 3, 4}) != 2.5 {
		t.Fatal("median even")
	}
}
