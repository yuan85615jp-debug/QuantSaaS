package quant

import "testing"

func TestPortfolioMath(t *testing.T) {
	p := Portfolio{Cash: 50_000, FloatHold: 100, DeadHold: 50}
	price := 100.0

eq := p.TotalEquity(price)
	if eq != 65_000 {
		t.Fatalf("equity %v", eq)
	}
	w := p.CurrentMicroWeight(price)
	if w < 0.15 || w > 0.16 {
		t.Fatalf("weight %v", w)
	}
	spend := p.SpendableCNY(price, 0.1)
	if spend != 43_500 {
		t.Fatalf("spendable %v", spend)
	}
}

func TestNormalizeQty(t *testing.T) {
	s := DefaultSpawnPoint()
	if s.NormalizeQty(250) != 200 {
		t.Fatalf("250 → %v", s.NormalizeQty(250))
	}
	if s.NormalizeQty(50) != 0 {
		t.Fatalf("below lot min should be 0")
	}
}
