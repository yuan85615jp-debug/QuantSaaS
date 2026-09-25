package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiveDryRunRefusesOrder(t *testing.T) {
	b := NewLive(LiveOpts{})
	require.Equal(t, "live-dryrun", b.Name())
	_, err := b.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c1", Symbol: "510300", Side: "BUY", Qty: 100, OrderType: "MARKET",
	})
	require.Error(t, err)
}

func TestLivePlaceOrderHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/orders", r.URL.Path)
		require.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "510300", body["symbol"])
		_ = json.NewEncoder(w).Encode(map[string]any{
			"filled_qty": 100, "filled_price": 4.55, "fee": 0.14, "status": "filled",
		})
	}))
	defer srv.Close()

	b := NewLive(LiveOpts{
		BaseURL: srv.URL, APIKey: "test-key", APISecret: "sec", AccountID: "acc1", DryRun: false,
	})
	res, err := b.PlaceOrder(context.Background(), OrderRequest{
		ClientOrderID: "c2", Symbol: "510300", Side: "BUY", Qty: 150,
	})
	require.NoError(t, err)
	require.Equal(t, "filled", res.Status)
	require.InDelta(t, 100, res.FilledQty, 1e-9)
	require.InDelta(t, 4.55, res.FilledPrice, 1e-9)
}

func TestNormalizeLot(t *testing.T) {
	require.Equal(t, 0.0, normalizeLot(50, 100, 100))
	require.Equal(t, 200.0, normalizeLot(250, 100, 100))
}
