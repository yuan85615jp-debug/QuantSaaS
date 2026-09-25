package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEastMoneySecID(t *testing.T) {
	require.Equal(t, "1.510300", eastMoneySecID("510300"))
	require.Equal(t, "0.159915", eastMoneySecID("159915"))
	require.Equal(t, "1.600519", eastMoneySecID("600519.SS"))
}

func TestEastMoneyFetchMock(t *testing.T) {
	const payload = `{"data":{"code":"510300","klines":[
"2024-06-03 09:31,4.50,4.52,4.53,4.49,12000",
"2024-06-03 09:32,4.52,4.51,4.54,4.50,11000",
"2024-06-03 09:33,4.51,4.55,4.56,4.50,13000"
]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "1.510300", r.URL.Query().Get("secid"))
		require.Equal(t, "1", r.URL.Query().Get("klt"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	p := NewEastMoney()
	p.BaseURL = srv.URL
	p.HTTP = srv.Client()

	bars, err := p.FetchKlines(context.Background(), "510300", "1m", 10)
	require.NoError(t, err)
	require.Len(t, bars, 3)
	require.InDelta(t, 4.55, bars[2].Close, 1e-9)
	require.Greater(t, bars[2].OpenTime, bars[0].OpenTime)
}

func TestParseEMTimeDaily(t *testing.T) {
	ms, err := parseEMTime("2024-06-03", "1d")
	require.NoError(t, err)
	require.Greater(t, ms, int64(0))
}
