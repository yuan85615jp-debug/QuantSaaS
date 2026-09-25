#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CFG="${1:-$ROOT/configs/config.agent.yaml}"
if [[ ! -f "$CFG" ]]; then
  echo "missing $CFG — run scripts/demo_paper.sh first (it writes the file)" >&2
  exit 1
fi
cd "$ROOT"
if [[ ! -x bin/agent ]]; then
  go build -o bin/agent ./cmd/agent
fi
exec ./bin/agent -config "$CFG"
