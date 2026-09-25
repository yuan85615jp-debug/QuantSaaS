#!/usr/bin/env bash
# End-to-end paper demo: health → seed → create/start → (optional agent) → trade → portfolio change.
# Requires a running SaaS at SAAS_URL (default http://127.0.0.1:8080).
set -euo pipefail

BASE="${SAAS_URL:-http://127.0.0.1:8080}"
USER_EMAIL="${DEMO_EMAIL:-demo@quantsaas.local}"
USER_PASS="${DEMO_PASSWORD:-demo1234}"
AGENT_EMAIL="${AGENT_EMAIL:-agent@quantsaas.local}"
AGENT_PASS="${AGENT_PASSWORD:-agent1234}"
SYMBOL="${SYMBOL:-510300}"
QTY="${QTY:-100}"
WAIT_SECS="${WAIT_SECS:-30}"
START_AGENT="${START_AGENT:-0}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

json_field() {
  python3 -c "import sys,json; d=json.load(sys.stdin); print(d$1)"
}

echo "== [1/8] health =="
curl -sf "$BASE/healthz" | tee /tmp/qs_health.json
echo

echo "== [2/8] register user/agent (ignore if exists) =="
curl -s -X POST "$BASE/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\",\"display_name\":\"Demo\"}" >/dev/null || true
curl -s -X POST "$BASE/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$AGENT_EMAIL\",\"password\":\"$AGENT_PASS\",\"display_name\":\"Agent\",\"role\":\"agent\"}" >/dev/null || true

echo "== [3/8] login user =="
TOKEN=$(curl -sf -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}" | json_field "['token']")
echo "token_len=${#TOKEN}"

echo "== [4/8] seed klines =="
curl -sf -X POST "$BASE/api/v1/klines/seed" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"symbol\":\"$SYMBOL\",\"bars\":200,\"start_px\":4.5}" | tee /tmp/qs_seed.json
echo
LAST=$(curl -sf "$BASE/api/v1/klines/last?symbol=$SYMBOL" -H "Authorization: Bearer $TOKEN")
echo "$LAST" | tee /tmp/qs_last.json
echo

echo "== [5/8] create + start instance =="
INST=$(curl -sf -X POST "$BASE/api/v1/instances" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"template_id\":\"lunar\",\"symbol\":\"$SYMBOL\",\"capital_quota\":100000}")
echo "$INST" | tee /tmp/qs_inst.json
IID=$(echo "$INST" | json_field "['id']")
echo "instance_id=$IID"

curl -sf -X POST "$BASE/api/v1/instances/$IID/start" \
  -H "Authorization: Bearer $TOKEN" | tee /tmp/qs_start.json
echo

STATUS=$(curl -sf "$BASE/api/v1/instances/$IID" -H "Authorization: Bearer $TOKEN" | json_field "['status']")
echo "status=$STATUS"
if [[ "$STATUS" != "RUNNING" && "$STATUS" != "running" ]]; then
  echo "ERROR: instance not RUNNING" >&2
  exit 1
fi

PORT0=$(curl -sf "$BASE/api/v1/instances/$IID/portfolio" -H "Authorization: Bearer $TOKEN")
echo "$PORT0" | tee /tmp/qs_port0.json
CASH0=$(echo "$PORT0" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('cny_balance', d.get('CNYBalance',0)))")
echo "cash_before=$CASH0"

echo "== [6/8] write agent config =="
AGENT_CFG="${AGENT_CFG:-$ROOT/configs/config.agent.yaml}"
cat > "$AGENT_CFG" <<YAML
agent_id: "agent-demo-1"
saas:
  base_url: "$BASE"
  email: "$AGENT_EMAIL"
  password: "$AGENT_PASS"
  token: ""
broker:
  driver: paper
  commission_rate: 0.0003
  stamp_tax_rate: 0.001
  initial_cash: 100000
  lot_step: 100
  lot_min: 100
instances: [$IID]
YAML
echo "wrote $AGENT_CFG"

AGENT_PID=""
if [[ "$START_AGENT" == "1" ]]; then
  echo "== starting paper agent in background =="
  if [[ -x "$ROOT/bin/agent" ]]; then
    AGENT_BIN="$ROOT/bin/agent"
  else
    (cd "$ROOT" && go build -o bin/agent ./cmd/agent)
    AGENT_BIN="$ROOT/bin/agent"
  fi
  "$AGENT_BIN" -config "$AGENT_CFG" > /tmp/qs_agent.log 2>&1 &
  AGENT_PID=$!
  echo "agent_pid=$AGENT_PID"
  sleep 2
fi

echo "== [7/8] dispatch trade =="
TRADE=$(curl -s -w "\n%{http_code}" -X POST "$BASE/api/v1/instances/$IID/trades" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"side\":\"BUY\",\"engine\":\"MACRO\",\"qty\":$QTY,\"order_type\":\"MARKET\"}")
HTTP=$(echo "$TRADE" | tail -n1)
BODY=$(echo "$TRADE" | sed '$d')
echo "$BODY" | tee /tmp/qs_trade.json
echo "http=$HTTP"
if [[ "$HTTP" == "503" ]]; then
  echo "WARN: no online agent — start agent then re-run trade, or START_AGENT=1"
  echo "  go run ./cmd/agent -config $AGENT_CFG"
  if [[ -n "$AGENT_PID" ]]; then
    kill "$AGENT_PID" 2>/dev/null || true
  fi
  exit 2
fi
if [[ "$HTTP" != "202" && "$HTTP" != "200" ]]; then
  echo "ERROR: trade dispatch failed" >&2
  exit 1
fi

echo "== [8/8] wait for portfolio change =="
ok=0
for i in $(seq 1 "$WAIT_SECS"); do
  PORT=$(curl -sf "$BASE/api/v1/instances/$IID/portfolio" -H "Authorization: Bearer $TOKEN")
  if python3 -c "import json,sys; d=json.loads(sys.argv[1]); dead=float(d.get('dead_hold') or d.get('DeadHold') or 0); flt=float(d.get('float_hold') or d.get('FloatHold') or 0); cash=float(d.get('cny_balance') or d.get('CNYBalance') or 0); print(f't={sys.argv[2]} dead={dead} float={flt} cash={cash}'); sys.exit(0 if (dead>0 or flt>0 or cash < float(sys.argv[3]) - 1) else 1)" "$PORT" "$i" "$CASH0"; then
    echo "$PORT" | tee /tmp/qs_port1.json
    ok=1
    break
  fi
  sleep 1
done

if [[ -n "$AGENT_PID" ]]; then
  kill "$AGENT_PID" 2>/dev/null || true
fi

if [[ "$ok" != "1" ]]; then
  echo "ERROR: portfolio did not change within ${WAIT_SECS}s" >&2
  echo "agent log (if any):"; tail -n 40 /tmp/qs_agent.log 2>/dev/null || true
  exit 1
fi

echo
echo "SUCCESS: instance $IID RUNNING → trade filled → portfolio updated"
echo "  portfolio: $(cat /tmp/qs_port1.json)"
