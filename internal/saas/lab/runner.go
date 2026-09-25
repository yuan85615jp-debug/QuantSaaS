package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ga"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/market"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type GARunner struct {
	DB     *store.DB
	Market *market.Service
	Log    *zap.Logger
}

type taskConfig struct {
	Interval       string  `json:"interval"`
	Bars           int     `json:"bars"`
	PopSize        int     `json:"pop_size"`
	MaxGenerations int     `json:"max_generations"`
	TestMode       bool    `json:"test_mode"`
	Promote        bool    `json:"promote"`
	Seed           int64   `json:"seed"`
	MutationProb   float64 `json:"mutation_prob"`
}

func (r *GARunner) Run(ctx context.Context, task *store.EvolutionTask) (*RunResult, error) {
	if r.DB == nil || r.Market == nil {
		return nil, fmt.Errorf("lab runner missing deps")
	}
	cfg := taskConfig{
		Interval: "1d", Bars: 250, PopSize: 24, MaxGenerations: 8,
		Promote: true, Seed: time.Now().UnixNano(),
	}
	if len(task.Config) > 0 {
		_ = json.Unmarshal(task.Config, &cfg)
	}
	if cfg.Bars <= 0 {
		cfg.Bars = 250
	}
	if cfg.Interval == "" {
		cfg.Interval = "1d"
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	bars, err := r.Market.ListBars(task.Symbol, cfg.Interval, cfg.Bars)
	if err != nil {
		return nil, fmt.Errorf("list bars: %w", err)
	}
	if len(bars) < 30 {
		return nil, fmt.Errorf("need at least 30 bars for %s %s, got %d (sync market or seed first)", task.Symbol, cfg.Interval, len(bars))
	}
	closes := make([]float64, len(bars))
	times := make([]int64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
		times[i] = b.OpenTime
	}
	gaCfg := ga.DefaultConfig()
	if cfg.PopSize > 0 {
		gaCfg.PopSize = cfg.PopSize
	}
	if cfg.MaxGenerations > 0 {
		gaCfg.MaxGenerations = cfg.MaxGenerations
	}
	if cfg.MutationProb > 0 {
		gaCfg.MutationProb = cfg.MutationProb
	}
	gaCfg.TestMode = cfg.TestMode
	engine := ga.NewEngine(ga.LunarEvolvable{}, gaCfg, cfg.Seed)
	result := engine.Run(closes, times)
	params := lunar.FromSlice(result.Best.Gene)
	paramRaw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	gene := store.GeneRecord{
		StrategyID: task.StrategyID, Symbol: task.Symbol, Role: store.GeneChallenger,
		ParamPack: datatypes.JSON(paramRaw), ScoreTotal: result.Best.Score, MaxDrawdown: result.Best.MaxDD,
	}
	if len(result.Best.WindowScores) > 0 {
		if ws, err := json.Marshal(result.Best.WindowScores); err == nil {
			gene.WindowScores = datatypes.JSON(ws)
		}
	}
	err = r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&gene).Error; err != nil {
			return err
		}
		if cfg.Promote {
			if err := tx.Model(&store.GeneRecord{}).
				Where("strategy_id = ? AND symbol = ? AND role = ?", task.StrategyID, task.Symbol, store.GeneChampion).
				Update("role", store.GeneRetired).Error; err != nil {
				return err
			}
			now := time.Now()
			if err := tx.Model(&gene).Updates(map[string]any{
				"role": store.GeneChampion, "activated_at": &now,
			}).Error; err != nil {
				return err
			}
			gene.Role = store.GeneChampion
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	hist, _ := json.Marshal(result.History)
	return &RunResult{
		GeneID: gene.ID, Score: result.Best.Score, MaxDrawdown: result.Best.MaxDD,
		Generations: result.Generation, ParamPack: paramRaw, History: hist,
	}, nil
}
