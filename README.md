# QuantSaaS

全天候智能量化管理工具 —— SaaS + LocalAgent + GA 进化 三端架构。

## 架构一览

| 端 | 角色 | 职责 |
|----|------|------|
| **saas** | 云端决策大脑 | 执行 `Step()`、下发指令 |
| **agent** | 本地执行手 | 券商下单，无策略代码 |
| **lab** | 算力实验室 | GA 进化与回测 |

## 开发进度

- [x] Phase 0–1 — 环境与文档
- [x] Phase 2 — Config + DB + Auth
- [x] Phase 3 — `internal/quant`
- [x] Phase 4 — `lunar` Step()
- [x] Phase 5 — GA 进化引擎
- [x] Phase 6 — 策略实例生命周期
- [x] Phase 7 — LocalAgent
- [x] **Phase 8 — WebSocket 通道**
  - `internal/saas/ws` — Hub / Session / JWT Upgrade
  - `internal/agent/wsclient` — 重连、心跳、接单回填
  - protocol codec（JSON Wire）
- [ ] Phase 9–13 — API / 前端 / Docker

## 本地验证

```bash
go test ./internal/quant/ ./internal/strategies/lunar/ ./internal/saas/ga/ ./internal/saas/instance/ ./internal/agent/... ./internal/saas/ws/ -count=1
go build -o bin/agent ./cmd/agent
```

## License

Private / All Rights Reserved
