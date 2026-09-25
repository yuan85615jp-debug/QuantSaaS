package market

import "context"

// Provider fetches OHLCV bars from an external market data source.
type Provider interface {
	Name() string
	FetchKlines(ctx context.Context, symbol, interval string, limit int) ([]BarInput, error)
}
