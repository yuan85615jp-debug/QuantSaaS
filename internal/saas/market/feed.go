package market

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"go.uber.org/zap"
)

type FeedConfig struct {
	Enabled  bool
	Provider string
	Symbols  []string
	Interval string
	Limit    int
	Every    time.Duration
}

type SymbolSource interface {
	MarketSymbols() ([]string, error)
}

type Feed struct {
	svc    *Service
	prov   Provider
	src    SymbolSource
	log    *zap.Logger
	cfg    FeedConfig
	stopCh chan struct{}
	done   chan struct{}
	mu     sync.Mutex
}

func NewFeed(svc *Service, prov Provider, src SymbolSource, log *zap.Logger, cfg FeedConfig) *Feed {
	if cfg.Interval == "" {
		cfg.Interval = "1m"
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 120
	}
	if cfg.Every <= 0 {
		cfg.Every = time.Minute
	}
	if log == nil {
		log = zap.NewNop()
	}
	if prov == nil {
		prov = NewEastMoney()
	}
	return &Feed{
		svc: svc, prov: prov, src: src, log: log, cfg: cfg,
		stopCh: make(chan struct{}), done: make(chan struct{}),
	}
}

func (f *Feed) Start() {
	if !f.cfg.Enabled {
		f.log.Info("market feed disabled")
		close(f.done)
		return
	}
	f.log.Info("market feed starting",
		zap.String("provider", f.prov.Name()),
		zap.String("interval", f.cfg.Interval),
		zap.Duration("every", f.cfg.Every),
		zap.Strings("symbols", f.cfg.Symbols),
	)
	go f.loop()
}

func (f *Feed) Stop() {
	f.mu.Lock()
	select {
	case <-f.stopCh:
	default:
		close(f.stopCh)
	}
	f.mu.Unlock()
	<-f.done
}

func (f *Feed) loop() {
	defer close(f.done)
	f.SyncOnce(context.Background())
	tk := time.NewTicker(f.cfg.Every)
	defer tk.Stop()
	for {
		select {
		case <-f.stopCh:
			return
		case <-tk.C:
			f.SyncOnce(context.Background())
		}
	}
}

func (f *Feed) SyncOnce(ctx context.Context) map[string]int {
	symbols := f.resolveSymbols()
	out := map[string]int{}
	for _, sym := range symbols {
		if ctx.Err() != nil {
			break
		}
		n, err := f.SyncSymbol(ctx, sym, f.cfg.Interval, f.cfg.Limit)
		if err != nil {
			f.log.Warn("market sync failed", zap.String("symbol", sym), zap.String("provider", f.prov.Name()), zap.Error(err))
			out[sym] = -1
			continue
		}
		out[sym] = n
		f.log.Info("market synced", zap.String("symbol", sym), zap.Int("bars", n), zap.String("interval", f.cfg.Interval))
	}
	return out
}

func (f *Feed) SyncSymbol(ctx context.Context, symbol, interval string, limit int) (int, error) {
	if interval == "" {
		interval = f.cfg.Interval
	}
	if limit <= 0 {
		limit = f.cfg.Limit
	}
	bars, err := f.prov.FetchKlines(ctx, symbol, interval, limit)
	if err != nil {
		return 0, err
	}
	return f.svc.ImportBars(symbol, interval, bars)
}

func (f *Feed) resolveSymbols() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range f.cfg.Symbols {
		add(s)
	}
	if f.src != nil {
		if extra, err := f.src.MarketSymbols(); err == nil {
			for _, s := range extra {
				add(s)
			}
		}
	}
	return out
}

type InstanceSymbolSource struct {
	DB *store.DB
}

func (s InstanceSymbolSource) MarketSymbols() ([]string, error) {
	if s.DB == nil {
		return nil, nil
	}
	var symbols []string
	err := s.DB.Model(&store.StrategyInstance{}).
		Where("status = ?", store.InstanceRunning).
		Distinct().
		Pluck("symbol", &symbols).Error
	return symbols, err
}
