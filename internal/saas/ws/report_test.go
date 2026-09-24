package ws

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
)

type mockFiller struct {
	last    instance.FillInput
	calls   int
	errMark string
	fail    bool
}

func (m *mockFiller) ApplyFillReport(in instance.FillInput) error {
	m.calls++
	m.last = in
	if m.fail {
		return ErrNoAgent
	}
	return nil
}

func (m *mockFiller) MarkError(id uint, msg string) error {
	m.errMark = msg
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
		Symbol:        "510300",
	})
	require.Equal(t, 1, m.calls)
	require.Equal(t, "c1", m.last.ClientOrderID)
	require.Equal(t, uint(7), m.last.InstanceID)
	require.InDelta(t, 100, m.last.Qty, 1e-9)
	require.InDelta(t, 10.5, m.last.Price, 1e-9)

	b.OnFill(protocol.FillReport{ClientOrderID: "c2", InstanceID: 3, Status: "failed", ErrorMsg: "broker down"})
	require.Equal(t, "broker down", m.errMark)
	require.Equal(t, "failed", m.last.Status)
}
