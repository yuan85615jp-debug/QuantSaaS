package quant

import "testing"

func TestGhostDCALumpSum(t *testing.T) {
	closes := []float64{10, 11, 9, 12}
	spawn := DefaultSpawnPoint()
	spawn.InitialCapital = 1000
	spawn.MonthlyInject = 0
	b := GhostDCA(closes, spawn, nil)
	if len(b.Equity) != 4 {
		t.Fatal(len(b.Equity))
	}
	if b.Equity[3] < 1199 || b.Equity[3] > 1201 {
		t.Fatalf("final equity %v", b.Equity[3])
	}
}
