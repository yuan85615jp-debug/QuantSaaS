# Phase 10–12 — Ticker + Frontend + Docker

## Ticker（cron Step → 自動下單）

路徑：`internal/saas/ticker`

| 項目 | 說明 |
|------|------|
| 觸發 | 每分鐘掃描 `RUNNING` 實例 |
| 資料 | 自 `k_lines` 取最近 N 根 close |
| 決策 | 載入冠軍參數 → `Strategy.Step()` |
| 釋放 | `ReleaseIntent` → `ApplyRelease` |
| 下單 | `OrderIntent` → pending + `Hub.SendTrade` |
| 狀態 | 寫回 `RuntimeState.snapshot` |

`lab` 不下發交易；`saas` / `dev` 啟用。

## 前端

`web/static` embed 進 `cmd/saas`：登入、實例 CRUD、帳本、手動下單。

## Docker

```bash
docker compose up --build
```

## 驗證

```bash
go test ./internal/saas/ticker/ ./internal/saas/instance/ -count=1
go build -o bin/saas ./cmd/saas
```
