package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/api"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ws"
	_ "github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar" // register lunar
	"go.uber.org/zap"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config.yaml")
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
	log.Info("saas starting",
		zap.String("app_role", string(cfg.AppRole)),
		zap.String("http_addr", cfg.Server.HTTPAddr),
	)

	db, err := store.NewDB(cfg, log)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer db.Close() //nolint:errcheck

	authSvc := auth.NewService(cfg)
	userSvc := auth.NewUserService(db, authSvc)
	instSvc := instance.NewService(db)
	if err := instSvc.EnsureTemplates(); err != nil {
		log.Fatal("ensure templates", zap.Error(err))
	}

	bridge := ws.NewFillBridge(instSvc, log)
	hub := ws.NewHub(bridge, log)
	defer hub.Close()

	wsHandler := &ws.Handler{Hub: hub, Auth: authSvc, Log: log}

	apiSrv := &api.Server{
		Users: userSvc,
		Auth:  authSvc,
		Inst:  instSvc,
		Hub:   hub,
		Cfg:   cfg,
		Log:   log,
	}

	root := http.NewServeMux()
	root.Handle("/", apiSrv.Routes())
	root.Handle("/ws/agent", wsHandler)

	httpSrv := &http.Server{
		Addr:              cfg.Server.HTTPAddr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("http listening", zap.String("addr", cfg.Server.HTTPAddr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server", zap.Error(err))
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	hub.Close()
	log.Info("saas stopped")
}
