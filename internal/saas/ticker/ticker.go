package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/market"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ws"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategy"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type Config struct {
	Interval      time.Duration
	LookbackBars  int
	Enabled       bool
	AllowDispatch bool
}

type Ticker struct {
	db     *store.DB
	inst   *instance.Service
	market *market.Service
	hub    *ws.Hub
	log    *zap.Logger
	cfg    Config

	mu     sync.Mutex
	stopCh chan struct{}
	done   chan struct{}
}

func New(db *store.DB, inst *instance.Service, hub *ws.Hub, log *zap.Logger, cfg Config) *Ticker {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}
	if cfg.LookbackBars <= 0 {
		cfg.LookbackBars = 120
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Ticker{
		db:     db,
		inst:   inst,
		market: market.New(db),
		hub:    hub,
		log:    log,
		cfg:    cfg,
		stopCh: make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (t *Ticker) Start() {
	if !t.cfg.Enabled {
		t.log.Info("ticker disabled")
		close(t.done)
		return
	}
	go t.loop()
}

func (t *Ticker) Stop() {
	t.mu.Lock()
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
	t.mu.Unlock()
	<-t.done
}

func (t *Ticker) loop() {
	defer close(t.done)
	select {
	case <-t.stopCh:
		return
	case <-time.After(2 * time.Second):
		t.TickOnce(context.Background())
	}
	tk := time.NewTicker(t.cfg.Interval)
	defer tk.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-tk.C:
			t.TickOnce(context.Background())
		}
	}
}

func (t *Ticker) TickOnce(ctx context.Context) {
	list, err := t.inst.ListRunning()
	if err != nil {
		t.log.Warn("list running", zap.Error(err))
		return
	}
	for i := range list {
		if ctx.Err() != nil {
			return
		}
		if err := t.processInstance(ctx, &list[i]); err != nil {
			t.log.Warn("tick instance", zap.Uint("instance_id", list[i].ID), zap.Error(err))
		}
	}
}

func (t *Ticker) processInstance(ctx context.Context, inst *store.StrategyInstance) error {
	bars, err := t.market.ListBars(inst.Symbol, "1m", t.cfg.LookbackBars)
	if err != nil {
		return fmt.Errorf("list bars: %w", err)
	}
	if len(bars) == 0 {
		return fmt.Errorf("no klines for %s", inst.Symbol)
	}
	closes := make([]float64, len(bars))
	times := make([]int64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
		times[i] = b.OpenTime
	}
	price := closes[len(closes)-1]

	portRow, err := t.inst.GetPortfolio(inst.ID)
	if err != nil {
		return err
	}
	port := quant.Portfolio{
		Cash: portRow.CNYBalance, DeadHold: portRow.DeadHold,
		FloatHold: portRow.FloatHold, ColdSealedHold: portRow.ColdSealedHold,
	}

	raw, err := t.inst.LoadChampionParams(inst)
	if err != nil {
		return err
	}
	params := lunar.DefaultParams()
	_ = json.Unmarshal(raw, &params)
	params = params.Clamp()

	var strat strategy.Strategy
	if inst.TemplateID == lunar.StrategyID || inst.TemplateID == "" {
		strat = lunar.New(params)
	} else if s, ok := strategy.Get(inst.TemplateID); ok {
		strat = s
	} else {
		strat = lunar.New(params)
	}

	in := quant.StrategyInput{
		NowMs: time.Now().UnixMilli(), Price: price, Closes: closes, Times: times,
		Portfolio: port, Spawn: quant.DefaultSpawnPoint(), Runtime: map[string]float64{},
		MicroReservePct: params.MicroReservePct, Kp: params.Kp, Kv: params.Kv, Ka: params.Ka,
		MinTradeThreshold: params.MinTradeThreshold,
	}
	var rt store.RuntimeState
	if err := t.db.Where("instance_id = ?", inst.ID).First(&rt).Error; err == nil && len(rt.Snapshot) > 0 {
		_ = json.Unmarshal(rt.Snapshot, &in.Runtime)
	}
	out := strat.Step(in)
	if out.Runtime != nil {
		if snap, err := json.Marshal(out.Runtime); err == nil {
			_ = t.db.Model(&store.RuntimeState{}).Where("instance_id = ?", inst.ID).
				Updates(map[string]any{"snapshot": datatypes.JSON(snap), "updated_at": time.Now()}).Error
		}
	}
	for _, rel := range out.Releases {
		if rel.Qty <= 0 {
			continue
		}
		if err := t.inst.ApplyRelease(inst.ID, rel.Qty, price); err != nil {
			t.log.Warn("apply release", zap.Uint("instance_id", inst.ID), zap.Error(err))
		}
	}
	if !t.cfg.AllowDispatch || t.hub == nil {
		return nil
	}
	for _, ord := range out.Orders {
		if ord.Qty <= 0 {
			continue
		}
		side := string(ord.Side)
		if side != "BUY" && side != "SELL" {
			continue
		}
		engine := ord.Engine
		if engine == "" {
			engine = "MICRO"
		}
		clientOID := fmt.Sprintf("tick-%d-%d-%s", inst.ID, time.Now().UnixMilli(), engine)
		cmd := protocol.TradeCommand{
			ClientOrderID: clientOID, InstanceID: inst.ID, Symbol: inst.Symbol,
			Side: side, Engine: engine, Qty: ord.Qty, OrderType: "MARKET",
		}
		_ = t.inst.RecordPendingExecution(inst.ID, clientOID, inst.Symbol, store.TradeAction(side), ord.Qty)
		if err := t.hub.SendTrade(cmd); err != nil {
			t.log.Warn("send trade", zap.Uint("instance_id", inst.ID), zap.String("client_order_id", clientOID), zap.Error(err))
			continue
		}
		t.log.Info("trade dispatched", zap.Uint("instance_id", inst.ID), zap.String("side", side),
			zap.String("engine", engine), zap.Float64("qty", ord.Qty), zap.String("client_order_id", clientOID))
	}
	return nil
}
