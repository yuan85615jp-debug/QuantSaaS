package broker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaperBuySell(t *testing.T) {
	b := NewPaper(PaperOpts{InitialCash: 50_000, LotStep: 100, LotMin: 100})
	b.SetMarkPrice("510300", 10)

	res, err := b.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c1", Symbol: "510300", Side: "BUY", Qty: 1000, OrderType: "MARKET",
	})
	require.NoError(t, err)
	require.Equal(t, "filled", res.Status)
	require.InDelta(t, 1000, res.FilledQty, 1e-9)
	require.InDelta(t, 10, res.FilledPrice, 1e-9)

	pos, err := b.Snapshot(context.Background(), "510300")
	require.NoError(t, err)
	require.InDelta(t, 1000, pos.Shares, 1e-9)
	require.True(t, pos.Cash < 50_000)

	b.SetMarkPrice("510300", 11)
	res, err = b.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c2", Symbol: "510300", Side: "SELL", Qty: 400, OrderType: "MARKET",
	})
	require.NoError(t, err)
	require.Equal(t, "filled", res.Status)
	require.InDelta(t, 400, res.FilledQty, 1e-9)

	pos, err = b.Snapshot(context.Background(), "510300")
	require.NoError(t, err)
	require.InDelta(t, 600, pos.Shares, 1e-9)
}

func TestPaperLotAndNoPrice(t *testing.T) {
	b := NewPaper(PaperOpts{InitialCash: 10_000})
	_, err := b.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "x", Symbol: "510300", Side: "BUY", Qty: 100,
	})
	require.Error(t, err)

	b.SetMarkPrice("510300", 5)
	_, err = b.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "x", Symbol: "510300", Side: "BUY", Qty: 50,
	})
	require.Error(t, err)
}
