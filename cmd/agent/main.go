package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/broker"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/client"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/executor"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/wsclient"
	"go.uber.org/zap"
)

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
			InitialCash: cfg.Broker.InitialCash, CommissionRate: cfg.Broker.CommissionRate,
			StampTaxRate: cfg.Broker.StampTaxRate, LotStep: cfg.Broker.LotStep, LotMin: cfg.Broker.LotMin,
		})
	case "live":
		b = broker.NewLive(broker.LiveOpts{
			BaseURL: cfg.Broker.BaseURL, APIKey: cfg.Broker.APIKey, APISecret: cfg.Broker.APISecret,
			AccountID: cfg.Broker.AccountID, LotStep: cfg.Broker.LotStep, LotMin: cfg.Broker.LotMin,
			DryRun: cfg.Broker.DryRun,
		})
		log.Info("live broker selected", zap.String("name", b.Name()), zap.Bool("dry_run", cfg.Broker.DryRun || cfg.Broker.APIKey == ""))
	default:
		log.Fatal("unsupported broker driver", zap.String("driver", cfg.Broker.Driver))
	}

	ex := executor.New(b, cfg.AgentID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	token, err := client.ResolveToken(ctx, cfg.SaaS.BaseURL, cfg.SaaS.Email, cfg.SaaS.Password, cfg.SaaS.Token)
	if err != nil {
		log.Warn("saas token not resolved; WS disabled (offline paper only)", zap.Error(err))
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		return
	}

	ws := &wsclient.Client{
		BaseURL: cfg.SaaS.BaseURL, Token: token, AgentID: cfg.AgentID, InstanceIDs: cfg.Instances,
		Executor: ex, Log: log,
		MarkPriceProvider: func(symbol string) float64 {
			px, err := client.LastClose(context.Background(), cfg.SaaS.BaseURL, token, symbol)
			if err != nil {
				log.Debug("mark price fetch failed", zap.String("symbol", symbol), zap.Error(err))
				return 0
			}
			return px
		},
	}
	go ws.Run(ctx)
	log.Info("agent WS session started")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	log.Info("agent shutdown")
}
