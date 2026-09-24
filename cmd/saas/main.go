package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/api"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ticker"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ws"
	_ "github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
	"github.com/yuan85615jp-debug/QuantSaaS/web"
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

	tk := ticker.New(db, instSvc, hub, log, ticker.Config{
		Interval:      time.Minute,
		LookbackBars:  120,
		Enabled:       cfg.AppRole == config.RoleSaaS || cfg.AppRole == config.RoleDev,
		AllowDispatch: cfg.AllowTradeDispatch(),
	})
	tk.Start()
	defer tk.Stop()

	wsHandler := &ws.Handler{Hub: hub, Auth: authSvc, Log: log}
	apiSrv := &api.Server{
		Users: userSvc, Auth: authSvc, Inst: instSvc, Hub: hub, Cfg: cfg, Log: log,
	}
	apiMux := apiSrv.Routes()

	staticRoot, err := fs.Sub(web.FS, "static")
	if err != nil {
		log.Fatal("web embed", zap.Error(err))
	}
	fileServer := http.FileServer(http.FS(staticRoot))

	root := http.NewServeMux()
	root.Handle("/ws/agent", wsHandler)
	root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/healthz" || strings.HasPrefix(path, "/api/") {
			apiMux.ServeHTTP(w, r)
			return
		}
		if path != "/" && !strings.Contains(path[strings.LastIndex(path, "/")+1:], ".") {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

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
	tk.Stop()
	hub.Close()
	log.Info("saas stopped")
}
