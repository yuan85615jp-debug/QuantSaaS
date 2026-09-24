package quant

import (
	"math"
	"testing"
)

func TestLogReturn(t *testing.T) {
	r := LogReturn(100, 110)
	want := math.Log(1.1)
	if math.Abs(r-want) > 1e-12 {
		t.Fatalf("got %v want %v", r, want)
	}
	if LogReturn(0, 1) != 0 {
		t.Fatal("zero price should return 0")
	}
}

func TestMaxDrawdown(t *testing.T) {

eq := []float64{100, 120, 90, 95}
	dd := MaxDrawdown(eq)
	if math.Abs(dd-0.25) > 1e-12 {
		t.Fatalf("got %v want 0.25", dd)
	}
}

func TestEMA(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}
	out := EMA(xs, 3)
	if len(out) != 5 || out[0] != 1 {
		t.Fatalf("unexpected %v", out)
	}
}
