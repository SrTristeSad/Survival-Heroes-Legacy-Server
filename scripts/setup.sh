#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker e obrigatorio. Instale Docker/Compose pelo aaPanel ou pela sua distribuicao antes de continuar." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 ('docker compose') e obrigatorio." >&2
  exit 1
fi

[ -f .env ] || cp .env.example .env

if [ "$#" -ge 1 ] && [ -n "$1" ]; then
  sh scripts/configure-public-host.sh "$1" "${2:-7001}"
fi

docker compose up -d --build

echo
echo "Servicos do Survival Heroes Legacy iniciados."
echo "  Consulta bootstrap: TCP/${QUERY_PORT:-88}"
echo "  Login/membro:       TCP/${MEMBER_PORT:-7001}"
echo "  Gateway de dev:     http://127.0.0.1:${GATEWAY_PORT:-8080}/healthz"
echo
if [ "$#" -ge 1 ] && [ -n "$1" ]; then
  echo "Host que deve ser configurado em assets/config.ini do cliente: $1"
fi
echo "Durante o teste do cliente original, use: docker compose logs -f query member"
