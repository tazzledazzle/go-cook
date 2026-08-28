#!/usr/bin/env bash
set -euo pipefail

lsof -ti:8080 | xargs kill -9 2>/dev/null || true
rm -rf generated .nerv-data

go run ./nerv/cmd/nervctl serve &
SERVER_PID=$!

# Wait for the server to actually be ready, rather than a fixed sleep.
for i in $(seq 1 20); do
  if curl -s -o /dev/null localhost:8080/healthz; then
    break
  fi
  sleep 0.25
done

echo "--- creating api-svc ---"
API_RESP=$(curl -s -X POST localhost:8080/projects -H 'Content-Type: application/json' \
  -d '{"name":"api-svc","language":"go"}')
echo "$API_RESP"
API_ID=$(echo "$API_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['project']['id'])")

echo "--- creating auth-lib ---"
AUTH_RESP=$(curl -s -X POST localhost:8080/projects -H 'Content-Type: application/json' \
  -d '{"name":"auth-lib","language":"go"}')
echo "$AUTH_RESP"
AUTH_ID=$(echo "$AUTH_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['project']['id'])")

echo "api-svc id:  $API_ID"
echo "auth-lib id: $AUTH_ID"

echo "--- linking api-svc depends-on auth-lib ---"
curl -i -s -X POST "localhost:8080/projects/$API_ID/depends-on" \
  -H 'Content-Type: application/json' \
  -d "{\"depends_on_id\":\"$AUTH_ID\"}"
echo ""

echo "--- dependents of auth-lib ---"
curl -s "localhost:8080/projects/$AUTH_ID/dependents" | python3 -m json.tool

echo "--- attempting cycle (auth-lib depends-on api-svc) ---"
curl -i -s -X POST "localhost:8080/projects/$AUTH_ID/depends-on" \
  -H 'Content-Type: application/json' \
  -d "{\"depends_on_id\":\"$API_ID\"}"
echo ""

kill "$SERVER_PID"