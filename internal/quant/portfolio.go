package quant

// Portfolio is the three-stack asset state used by Step() and the ledger.
// DeadHold: macro base, only increase until release rules fire.
// FloatHold: micro floating, can buy/sell.
// ColdSealedHold: never released.
type Portfolio struct {
	Cash           float64
	DeadHold       float64 // shares
	FloatHold      float64
	ColdSealedHold float64
}

func (p Portfolio) TotalEquity(price float64) float64 {
	if price < 0 {
		price = 0
	}
	return p.Cash + (p.DeadHold+p.FloatHold+p.ColdSealedHold)*price
}

func (p Portfolio) ReserveFloor(price, microReservePct float64) float64 {
	pct := Clamp(microReservePct, 0, 1)
	return p.TotalEquity(price) * pct
}

func (p Portfolio) SpendableCNY(price, microReservePct float64) float64 {
	s := p.Cash - p.ReserveFloor(price, microReservePct)
	if s < 0 {
		return 0
	}
	return s
}

func (p Portfolio) CurrentMicroWeight(price float64) float64 {

eq := p.TotalEquity(price)
	if eq <= 0 || price <= 0 {
		return 0
	}
	return (p.FloatHold * price) / eq
}

func (p Portfolio) Clone() Portfolio {
	return p
}
