#!/usr/bin/env bash
# End-to-end paper demo against a running SaaS (default http://127.0.0.1:8080).
set -euo pipefail
BASE="${SAAS_URL:-http://127.0.0.1:8080}"
USER_EMAIL="${DEMO_EMAIL:-demo@quantsaas.local}"
USER_PASS="${DEMO_PASSWORD:-demo1234}"
AGENT_EMAIL="${AGENT_EMAIL:-agent@quantsaas.local}"
AGENT_PASS="${AGENT_PASSWORD:-agent1234}"
SYMBOL="${SYMBOL:-510300}"

echo "== health =="
curl -sf "$BASE/healthz" | tee /tmp/qs_health.json
echo

echo "== register user (ignore if exists) =="
curl -s -X POST "$BASE/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\",\"display_name\":\"Demo\"}" || true
echo

echo "== register agent (ignore if exists) =="
curl -s -X POST "$BASE/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$AGENT_EMAIL\",\"password\":\"$AGENT_PASS\",\"display_name\":\"Agent\",\"role\":\"agent\"}" || true
echo

echo "== login user =="
TOKEN=$(curl -sf -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
echo "token_len=${#TOKEN}"

echo "== seed klines =="
curl -sf -X POST "$BASE/api/v1/klines/seed" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"symbol\":\"$SYMBOL\",\"bars\":200,\"start_px\":4.5}" | tee /tmp/qs_seed.json
echo

echo "== last close =="
curl -sf "$BASE/api/v1/klines/last?symbol=$SYMBOL" \
  -H "Authorization: Bearer $TOKEN" | tee /tmp/qs_last.json
echo

echo "== create instance =="
INST=$(curl -sf -X POST "$BASE/api/v1/instances" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"template_id\":\"lunar\",\"symbol\":\"$SYMBOL\",\"capital_quota\":100000}")
echo "$INST" | tee /tmp/qs_inst.json
IID=$(echo "$INST" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "instance_id=$IID"

echo "== start instance =="
curl -sf -X POST "$BASE/api/v1/instances/$IID/start" \
  -H "Authorization: Bearer $TOKEN" | tee /tmp/qs_start.json
echo

echo "== portfolio =="
curl -sf "$BASE/api/v1/instances/$IID/portfolio" \
  -H "Authorization: Bearer $TOKEN" | tee /tmp/qs_port.json
echo

echo
echo "Next steps:"
echo "  1. Start paper agent with instances: [$IID]"
echo "  2. Wait for ticker or POST /api/v1/instances/$IID/trades"
