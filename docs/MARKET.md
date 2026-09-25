# 真实行情喂入

## 数据源

默认 **东方财富 push2his**（无需 API Key），覆盖 A 股 / ETF：

| 代码示例 | 市场 | secid |
|---------|------|-------|
| 510300 | 上交所 ETF | `1.510300` |
| 159915 | 深交所 ETF | `0.159915` |
| 600519 | 上交所股票 | `1.600519` |

支持周期：`1m` / `5m` / `15m` / `30m` / `1h` / `1d`

## 配置

`configs/config.yaml`：

```yaml
market:
  enabled: true
  provider: eastmoney
  symbols: ["510300"]
  interval: "1m"
  limit: 120
  every_sec: 60
```

环境变量覆盖：

- `QS_MARKET_ENABLED=true|false`
- `QS_MARKET_SYMBOLS=510300,159915`
- `QS_MARKET_INTERVAL=1m`

SaaS 启动后会：

1. 立即拉一次配置里的 symbols
2. 合并所有 `RUNNING` 实例的 symbol
3. 按 `every_sec` 轮询并 upsert 到 `k_lines`

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/klines/sync` | 手动同步；body 可带 `symbol`/`interval`/`limit`，空 body 同步全部 |
| GET | `/api/v1/klines/last?symbol=` | 最新收盘（Agent mark / Ticker） |
| GET | `/api/v1/klines?symbol=&interval=&limit=` | 列表 |

## CLI

```bash
# 真实 K 线写入 DB
go run ./cmd/seed -config configs/config.yaml -real -symbol 510300 -bars 200 -interval 1m
```

## 与 Paper Demo

```bash
curl -X POST http://127.0.0.1:8080/api/v1/klines/sync \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"510300","interval":"1m","limit":120}'
```

Ticker / Agent 的 mark price 均读本地 `k_lines`，无需改执行链路。
