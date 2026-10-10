#!/usr/bin/env sh
set -eu
BASE="${BASE:-http://127.0.0.1:8080}"
echo "[1] health"
curl -fsS "$BASE/healthz"; echo

echo "[2] guest A"
A=$(curl -fsS -X POST "$BASE/v1/auth/guest" -H 'content-type: application/json' -d '{"name":"Alex"}')
echo "$A"
TA=$(printf '%s' "$A" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

echo "[3] guest B"
B=$(curl -fsS -X POST "$BASE/v1/auth/guest" -H 'content-type: application/json' -d '{"name":"Tester"}')
echo "$B"
TB=$(printf '%s' "$B" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

echo "[4] enqueue A"
curl -fsS -X POST "$BASE/v1/matchmaking/enqueue" -H "authorization: Bearer $TA" -H 'content-type: application/json' -d '{"mode":"legacy-duo"}'; echo

echo "[5] enqueue B -> match should form"
curl -fsS -X POST "$BASE/v1/matchmaking/enqueue" -H "authorization: Bearer $TB" -H 'content-type: application/json' -d '{"mode":"legacy-duo"}'; echo

echo "[6] status A"
curl -fsS "$BASE/v1/matchmaking/status" -H "authorization: Bearer $TA"; echo
