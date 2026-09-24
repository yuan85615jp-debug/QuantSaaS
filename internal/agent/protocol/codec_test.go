package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeTrade(t *testing.T) {
	cmd := TradeCommand{
		ClientOrderID: "o1", InstanceID: 7, Symbol: "510300",
		Side: "BUY", Engine: "MICRO", Qty: 100,
	}
	raw, err := Encode(TypeTradeCommand, "req-1", cmd)
	require.NoError(t, err)

	w, err := Decode(raw)
	require.NoError(t, err)
	require.Equal(t, TypeTradeCommand, w.Type)
	require.Equal(t, "req-1", w.RequestID)

	got, err := DecodePayload[TradeCommand](w)
	require.NoError(t, err)
	require.Equal(t, cmd, got)
}
