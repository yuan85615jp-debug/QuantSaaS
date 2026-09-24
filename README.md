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
- [x] **Phase 5 — GA 进化引擎**
  - `internal/adapters/backtest` — 单路径回测（手续费/手数）
  - `internal/saas/ga` — 种群、锦标赛、交叉、变异斜坡
  - `LunarEvolvable` — 四窗 Alpha vs Ghost DCA 适应度
- [ ] Phase 6–13 — 实例 / Agent / WS / API / 前端 / Docker

## 本地验证

```bash
go test ./internal/quant/ ./internal/strategies/lunar/ ./internal/saas/ga/ -count=1
```

快速进化（TestMode：Pop=10, Gen=3）见 `engine_test.go`。

## License

Private / All Rights Reserved
