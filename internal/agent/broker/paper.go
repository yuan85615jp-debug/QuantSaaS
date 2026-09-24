package broker

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// PaperBroker simulates fills at the last mark price with lot rounding and fees.
// No network; suitable for local dev and unit tests.
type PaperBroker struct {
	mu             sync.Mutex
	commissionRate float64
	stampTaxRate   float64
	lotStep        float64
	lotMin         float64
	cash           float64
	shares         map[string]float64
	marks          map[string]float64
}

type PaperOpts struct {
	InitialCash    float64
	CommissionRate float64
	StampTaxRate   float64
	LotStep        float64
	LotMin         float64
}

func NewPaper(opts PaperOpts) *PaperBroker {
	if opts.InitialCash <= 0 {
		opts.InitialCash = 100_000
	}
	if opts.CommissionRate < 0 {
		opts.CommissionRate = 0.0003
	}
	if opts.StampTaxRate < 0 {
		opts.StampTaxRate = 0.001
	}
	if opts.LotStep <= 0 {
		opts.LotStep = 100
	}
	if opts.LotMin <= 0 {
		opts.LotMin = 100
	}
	return &PaperBroker{
		commissionRate: opts.CommissionRate,
		stampTaxRate:   opts.StampTaxRate,
		lotStep:        opts.LotStep,
		lotMin:         opts.LotMin,
		cash:           opts.InitialCash,
		shares:         map[string]float64{},
		marks:          map[string]float64{},
	}
}

func (p *PaperBroker) Name() string { return "paper" }

func (p *PaperBroker) SetMarkPrice(symbol string, price float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if price > 0 {
		p.marks[symbol] = price
	}
}

func (p *PaperBroker) normalizeQty(qty float64) float64 {
	if qty <= 0 {
		return 0
	}
	q := math.Floor(qty/p.lotStep) * p.lotStep
	if q < p.lotMin {
		return 0
	}
	return q
}

func (p *PaperBroker) PlaceOrder(ctx context.Context, req OrderRequest) (OrderResult, error) {
	_ = ctx
	out := OrderResult{
		ClientOrderID: req.ClientOrderID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Status:        "failed",
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	price := p.marks[req.Symbol]
	if price <= 0 {
		out.ErrorMsg = "no mark price"
		return out, fmt.Errorf("paper: no mark price for %s", req.Symbol)
	}
	qty := p.normalizeQty(req.Qty)
	if qty <= 0 {
		out.ErrorMsg = "qty below lot minimum"
		return out, fmt.Errorf("paper: qty below lot min")
	}

	switch req.Side {
	case "BUY":
		notional := qty * price
		fee := notional * p.commissionRate
		cost := notional + fee
		if cost > p.cash {
			affordable := math.Floor((p.cash/(price*(1+p.commissionRate)))/p.lotStep) * p.lotStep
			if affordable < p.lotMin {
				out.ErrorMsg = "insufficient cash"
				return out, fmt.Errorf("paper: insufficient cash")
			}
			qty = affordable
			notional = qty * price
			fee = notional * p.commissionRate
			cost = notional + fee
		}
		p.cash -= cost
		p.shares[req.Symbol] += qty
		out.FilledQty = qty
		out.FilledPrice = price
		out.Fee = fee
		out.Status = "filled"
		return out, nil

	case "SELL":
		have := p.shares[req.Symbol]
		if qty > have {
			qty = p.normalizeQty(have)
			if qty <= 0 {
				out.ErrorMsg = "insufficient shares"
				return out, fmt.Errorf("paper: insufficient shares")
			}
		}
		notional := qty * price
		fee := notional * (p.commissionRate + p.stampTaxRate)
		p.shares[req.Symbol] -= qty
		p.cash += notional - fee
		out.FilledQty = qty
		out.FilledPrice = price
		out.Fee = fee
		out.Status = "filled"
		return out, nil

	default:
		out.ErrorMsg = "unknown side"
		return out, fmt.Errorf("paper: unknown side %q", req.Side)
	}
}

func (p *PaperBroker) Snapshot(ctx context.Context, symbol string) (Position, error) {
	_ = ctx
	p.mu.Lock()
	defer p.mu.Unlock()
	return Position{
		Symbol: symbol,
		Shares: p.shares[symbol],
		Cash:   p.cash,
	}, nil
}
