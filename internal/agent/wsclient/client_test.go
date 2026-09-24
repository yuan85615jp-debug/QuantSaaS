package wsclient

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/broker"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/executor"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ws"
	"go.uber.org/zap"
)

type capH struct {
	fills []protocol.FillReport
}

func (c *capH) OnFill(r protocol.FillReport) { c.fills = append(c.fills, r) }
func (c *capH) OnDelta(protocol.DeltaReport) {}

func TestAgentClientRoundtrip(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "ws-client-test", ExpireHour: 1}}
	authSvc := auth.NewService(cfg)
	token, err := authSvc.SignToken(9, "agent")
	require.NoError(t, err)

	cap := &capH{}
	hub := ws.NewHub(cap, zap.NewNop())
	defer hub.Close()
	srv := httptest.NewServer(&ws.Handler{Hub: hub, Auth: authSvc, Log: zap.NewNop()})
	defer srv.Close()

	b := broker.NewPaper(broker.PaperOpts{InitialCash: 100_000, LotStep: 100, LotMin: 100})
	ex := executor.New(b, "agent-t")
	client := &Client{
		BaseURL:     srv.URL,
		Token:       token,
		AgentID:     "agent-t",
		InstanceIDs: []uint{99},
		Executor:    ex,
		Log:         zap.NewNop(),
		MarkPriceProvider: func(string) float64 { return 5.0 },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if contains(hub.OnlineAgents(), "agent-t") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.Contains(t, hub.OnlineAgents(), "agent-t")

	require.NoError(t, hub.SendTrade(protocol.TradeCommand{
		ClientOrderID: "ord-x", InstanceID: 99, Symbol: "510300",
		Side: "BUY", Engine: "MICRO", Qty: 200,
	}))

	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(cap.fills) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NotEmpty(t, cap.fills)
	require.Equal(t, "filled", cap.fills[0].Status)
	require.InDelta(t, 200, cap.fills[0].FilledQty, 1e-9)
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}
