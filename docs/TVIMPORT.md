# TV MCP → klines/import

独立小工具 `cmd/tvimport`：从 **TradingView 官方 MCP** 的 `get_ohlcv` 拉 K 线，写入 QuantSaaS `POST /api/v1/klines/import`。

- 默认场景：**日线、单标的**
- 不进入 SaaS 热路径；由开发机 / 运维手动或 cron 执行

官方文档：https://www.tradingview.com/mcp/docs  
MCP URL：`https://mcp.tradingview.com/mcp`（需 Essential+，OAuth 2.1）

## 前置

1. QuantSaaS 已启动，且有可登录账号  
2. TradingView **Essential 及以上**（试用套餐不含 MCP）  
3. 拿到 MCP OAuth **access token**（见下），或先把 `get_ohlcv` 结果存成 JSON 文件

## 用法

### A. 在线拉取（有 token）

```bash
export QS_BASE_URL=http://127.0.0.1:8080
export QS_EMAIL=demo@quantsaas.local
export QS_PASSWORD=demo1234
export TRADINGVIEW_MCP_TOKEN='<oauth-access-token>'

go run ./cmd/tvimport \
  -tv-symbol SSE:510300 \
  -qs-symbol 510300 \
  -interval 1D \
  -count 300
```

- `-tv-symbol`：TradingView 路由符号 `EXCHANGE:TICKER`  
- `-qs-symbol`：写入 `k_lines` 的代码（默认取 `:` 后半段）  
- `-interval 1D` → 库内 `1d`  
- `-dry-run`：只解析，不 POST

### B. 离线文件（无 token）

在 Claude / Cursor 里接好官方 MCP，让助手执行 `get_ohlcv`，把返回的 bars JSON 存为文件：

```json
[
  {"t": 1704067200, "o": 4.5, "h": 4.6, "l": 4.4, "c": 4.55, "v": 100000}
]
```

```bash
go run ./cmd/tvimport \
  -file /tmp/nvda_1d.json \
  -qs-symbol AAPL \
  -interval 1D
```

也支持 MCP JSON-RPC / `content[].text` 信封，工具会尽量解包。

## 限制（官方 MCP）

| 项 | 值 |
|----|-----|
| 单次 bars | ≤ 5000 |
| 限流 | ~100 req/min/user |
| 鉴权 | 用户 OAuth，非服务账号 API Key |
| 套餐 | Essential+ |

因此适合 **按需回填 / 研究导入**，不适合替代 `market.Feed` 多标的分钟轮询。

## 与现有链路

```text
tvimport → /api/v1/klines/import → k_lines → Ticker / Lab / Agent mark
```

不改 Agent 铁律；不发 TradeCommand。

## 符号提示

| 市场 | 示例 |
|------|------|
| 上交所 ETF | `SSE:510300` |
| 深交所 | `SZSE:159915` |
| 美股 | `NASDAQ:AAPL` |
| 加密 | `BINANCE:BTCUSDT` |

以 `search_symbols`（官方 MCP）或 TradingView 搜索结果为准。
