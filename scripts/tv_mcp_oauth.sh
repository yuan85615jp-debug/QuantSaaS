#!/usr/bin/env bash
# TradingView MCP OAuth 引导（包装 go run ./cmd/tvimport -oauth）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
exec go run ./cmd/tvimport -oauth "$@"
