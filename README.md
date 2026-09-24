# QuantSaaS

全天候智能量化管理工具 —— SaaS + LocalAgent + GA 进化 三端架构。

## 架构一览

| 端 | 角色 | 职责 |
|----|------|------|
| **saas** | 云端决策大脑 | 执行 `Step()`、下发 TradeCommand、管理实例与用户 |
| **agent** | 用户本地执行手 | 调券商 API 下单并上报，**不含策略代码** |
| **lab** | 本地算力实验室 | GA 进化与回测，**不下真实交易** |

## 核心铁律

1. 回测与实盘调用**同一个** `Step()`
2. `Step()` 纯函数：无网络 / DB / 定时器 / 文件 IO
3. 券商 API Key 只在 `config.agent.yaml`
4. GORM Code-First + AutoMigrate
5. 价格计算优先无量纲

## 开发进度

- [x] Phase 0–1 — 环境与文档
- [x] Phase 2 — Config + DB + Auth
- [x] Phase 3 — `internal/quant` 数学层
- [x] **Phase 4 — 策略模块 Step()**
  - `internal/strategy` — Manifest 接口 + 注册表
  - `internal/strategies/lunar` — 默认策略
    - 宏观：EMA 偏离触发 DCA 买入底仓（MACRO）
    - 微观：PDE 信号 + Sigmoid 目标权重（MICRO）
    - 库存桥：加速度超阈值时 Dead→Float（仅账本，不下单）
- [ ] Phase 5 — GA 进化引擎
- [ ] Phase 6–13 — 实例 / Agent / WS / API / 前端 / Docker

## 本地验证

```bash
export QS_JWT_SECRET=dev-secret-change-me
go test ./internal/quant/ ./internal/strategies/lunar/ -count=1
```

## License

Private / All Rights Reserved
