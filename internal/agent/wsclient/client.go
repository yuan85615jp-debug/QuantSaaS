package wsclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/executor"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"go.uber.org/zap"
)

type Client struct {
	BaseURL     string
	Token       string
	AgentID     string
	InstanceIDs []uint
	Executor    *executor.Executor
	Log         *zap.Logger

	MarkPriceProvider func(symbol string) float64

	mu   sync.Mutex
	conn *websocket.Conn
}

func (c *Client) wsURL() (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported scheme %s", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/ws/agent"
	q := u.Query()
	q.Set("token", c.Token)
	q.Set("agent_id", c.AgentID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) Run(ctx context.Context) {
	if c.Log == nil {
		c.Log = zap.NewNop()
	}
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := c.session(ctx)
		if ctx.Err() != nil {
			return
		}
		c.Log.Warn("ws session ended; reconnecting", zap.Error(err), zap.Duration("backoff", backoff))
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (c *Client) session(ctx context.Context) error {
	wsURL, err := c.wsURL()
	if err != nil {
		return err
	}
	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+c.Token)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, hdr)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		_ = conn.Close()
	}()

	hello, err := protocol.Encode(protocol.TypeHello, "", protocol.Hello{
		AgentID: c.AgentID, Version: "0.8.0", InstanceIDs: c.InstanceIDs,
	})
	if err != nil {
		return err
	}
	if err := conn.WriteMessage(websocket.TextMessage, hello); err != nil {
		return err
	}

	sessCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.heartbeat(sessCtx, conn)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if err := c.handleFrame(conn, data); err != nil {
			c.Log.Warn("handle frame", zap.Error(err))
		}
	}
}

func (c *Client) heartbeat(ctx context.Context, conn *websocket.Conn) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			raw, err := protocol.Encode(protocol.TypeHeartbeat, "", protocol.Heartbeat{
				AgentID: c.AgentID, TsMs: time.Now().UnixMilli(),
			})
			if err != nil {
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleFrame(conn *websocket.Conn, data []byte) error {
	w, err := protocol.Decode(data)
	if err != nil {
		return err
	}
	switch w.Type {
	case protocol.TypeTradeCommand:
		cmd, err := protocol.DecodePayload[protocol.TradeCommand](w)
		if err != nil {
			return err
		}
		if c.MarkPriceProvider != nil && c.Executor != nil {
			if px := c.MarkPriceProvider(cmd.Symbol); px > 0 {
				c.Executor.SetMarkPrice(cmd.Symbol, px)
			}
		}
		if c.Executor == nil {
			return fmt.Errorf("no executor")
		}
		fill := c.Executor.HandleTrade(context.Background(), cmd)
		raw, err := protocol.Encode(protocol.TypeFillReport, cmd.ClientOrderID, fill)
		if err != nil {
			return err
		}
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			return err
		}
		delta, err := c.Executor.BuildDelta(context.Background(), cmd.InstanceID, cmd.Symbol)
		if err == nil {
			draw, err := protocol.Encode(protocol.TypeDeltaReport, cmd.ClientOrderID, delta)
			if err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, draw)
			}
		}
	}
	return nil
}
