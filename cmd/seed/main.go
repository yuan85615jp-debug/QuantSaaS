package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/market"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"go.uber.org/zap"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config.yaml")
	symbol := flag.String("symbol", "510300", "symbol to seed")
	bars := flag.Int("bars", 200, "number of bars")
	startPx := flag.Float64("start-px", 4.50, "starting price (synthetic only)")
	users := flag.Bool("users", false, "also create demo user + agent accounts")
	real := flag.Bool("real", false, "fetch real klines from eastmoney instead of synthetic")
	interval := flag.String("interval", "1m", "bar interval when -real")
	flag.Parse()

	log, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}
	db, err := store.NewDB(cfg, log)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer db.Close() //nolint:errcheck

	mkt := market.New(db)
	var n int
	if *real {
		prov := market.NewEastMoney()
		barsIn, err := prov.FetchKlines(context.Background(), *symbol, *interval, *bars)
		if err != nil {
			log.Fatal("fetch real klines", zap.Error(err))
		}
		n, err = mkt.ImportBars(*symbol, *interval, barsIn)
		if err != nil {
			log.Fatal("import real klines", zap.Error(err))
		}
		log.Info("real klines imported", zap.String("provider", prov.Name()), zap.Int("bars", n))
	} else {
		n, err = mkt.SeedSynthetic(market.SeedOpts{
			Symbol: *symbol, Bars: *bars, StartPx: *startPx,
		})
		if err != nil {
			log.Fatal("seed klines", zap.Error(err))
		}
	}
	px, ts, _ := mkt.LastClose(*symbol)
	log.Info("klines seeded",
		zap.String("symbol", *symbol),
		zap.Int("bars", n),
		zap.Float64("last_close", px),
		zap.Int64("last_open_time", ts),
	)

	if *users {
		authSvc := auth.NewService(cfg)
		us := auth.NewUserService(db, authSvc)
		for _, u := range []struct {
			email, pass, role, name string
		}{
			{"demo@quantsaas.local", "demo1234", "user", "Demo User"},
			{"agent@quantsaas.local", "agent1234", "agent", "Paper Agent"},
		} {
			_, err := us.Register(auth.RegisterRequest{
				Email: u.email, Password: u.pass, DisplayName: u.name, Role: u.role,
			})
			if err != nil {
				log.Warn("register", zap.String("email", u.email), zap.Error(err))
			} else {
				log.Info("user created", zap.String("email", u.email), zap.String("role", u.role))
			}
		}
	}
}
