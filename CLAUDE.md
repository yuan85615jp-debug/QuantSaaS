# QuantSaaS — AI 协作宪法

## 唯一功能真源

当前功能只依据 `docs/` 下的三份文档：

1. `docs/系统总体拓扑结构.md` — 物理端、逻辑模块、状态流转、生命周期
2. `docs/策略数学引擎.md` — Step() 输入输出契约、宏观/微观引擎、仓位三态
3. `docs/进化计算引擎.md` — GA 黑盒、适应度、EvolvableStrategy 接口

**三份文档没有定义的功能不进入实现。** 如有歧义，先补文档再写代码。

## 工作顺序

1. 涉及策略和回测：先读 `策略数学引擎.md` 与对应策略文档
2. 涉及 Go 后端：遵守 GORM Code-First，只用 AutoMigrate，永不写 SQL migration
3. 涉及价格计算：优先无量纲表达（对数收益率、比率），禁止跨标的绝对价格比较
4. 涉及架构边界：保持 SaaS–Strategy–Agent 分工，不做预防性解耦

## 核心约束（铁律）

1. **复利前置条件**：任何策略机制必须能清楚说明复利如何发生（资金规模随权益正反馈滚动）
2. **策略同构**：回测与实盘必须调用同一个 `Step()` 实现，内部禁止 `if isBacktest` 分支
3. **Step() 只在 SaaS 侧执行**：Agent 二进制不包含任何策略代码
4. **策略纯函数**：策略包内部禁止定时器、网络请求、数据库读写、任何文件 I/O
5. **API Key 物理隔离**：券商 API 凭证只允许存在于 `config.agent.yaml`，永不进入 SaaS、永不写入数据库、永不通过网络传输

## 代码目录职责

| 路径 | 职责 |
|------|------|
| `cmd/saas/` | SaaS 主入口（app_role: saas / lab / dev） |
| `cmd/agent/` | LocalAgent 极简入口 |
| `internal/saas/` | SaaS 业务：实例、cron、WS Hub、REST、进化调度 |
| `internal/agent/` | Agent：登录、WS 客户端、券商适配、DeltaReport |
| `internal/strategy/` | 策略公共契约（StrategyInput/Output、Manifest） |
| `internal/strategies/[name]/` | 具体策略实现（纯 Step() + 参数结构） |
| `internal/quant/` | 无量纲数学原语、Bar、SpawnPoint、仓位三态类型 |
| `internal/adapters/backtest/` | 回测适配器（把 K 线喂给 Step()） |
| `internal/saas/ga/` | GA 引擎 + EvolvableStrategy 实现 |

## 验证命令

```bash
go list ./...
go test ./...
```

## 不可推翻的技术决策（完整列表见拓扑文档第 7 章）

- GORM AutoMigrate 为唯一 schema 真源
- 单一 Postgres，Redis 仅缓存
- 仓位三态：DeadStack / FloatStack / ColdSealedStack
- 底仓释放只在 SaaS 侧更新账本，不下发 TradeCommand
- 用户界面零数学裸术语
