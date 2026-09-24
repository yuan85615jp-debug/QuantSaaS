package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"go.uber.org/zap"
)

type captureHandler struct {
	fills  []protocol.FillReport
	deltas []protocol.DeltaReport
}

func (c *captureHandler) OnFill(r protocol.FillReport)   { c.fills = append(c.fills, r) }
func (c *captureHandler) OnDelta(r protocol.DeltaReport) { c.deltas = append(c.deltas, r) }

func TestHubTradeAndFillRoundtrip(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret-phase8", ExpireHour: 1}}
	authSvc := auth.NewService(cfg)
	token, err := authSvc.SignToken(1, "agent")
	require.NoError(t, err)

	cap := &captureHandler{}
	hub := NewHub(cap, zap.NewNop())
	defer hub.Close()

	h := &Handler{Hub: hub, Auth: authSvc, Log: zap.NewNop()}
	srv := httptest.NewServer(h)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/agent?token=" + token + "&agent_id=a1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	helloRaw, err := protocol.Encode(protocol.TypeHello, "", protocol.Hello{
		AgentID: "a1", Version: "0.1.0", InstanceIDs: []uint{42},
	})
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, helloRaw))
	time.Sleep(50 * time.Millisecond)

	require.Contains(t, hub.OnlineAgents(), "a1")

	cmd := protocol.TradeCommand{
		ClientOrderID: "c-1", InstanceID: 42, Symbol: "510300",
		Side: "BUY", Engine: "MACRO", Qty: 100,
	}
	require.NoError(t, hub.SendTrade(cmd))

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	w, err := protocol.Decode(data)
	require.NoError(t, err)
	require.Equal(t, protocol.TypeTradeCommand, w.Type)
	got, err := protocol.DecodePayload[protocol.TradeCommand](w)
	require.NoError(t, err)
	require.Equal(t, "c-1", got.ClientOrderID)

	fillRaw, err := protocol.Encode(protocol.TypeFillReport, "c-1", protocol.FillReport{
		ClientOrderID: "c-1", InstanceID: 42, Symbol: "510300",
		Side: "BUY", Engine: "MACRO", FilledQty: 100, FilledPrice: 4.5,
		Status: "filled", TsMs: time.Now().UnixMilli(),
	})
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, fillRaw))
	time.Sleep(50 * time.Millisecond)
	require.Len(t, cap.fills, 1)
	require.Equal(t, "filled", cap.fills[0].Status)
}

func TestUnauthorized(t *testing.T) {
	hub := NewHub(nil, zap.NewNop())
	defer hub.Close()
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret-phase8", ExpireHour: 1}}
	h := &Handler{Hub: hub, Auth: auth.NewService(cfg)}
	srv := httptest.NewServer(h)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/agent"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.Error(t, err)
	if resp != nil {
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}
