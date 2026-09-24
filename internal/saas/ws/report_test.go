package ws

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
)

type mockFiller struct {
	lastAction store.TradeAction
	lastEngine store.TradeEngine
	lastQty    float64
	lastPrice  float64
	lastFee    float64
	lastInst   uint
	errMarked  string
	failApply  bool
}

func (m *mockFiller) ApplyFill(instanceID uint, action store.TradeAction, engine store.TradeEngine, qty, price, fee float64) error {
	if m.failApply {
		return ErrNoAgent
	}
	m.lastInst = instanceID
	m.lastAction = action
	m.lastEngine = engine
	m.lastQty = qty
	m.lastPrice = price
	m.lastFee = fee
	return nil
}

func (m *mockFiller) MarkError(id uint, msg string) error {
	m.lastInst = id
	m.errMarked = msg
	return nil
}

func TestFillBridgeOnFill(t *testing.T) {
	m := &mockFiller{}
	b := NewFillBridge(m, nil)

	b.OnFill(protocol.FillReport{
		ClientOrderID: "c1",
		InstanceID:    7,
		Side:          "BUY",
		Engine:        "MACRO",
		FilledQty:     100,
		FilledPrice:   10.5,
		Fee:           1.2,
		Status:        "filled",
	})
	require.Equal(t, uint(7), m.lastInst)
	require.Equal(t, store.ActionBuy, m.lastAction)
	require.Equal(t, store.EngineMacro, m.lastEngine)
	require.InDelta(t, 100, m.lastQty, 1e-9)
	require.InDelta(t, 10.5, m.lastPrice, 1e-9)
	require.InDelta(t, 1.2, m.lastFee, 1e-9)

	b.OnFill(protocol.FillReport{InstanceID: 3, Status: "failed", ErrorMsg: "broker down"})
	require.Equal(t, "broker down", m.errMarked)
	require.Equal(t, uint(3), m.lastInst)
}
