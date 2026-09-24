# QuantSaaS

全天候智能量化管理工具 —— SaaS + LocalAgent + GA 进化 三端架构。

## 架构一览

| 端 | 角色 | 职责 |
|----|------|------|
| **saas** | 云端决策大脑 | 执行 `Step()`、下发指令 |
| **agent** | 本地执行手 | 券商下单，无策略代码 |
| **lab** | 算力实验室 | GA 进化与回测 |

## 开发进度

- [x] Phase 0–8 — 核心引擎、实例、Agent、WebSocket
- [x] Phase 9 — REST API + cmd/saas + FillBridge
- [x] TradeRecord 落库与 client_order_id 幂等
- [x] **Phase 10 — Ticker（cron Step 自动下单）**
- [x] **Phase 11 — 前端 SPA（embed）**
- [x] **Phase 12 — Docker Compose**
- [x] **Phase 13 — K 线喂入 + Paper Demo**

## 本地验证

```bash
export QS_JWT_SECRET=dev-secret-change-me

go test ./internal/saas/... ./internal/agent/... ./internal/quant/... -count=1
go build -o bin/saas ./cmd/saas
go build -o bin/agent ./cmd/agent
go build -o bin/seed ./cmd/seed
```

## 启动

```bash
export QS_JWT_SECRET=dev-secret-change-me
export QS_DB_PASSWORD=...
go run ./cmd/saas -config configs/config.yaml

go run ./cmd/seed -config configs/config.yaml -users
./scripts/demo_paper.sh
```

## Docker

```bash
docker compose up --build
```

详见 `docs/PHASE13.md`。

## License

Private / All Rights Reserved
