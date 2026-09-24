# Phase 13 — K 線餵入 + Paper Demo

## 目標

讓 Ticker 有價格可讀，並能用 Paper Agent 跑通：

```
seed/import K 線 → Start 實例 → Ticker.Step() → SendTrade → Paper fill → ApplyFillReport
```

## 模組

| 路徑 | 職責 |
|------|------|
| `internal/saas/market` | Import / List / LastClose / SeedSynthetic |
| `internal/saas/api/klines.go` | REST 入口 |
| `cmd/seed` | CLI 種子（可同時建 demo 用戶） |
| `scripts/demo_paper.sh` | curl 端到端腳本 |

## API

| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | `/api/v1/klines/import` | 批量 upsert OHLCV |
| POST | `/api/v1/klines/seed` | 合成隨機漫步（預設 510300 × 200 根 1m） |
| GET | `/api/v1/klines?symbol=&interval=&limit=` | 列表（時間正序） |
| GET | `/api/v1/klines/last?symbol=` | 最新收盤（Agent mark 用） |

唯一鍵：`(symbol, interval, open_time)` — 重複匯入為 upsert。

## CLI

```bash
export QS_JWT_SECRET=dev-secret-change-me
export QS_DB_PASSWORD=quantsaas

go run ./cmd/seed -config configs/config.yaml -symbol 510300 -bars 200 -users
```

## Paper Demo

```bash
docker compose up --build
./scripts/demo_paper.sh
# 另開終端跑 paper agent
```

## 驗證

```bash
go test ./internal/saas/market/ ./internal/saas/api/ -count=1
go build -o bin/saas ./cmd/saas && go build -o bin/seed ./cmd/seed
```
