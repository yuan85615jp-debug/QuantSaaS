# Live Broker 适配骨架

Agent 通过 `broker.Broker` 接口下单。实现：

| Driver | 包 | 说明 |
|--------|-----|------|
| `paper` | `internal/agent/broker/paper.go` | 本地模拟成交 |
| `live` | `internal/agent/broker/live.go` | HTTP 网关骨架（vendor 待填） |

## 配置（config.agent.yaml）

```yaml
broker:
  driver: live
  base_url: "https://broker-gateway.example"
  api_key: ""          # QS_BROKER_API_KEY
  api_secret: ""       # QS_BROKER_API_SECRET
  account_id: ""
  dry_run: true        # 无密钥或 dry_run=true 时拒绝真实下单
  lot_step: 100
  lot_min: 100
```

## 接入真实券商

1. 复制 `LiveBroker.PlaceOrder` / `Snapshot` 中的 PLACEHOLDER 路径与 JSON schema。
2. 将 `X-API-Key` / `X-API-Secret` 换成厂商鉴权（OAuth / 签名）。
3. 映射成交回执到 `OrderResult{FilledQty,FilledPrice,Fee,Status}`。
4. 设置 `dry_run: false` 并提供密钥。

铁律：**密钥只在 Agent 进程内**，永不上传 SaaS。
