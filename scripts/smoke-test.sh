#!/usr/bin/env sh
set -eu
BASE="${BASE:-http://127.0.0.1:8080}"
echo "[1] saude do servico"
curl -fsS "$BASE/healthz"; echo

echo "[2] convidado A"
A=$(curl -fsS -X POST "$BASE/v1/auth/guest" -H 'content-type: application/json' -d '{"name":"Alex"}')
echo "$A"
TA=$(printf '%s' "$A" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

echo "[3] convidado B"
B=$(curl -fsS -X POST "$BASE/v1/auth/guest" -H 'content-type: application/json' -d '{"name":"Tester"}')
echo "$B"
TB=$(printf '%s' "$B" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

echo "[4] colocar A na fila"
curl -fsS -X POST "$BASE/v1/matchmaking/enqueue" -H "authorization: Bearer $TA" -H 'content-type: application/json' -d '{"mode":"legacy-duo"}'; echo

echo "[5] colocar B na fila -> a partida deve ser formada"
curl -fsS -X POST "$BASE/v1/matchmaking/enqueue" -H "authorization: Bearer $TB" -H 'content-type: application/json' -d '{"mode":"legacy-duo"}'; echo

echo "[6] status de A"
curl -fsS "$BASE/v1/matchmaking/status" -H "authorization: Bearer $TA"; echo
