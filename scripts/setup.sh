#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required. Install Docker/Compose from aaPanel or your distro first." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 ('docker compose') is required." >&2
  exit 1
fi

[ -f .env ] || cp .env.example .env

if [ "$#" -ge 1 ] && [ -n "$1" ]; then
  sh scripts/configure-public-host.sh "$1" "${2:-7001}"
fi

docker compose up -d --build

echo
echo "Survival Heroes revival services started."
echo "  Bootstrap query: TCP/${QUERY_PORT:-88}"
echo "  Member/login:    TCP/${MEMBER_PORT:-7001}"
echo "  Dev gateway:     http://127.0.0.1:${GATEWAY_PORT:-8080}/healthz"
echo
if [ "$#" -ge 1 ] && [ -n "$1" ]; then
  echo "Client bootstrap host to patch into assets/config.ini: $1"
fi
echo "Use 'docker compose logs -f query member' while testing the original client."
