# Phase 6 — 策略实例生命周期

包路径：`internal/saas/instance`

## 职责边界

本阶段只管理 **SaaS 侧账本与实例状态**，不调用 `Step()`、不下发 TradeCommand、不连 Agent。

| 能力 | 说明 |
|------|------|
| EnsureTemplates | 幂等写入内置模板（lunar） |
| Create | 创建 STOPPED 实例 + Portfolio + Runtime；若有 champion 则绑定 |
| Get / ListByUser | 查询；支持 user 归属校验 |
| Start / Stop | RUNNING ↔ STOPPED；Start 时重新绑定 champion |
| MarkError | 写入 ERROR + last_error |
| GetPortfolio | 读取三态账本 |
| ApplyRelease | DeadHold → FloatHold（库存桥，无 TradeCommand） |
| ApplyFill | Agent 成交回报驱动账本（MACRO→Dead，MICRO→Float） |
| LoadChampionParams | 冠军 ParamPack；无冠军时回落 lunar 默认 |
| PromoteGene | challenger → champion，旧冠军 retired |
| ToQuantPortfolio | store → quant.Portfolio 映射，供后续 Step 调用方使用 |

## 状态机

```
STOPPED ──Start──▶ RUNNING
RUNNING ──Stop───▶ STOPPED
RUNNING ──MarkError──▶ ERROR
ERROR   ──Stop───▶ STOPPED
ERROR   ──Start──▶ RUNNING
```

## 验证

```bash
go test ./internal/saas/instance/ -count=1
```

使用 sqlite 内存库；生产仍走 Postgres AutoMigrate。

## 明确留给后续 Phase

- Phase 7+：LocalAgent、WS、TradeCommand 下发、cron tick 调 Step()
- REST API / 前端 / Docker
