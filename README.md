# QuantSaaS

全天候智能量化管理工具 —— SaaS + LocalAgent + GA 进化 三端架构。

## 架构一览

| 端 | 角色 | 职责 |
|----|------|------|
| **saas** | 云端决策大脑 | 执行 `Step()`、下发 TradeCommand、管理实例与用户，**不持有任何券商 API Key** |
| **agent** | 用户本地执行手 | 接收指令、调用券商 API 下单、上报 DeltaReport，**不含任何策略代码** |
| **lab** | 本地算力实验室 | 跑 GA 进化与回测，共享同一 Postgres，**不下发真实交易** |

## 核心铁律

1. 回测与实盘调用**同一个** `Step()` 实现
2. `Step()` 是纯函数：禁止网络、DB、文件 I/O、定时器
3. 券商 API Key **只存在于** `config.agent.yaml`
4. GORM Code-First + AutoMigrate，无 SQL migration 文件
5. 价格计算优先无量纲（对数收益率 / 比率）

## 文档真源

- [策略数学引擎](docs/策略数学引擎.md)
- [进化计算引擎](docs/进化计算引擎.md)

## 开发进度

- [x] Phase 0 — 环境初始化
- [x] Phase 1 — 真源文档
- [x] Phase 2 — Config + DB + Auth
- [x] **Phase 3 — 量化数学基础层 (`internal/quant`)**
  - `math.go` — 对数收益、EMA/SMA、MaxDD、ROI
  - `bar.go` — OHLCV
  - `portfolio.go` — 仓位三态与权益公式
  - `spawn.go` — 出生点（资金/费率/手数）
  - `sigmoid.go` — Sigmoid 目标权重 + PDE 信号
  - `dca.go` — Ghost DCA 基准
  - `intent.go` — StrategyInput / StrategyOutput
- [ ] Phase 4 — 策略模块 Step()
- [ ] Phase 5 — GA 进化引擎
- [ ] Phase 6–13 — 实例 / Agent / WS / API / 前端 / Docker

## 本地验证

```bash
export QS_JWT_SECRET=dev-secret-change-me
go test ./internal/saas/auth/ ./internal/quant/ -count=1
```

## License

Private / All Rights Reserved
