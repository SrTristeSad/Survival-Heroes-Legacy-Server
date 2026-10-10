#!/usr/bin/env sh
set -eu
[ -f .env ] || cp .env.example .env
docker compose up -d --build
printf '\nGateway: http://127.0.0.1:%s/healthz\n' "${GATEWAY_PORT:-8080}"
