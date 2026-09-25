# TV MCP → klines/import

独立工具 `cmd/tvimport`：从 **TradingView 官方 MCP** 的 `get_ohlcv` 拉 K 线，写入 QuantSaaS `POST /api/v1/klines/import`。

- 支持：**OAuth 2.1 + PKCE 引导**、**单标的**、**多标的批处理**
- 不进入 SaaS 热路径

官方文档：https://www.tradingview.com/mcp/docs  
MCP：`https://mcp.tradingview.com/mcp`（Essential+）

---

## 1. OAuth 引导（详细）

官方 MCP **没有 API Key**，必须用账号 OAuth。工具已对接公开发现文档：

| 项 | URL |
|----|-----|
| Resource | `https://mcp.tradingview.com/mcp` |
| Resource metadata | `https://mcp.tradingview.com/.well-known/oauth-protected-resource/mcp` |
| AS metadata | `https://www.tradingview.com/.well-known/oauth-authorization-server` |
| Authorize | `https://www.tradingview.com/mcp/oauth/authorize` |
| Token | `https://www.tradingview.com/mcp/oauth/token` |
| Register | `https://www.tradingview.com/mcp/oauth/register` |
| Scopes | `mcp:read` `mcp:tools` |
| PKCE | S256 |

### 一键流程

```bash
# 需要本机可监听 127.0.0.1:8765，并完成浏览器登录
./scripts/tv_mcp_oauth.sh
# 等价：
go run ./cmd/tvimport -oauth
```

步骤说明：

1. 拉取 AS 元数据  
2. **动态客户端注册**（`token_endpoint_auth_method=none`）  
3. 生成 PKCE `code_verifier` / `code_challenge`  
4. 打开浏览器 → TradingView 登录授权（`resource=https://mcp.tradingview.com/mcp`）  
5. 回调 `http://127.0.0.1:8765/callback` 取 `code`  
6. 换取 `access_token` + `refresh_token`  
7. 写入 `~/.config/quantsaas/tv_mcp_token.json`（权限 600）

下次 `tvimport` 会自动读该文件；若 access 将过期且有 refresh，会尝试刷新。

可选：

```bash
go run ./cmd/tvimport -oauth -no-browser          # 只打印 URL
go run ./cmd/tvimport -oauth -token-file /tmp/tv.json
export TV_MCP_TOKEN_FILE=/path/to/token.json
```

### 手工备用（Claude / Cursor）

若动态注册被策略拒绝，仍可用官方客户端完成 OAuth，再把 access token 写入环境变量：

```bash
export TRADINGVIEW_MCP_TOKEN='...'
```

或明文/JSON 写入 `-token-file`。

### 套餐与限流

- Essential 及以上；试用无 MCP  
- ~100 req/min/user；单次 `get_ohlcv` ≤ 5000 bars  

---

## 2. 单标的导入

```bash
export QS_BASE_URL=http://127.0.0.1:8080
export QS_EMAIL=demo@quantsaas.local
export QS_PASSWORD=demo1234

# 已完成 -oauth 后，无需再 export token
go run ./cmd/tvimport \
  -tv-symbol SSE:510300 \
  -qs-symbol 510300 \
  -interval 1D \
  -count 300
```

离线文件：

```bash
go run ./cmd/tvimport -file /tmp/bars.json -qs-symbol 510300 -interval 1D
```

---

## 3. 多标的批处理

### 命令行列表

```bash
go run ./cmd/tvimport \
  -symbols 'SSE:510300,SSE:510500=510500,NASDAQ:AAPL' \
  -interval 1D \
  -count 300 \
  -delay 700ms
```

符号格式：

| 写法 | 含义 |
|------|------|
| `NASDAQ:AAPL` | TV=NASDAQ:AAPL，QS=AAPL |
| `SSE:510300=510300` | 显式映射（推荐） |
| `SSE:510300:ETF510300` | 最后一段为 QS |

### 符号文件

见 [`scripts/symbols.example.txt`](../scripts/symbols.example.txt)：

```bash
go run ./cmd/tvimport -symbols-file scripts/symbols.example.txt -interval 1D -count 200
```

### 离线目录

```bash
# 目录内 510300.json、AAPL.json …
go run ./cmd/tvimport -dir /tmp/bars_json -symbols-file scripts/symbols.example.txt
```

### 批处理行为

- 默认 `-continue`：单标失败继续后续  
- `-delay`：调用间隔，默认 700ms，降低撞限流概率  
- 结束打印 `ok / fail / bars` 汇总  

---

## 4. 与现有链路

```text
-oauth → token 文件
tvimport (batch) → get_ohlcv → /api/v1/klines/import → k_lines → Ticker / Lab
```

不改 Agent 铁律；不发 TradeCommand。生产主行情仍建议 `eastmoney` / 授权 vendor；本工具用于官方 TV 数据回填与研究。
