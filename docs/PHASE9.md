# Phase 9 — REST API + cmd/saas

## 职责

1. 提供用户 / Agent 登录与策略实例 CRUD REST API
2. `cmd/saas` 主入口：加载配置、DB、JWT、Hub、HTTP、WS
3. 将 WS `FillReport` 正式接到 `instance.Service.ApplyFill`

## 模块

| 路径 | 职责 |
|------|------|
| `cmd/saas` | SaaS 主进程：HTTP + `/ws/agent` |
| `internal/saas/api` | REST handlers、JWT middleware |
| `internal/saas/auth` | 密码哈希、Register / Login / AgentLogin |
| `internal/saas/ws/report.go` | `FillBridge`：OnFill → ApplyFill |

## API

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/healthz` | 无 | 健康检查 |
| POST | `/api/v1/auth/register` | 无 | 注册 user |
| POST | `/api/v1/auth/login` | 无 | 登录拿 JWT |
| POST | `/api/v1/agent/login` | 无 | Agent 登录（role=agent\|admin） |
| GET | `/api/v1/instances` | Bearer | 列出当前用户实例 |
| POST | `/api/v1/instances` | Bearer | 创建实例 |
| GET | `/api/v1/instances/{id}` | Bearer | 查询实例 |
| POST | `/api/v1/instances/{id}/start` | Bearer | 启动 |
| POST | `/api/v1/instances/{id}/stop` | Bearer | 停止 |
| GET | `/api/v1/instances/{id}/portfolio` | Bearer | 三态账本 |
| POST | `/api/v1/instances/{id}/trades` | Bearer | 下发 TradeCommand（需在线 Agent） |
| GET | `/ws/agent` | token query 或 Bearer | Agent WebSocket |

### 创建实例 body

```json
{"template_id":"lunar","symbol":"510300","capital_quota":100000}
```

### 下发交易 body

```json
{"side":"BUY","engine":"MICRO","qty":100,"order_type":"MARKET","client_order_id":"optional"}
```

`app_role: lab` 时 `/trades` 返回 403。

## Fill 接线

```
Agent FillReport → Hub.handleInbound → FillBridge.OnFill
  → filled/partial: instance.ApplyFill(MACRO→Dead, MICRO→Float)
  → failed: instance.MarkError
```

DeltaReport 仅日志；Dead/Float 语义只在 SaaS。

## 运行

```bash
export QS_JWT_SECRET=dev-secret-change-me
# Postgres 需可达，或改 store 测试用 sqlite
go run ./cmd/saas -config configs/config.yaml
```

## 验证

```bash
go test ./internal/saas/auth/ ./internal/saas/api/ ./internal/saas/ws/ ./internal/saas/instance/ -count=1
go build -o bin/saas ./cmd/saas
go build -o bin/agent ./cmd/agent
```

## 明确留给后续

- cron tick 调 Step() 并自动 SendTrade
- 前端
- Docker Compose
- TradeRecord / SpotExecution 落库与幂等 client_order_id
