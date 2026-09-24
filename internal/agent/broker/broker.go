package broker

import "context"

// OrderRequest is a pure execution request (no strategy semantics).
type OrderRequest struct {
	ClientOrderID string
	Symbol        string
	Side          string // BUY | SELL
	Qty           float64
	OrderType     string // MARKET
}

// OrderResult is the broker fill (or failure).
type OrderResult struct {
	ClientOrderID string
	Symbol        string
	Side          string
	FilledQty     float64
	FilledPrice   float64
	Fee           float64
	Status        string // filled | failed | partial
	ErrorMsg      string
}

// Position is the broker-visible holding for one symbol.
type Position struct {
	Symbol string
	Shares float64
	Cash   float64 // account-level cash (same for all symbols on paper)
}

// Broker is the only way Agent talks to a securities firm.
// Implementations must keep API keys internal; never log secrets.
type Broker interface {
	Name() string
	// SetMarkPrice injects the latest price for market orders (paper / sim).
	SetMarkPrice(symbol string, price float64)
	PlaceOrder(ctx context.Context, req OrderRequest) (OrderResult, error)
	// Snapshot returns cash + shares for symbol (cash is account-wide).
	Snapshot(ctx context.Context, symbol string) (Position, error)
}
