# cmd/tvimport

TradingView official MCP → QuantSaaS `klines/import`.

## Features

- `-oauth` — OAuth 2.1 + PKCE (dynamic client registration, local callback `:8765`)
- Multi-symbol: `-symbols`, `-symbols-file`, `-dir`
- Token file: `~/.config/quantsaas/tv_mcp_token.json` (auto refresh when possible)

## Docs

See [docs/TVIMPORT.md](../../docs/TVIMPORT.md).

## Quick start

```bash
./scripts/tv_mcp_oauth.sh
go run ./cmd/tvimport -symbols-file scripts/symbols.example.txt -interval 1D -count 200
```

## Source layout

| File | Role |
|------|------|
| `main.go` | CLI, batch loop, MCP get_ohlcv, SaaS import |
| `oauth_types.go` | AS metadata, register, PKCE, token load/save |
| `oauth_guide.go` | Interactive `-oauth` browser flow |
| `oauth_token.go` | code/refresh token exchange |
| `main_test.go` | unit tests |

If `main.go` / oauth sources lag on `main`, apply the zip from the project artifacts (`tvimport-oauth-batch.zip`) or pull the local commit that added OAuth+batch.
