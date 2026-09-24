package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/broker"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/client"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/executor"
	"go.uber.org/zap"
)

// LocalAgent entry: load local config, init broker, resolve SaaS token.
// WebSocket session and full TradeCommand loop land in Phase 8.
// Iron rule: this binary must not import strategy packages.
func main() {
	cfgPath := flag.String("config", "configs/config.agent.yaml", "path to config.agent.yaml")
	flag.Parse()

	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}
	red := cfg.Redacted()
	log.Info("agent starting",
		zap.String("agent_id", red.AgentID),
		zap.String("broker", red.Broker.Driver),
		zap.String("saas", red.SaaS.BaseURL),
		zap.Uints("instances", red.Instances),
	)

	var b broker.Broker
	switch cfg.Broker.Driver {
	case "paper", "":
		b = broker.NewPaper(broker.PaperOpts{
			InitialCash:    cfg.Broker.InitialCash,
			CommissionRate: cfg.Broker.CommissionRate,
			StampTaxRate:   cfg.Broker.StampTaxRate,
			LotStep:        cfg.Broker.LotStep,
			LotMin:         cfg.Broker.LotMin,
		})
	default:
		log.Fatal("unsupported broker driver (Phase 7 ships paper only)",
			zap.String("driver", cfg.Broker.Driver))
	}

	ex := executor.New(b, cfg.AgentID)
	_ = ex // used by Phase 8 WS session loop

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	token, err := client.ResolveToken(ctx, cfg.SaaS.BaseURL, cfg.SaaS.Email, cfg.SaaS.Password, cfg.SaaS.Token)
	if err != nil {
		log.Warn("saas token not resolved; running offline paper mode", zap.Error(err))
	} else {
		log.Info("saas token ready", zap.Int("token_len", len(token)))
	}

	log.Info("agent ready (Phase 7); waiting for Phase 8 WS session")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("agent shutdown")
}
