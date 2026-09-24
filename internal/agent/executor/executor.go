package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/broker"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
)

// Executor turns TradeCommands into broker orders and protocol reports.
// It holds no strategy logic and never sees API keys (broker owns them).
type Executor struct {
	Broker  broker.Broker
	AgentID string
}

func New(b broker.Broker, agentID string) *Executor {
	return &Executor{Broker: b, AgentID: agentID}
}

func (e *Executor) HandleTrade(ctx context.Context, cmd protocol.TradeCommand) protocol.FillReport {
	now := time.Now().UnixMilli()
	rep := protocol.FillReport{
		ClientOrderID: cmd.ClientOrderID,
		InstanceID:    cmd.InstanceID,
		Symbol:        cmd.Symbol,
		Side:          cmd.Side,
		Engine:        cmd.Engine,
		Status:        "failed",
		TsMs:          now,
	}
	if cmd.ClientOrderID == "" || cmd.Symbol == "" || cmd.Qty <= 0 {
		rep.ErrorMsg = "invalid trade command"
		return rep
	}
	if cmd.Side != "BUY" && cmd.Side != "SELL" {
		rep.ErrorMsg = "invalid side"
		return rep
	}
	orderType := cmd.OrderType
	if orderType == "" {
		orderType = "MARKET"
	}
	res, err := e.Broker.PlaceOrder(ctx, broker.OrderRequest{
		ClientOrderID: cmd.ClientOrderID,
		Symbol:        cmd.Symbol,
		Side:          cmd.Side,
		Qty:           cmd.Qty,
		OrderType:     orderType,
	})
	if err != nil {
		rep.ErrorMsg = err.Error()
		if res.Status != "" {
			rep.Status = res.Status
		}
		return rep
	}
	rep.FilledQty = res.FilledQty
	rep.FilledPrice = res.FilledPrice
	rep.Fee = res.Fee
	rep.Status = res.Status
	return rep
}

func (e *Executor) BuildDelta(ctx context.Context, instanceID uint, symbol string) (protocol.DeltaReport, error) {
	pos, err := e.Broker.Snapshot(ctx, symbol)
	if err != nil {
		return protocol.DeltaReport{}, err
	}
	return protocol.DeltaReport{
		InstanceID: instanceID,
		Symbol:     symbol,
		Cash:       pos.Cash,
		Shares:     pos.Shares,
		TsMs:       time.Now().UnixMilli(),
	}, nil
}

func (e *Executor) SetMarkPrice(symbol string, price float64) {
	e.Broker.SetMarkPrice(symbol, price)
}

func ValidateCommand(cmd protocol.TradeCommand) error {
	if cmd.ClientOrderID == "" {
		return fmt.Errorf("client_order_id required")
	}
	if cmd.InstanceID == 0 {
		return fmt.Errorf("instance_id required")
	}
	if cmd.Symbol == "" {
		return fmt.Errorf("symbol required")
	}
	if cmd.Side != "BUY" && cmd.Side != "SELL" {
		return fmt.Errorf("side must be BUY or SELL")
	}
	if cmd.Qty <= 0 {
		return fmt.Errorf("qty must be > 0")
	}
	return nil
}
