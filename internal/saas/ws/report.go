package ws

import (
	"strings"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"go.uber.org/zap"
)

// InstanceFiller is the subset of instance.Service used by the report bridge.
type InstanceFiller interface {
	ApplyFill(instanceID uint, action store.TradeAction, engine store.TradeEngine, qty, price, fee float64) error
	MarkError(id uint, msg string) error
}

// FillBridge implements ReportHandler by applying fills to the SaaS ledger.
type FillBridge struct {
	Inst InstanceFiller
	Log  *zap.Logger
}

func NewFillBridge(inst InstanceFiller, log *zap.Logger) *FillBridge {
	if log == nil {
		log = zap.NewNop()
	}
	return &FillBridge{Inst: inst, Log: log}
}

func (b *FillBridge) OnFill(report protocol.FillReport) {
	status := strings.ToLower(strings.TrimSpace(report.Status))
	if status == "failed" {
		msg := report.ErrorMsg
		if msg == "" {
			msg = "fill failed"
		}
		if err := b.Inst.MarkError(report.InstanceID, msg); err != nil {
			b.Log.Warn("mark error after failed fill", zap.Uint("instance_id", report.InstanceID), zap.Error(err))
		}
		return
	}
	if status != "" && status != "filled" && status != "partial" {
		b.Log.Debug("ignore fill with unknown status", zap.String("status", report.Status), zap.String("client_order_id", report.ClientOrderID))
		return
	}
	if report.FilledQty <= 0 || report.FilledPrice <= 0 {
		b.Log.Debug("ignore zero fill", zap.String("client_order_id", report.ClientOrderID))
		return
	}
	action := store.TradeAction(strings.ToUpper(report.Side))
	engine := store.TradeEngine(strings.ToUpper(report.Engine))
	if action != store.ActionBuy && action != store.ActionSell {
		b.Log.Warn("invalid fill side", zap.String("side", report.Side))
		return
	}
	if engine != store.EngineMacro && engine != store.EngineMicro {
		// default micro if omitted
		engine = store.EngineMicro
	}
	if err := b.Inst.ApplyFill(report.InstanceID, action, engine, report.FilledQty, report.FilledPrice, report.Fee); err != nil {
		b.Log.Warn("ApplyFill failed",
			zap.Uint("instance_id", report.InstanceID),
			zap.String("client_order_id", report.ClientOrderID),
			zap.Error(err),
		)
		_ = b.Inst.MarkError(report.InstanceID, "ApplyFill: "+err.Error())
		return
	}
	b.Log.Info("fill applied",
		zap.Uint("instance_id", report.InstanceID),
		zap.String("client_order_id", report.ClientOrderID),
		zap.String("side", string(action)),
		zap.String("engine", string(engine)),
		zap.Float64("qty", report.FilledQty),
		zap.Float64("price", report.FilledPrice),
	)
}

// OnDelta is informational; Dead/Float semantics live only on SaaS via ApplyFill history.
func (b *FillBridge) OnDelta(report protocol.DeltaReport) {
	b.Log.Debug("delta report",
		zap.Uint("instance_id", report.InstanceID),
		zap.String("symbol", report.Symbol),
		zap.Float64("cash", report.Cash),
		zap.Float64("shares", report.Shares),
	)
}

// Ensure interface compliance at compile time.
var _ ReportHandler = (*FillBridge)(nil)
var _ InstanceFiller = (*instance.Service)(nil)
