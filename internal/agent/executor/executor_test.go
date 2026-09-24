package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/broker"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
)

func TestHandleTradeBuyAndDelta(t *testing.T) {
	b := broker.NewPaper(broker.PaperOpts{InitialCash: 100_000, LotStep: 100, LotMin: 100})
	ex := New(b, "agent-1")
	ex.SetMarkPrice("510300", 4.0)

	cmd := protocol.TradeCommand{
		ClientOrderID: "ord-1",
		InstanceID:    42,
		Symbol:        "510300",
		Side:          "BUY",
		Engine:        "MACRO",
		Qty:           1000,
		OrderType:     "MARKET",
	}
	require.NoError(t, ValidateCommand(cmd))

	fill := ex.HandleTrade(context.Background(), cmd)
	require.Equal(t, "filled", fill.Status)
	require.InDelta(t, 1000, fill.FilledQty, 1e-9)
	require.InDelta(t, 4.0, fill.FilledPrice, 1e-9)
	require.Equal(t, "MACRO", fill.Engine)
	require.Equal(t, uint(42), fill.InstanceID)

	delta, err := ex.BuildDelta(context.Background(), 42, "510300")
	require.NoError(t, err)
	require.InDelta(t, 1000, delta.Shares, 1e-9)
	require.True(t, delta.Cash < 100_000)
}

func TestHandleTradeInvalid(t *testing.T) {
	b := broker.NewPaper(broker.PaperOpts{InitialCash: 10_000})
	ex := New(b, "a")
	fill := ex.HandleTrade(context.Background(), protocol.TradeCommand{})
	require.Equal(t, "failed", fill.Status)
	require.NotEmpty(t, fill.ErrorMsg)
}
