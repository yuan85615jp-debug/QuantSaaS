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

- [系统总体拓扑结构](docs/系统总体拓扑结构.md)（待完整上传）
- [策略数学引擎](docs/策略数学引擎.md) — **已增强**：宏观老农定投基因 + PDE 狙击手（kp/kv/ka）+ 库存桥
- [进化计算引擎](docs/进化计算引擎.md) — **已增强**：三层冻结（Environment/Season/Genes）、正交交叉、全悲观摩擦、MC 终审、1-4-5 种群

## 开发进度（按 Plan Phase）

- [x] Phase 0 — 环境初始化与 AI 协作基础设施
- [x] Phase 1 — 三份真源文档（骨架 + GA 模块增强整合）
- [ ] Phase 2 — 基础设施层（Config + DB + Auth）
- [ ] Phase 3 — 量化数学基础层
- [ ] Phase 4 — 策略模块 Step()
- [ ] Phase 5 — GA 进化引擎
- [ ] Phase 6 — 实例生命周期 + Cron
- [ ] Phase 7 — LocalAgent
- [ ] Phase 8 — WebSocket Hub
- [ ] Phase 9 — REST API
- [ ] Phase 10 — 系统入口
- [ ] Phase 11 — 测试与验证
- [ ] Phase 12 — Web 前端
- [ ] Phase 13 — Docker 部署

## 快速开始（开发中）

```bash
go mod download
go list ./...
```

## License

Private / All Rights Reserved
