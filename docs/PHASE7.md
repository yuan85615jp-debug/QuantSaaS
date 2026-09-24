# Phase 7 — LocalAgent

包路径：`internal/agent/*`、`cmd/agent`

## 职责边界

LocalAgent 是**纯执行手**：接收 SaaS 下发的 `TradeCommand`，通过券商适配器下单，回传 `FillReport` / `DeltaReport`。

| 铁律 | 落实 |
|------|------|
| Agent 不含策略代码 | `cmd/agent` / `internal/agent` **禁止** import `internal/strategies`、`internal/strategy`、`internal/quant` |
| API Key 物理隔离 | 仅存在于 `config.agent.yaml` / 环境变量；不进 SaaS、不进 DB、不上网传 |
| 底仓释放不下单 | Agent 不处理 ReleaseIntent；只执行买卖 TradeCommand |

## 模块

| 路径 | 职责 |
|------|------|
| `internal/agent/config` | 加载 `config.agent.yaml`，密钥 env 覆盖，Redacted 日志 |
| `internal/agent/protocol` | Hello / Heartbeat / TradeCommand / FillReport / DeltaReport |
| `internal/agent/broker` | `Broker` 接口 + `PaperBroker`（手数截断、手续费、印花税） |
| `internal/agent/executor` | TradeCommand → PlaceOrder → FillReport；Snapshot → DeltaReport |
| `internal/agent/client` | SaaS Agent 登录拿 JWT（`/api/v1/agent/login`，Phase 9 落地） |
| `cmd/agent` | 极简入口：加载配置、初始化 broker、解析 token、等待信号 |
| `configs/config.agent.example.yaml` | 无密钥模板（真实文件 gitignore） |

## 与 Phase 6 的衔接

SaaS 收到 `FillReport` 后调用 `instance.Service.ApplyFill` 更新三态账本。  
`DeltaReport` 仅反映券商侧现金与总持股，Dead/Float 语义只在 SaaS。

## 明确留给 Phase 8

- WebSocket 长连接、消息帧编解码、重连
- SaaS 侧 WS Hub 与 TradeCommand 推送
- 在线会话与 Heartbeat 超时踢除

## 验证

```bash
go test ./internal/agent/... -count=1
go build -o bin/agent ./cmd/agent
```
