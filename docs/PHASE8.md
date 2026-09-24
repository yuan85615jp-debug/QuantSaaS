# Phase 8 — WebSocket 通道

## 职责

在 SaaS 与 LocalAgent 之间建立**长连接**，下发 `TradeCommand`，回传 `FillReport` / `DeltaReport`，并以 Heartbeat 维持会话。

## 模块

| 路径 | 职责 |
|------|------|
| `internal/agent/protocol/codec.go` | Wire JSON 编解码 |
| `internal/saas/ws` | Hub / Session / HTTP Upgrade Handler |
| `internal/agent/wsclient` | Agent 侧客户端：重连、心跳、接单执行 |

## 协议

帧格式：

```json
{"type":"trade_command","request_id":"...","payload":{...}}
```

类型：`hello` / `heartbeat` / `trade_command` / `fill_report` / `delta_report` / `error`

### 鉴权

- Agent 连接：`GET /ws/agent?token=<JWT>&agent_id=<id>`
- 或 `Authorization: Bearer <JWT>`
- JWT `role` 必须为 `agent` 或 `admin`

### 会话

1. Agent 连接后发送 `hello`（含 `instance_ids`）
2. Hub 登记 agent → instances 路由表
3. SaaS 调用 `Hub.SendTrade(cmd)` 推送
4. Agent `executor.HandleTrade` → `fill_report` + 可选 `delta_report`
5. 超过 45s 无心跳则踢除

## 与 Phase 6/7 衔接

- `ReportHandler.OnFill` → 可接入 `instance.Service.ApplyFill`
- Agent 仍不 import 策略包；密钥不入帧

## 明确留给 Phase 9

- REST API（登录、实例 CRUD、触发 SendTrade）
- `cmd/saas` 主入口挂载 Hub

## 验证

```bash
go test ./internal/saas/ws/ ./internal/agent/wsclient/ ./internal/agent/protocol/ -count=1
```
