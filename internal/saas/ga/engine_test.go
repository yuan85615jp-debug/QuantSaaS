package ga

import (
	"math"
	"testing"
)

func synthCloses(n int) ([]float64, []int64) {
	closes := make([]float64, n)
	times := make([]int64, n)
	p := 100.0
	for i := 0; i < n; i++ {
		p = p * (1 + 0.01*math.Sin(float64(i)/20) + 0.002*math.Sin(float64(i)/7))
		if p < 1 {
			p = 1
		}
		closes[i] = p
		times[i] = int64(i) * 86400_000
	}
	return closes, times
}

func TestEngineTestMode(t *testing.T) {
	closes, times := synthCloses(250)
	cfg := DefaultConfig()
	cfg.TestMode = true
	eng := NewEngine(LunarEvolvable{}, cfg, 42)
	res := eng.Run(closes, times)
	if len(res.Best.Gene) != 14 {
		t.Fatalf("gene dim %d", len(res.Best.Gene))
	}
	if len(res.History) == 0 {
		t.Fatal("no history")
	}
	t.Logf("best score=%.4f maxDD=%.3f trades=%d", res.Best.Score, res.Best.MaxDD, res.Best.Trades)
}

func TestLunarSampleClamp(t *testing.T) {
	e := LunarEvolvable{}
	eng := NewEngine(e, DefaultConfig(), 1)
	g := e.Sample(eng.RNG)
	g2 := e.Clamp(g)
	if len(g2) != e.Dim() {
		t.Fatal(len(g2))
	}
}
