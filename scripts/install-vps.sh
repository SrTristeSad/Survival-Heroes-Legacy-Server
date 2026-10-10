#!/usr/bin/env sh
set -eu

REPO_URL="https://github.com/SrTristeSad/Survival-Heroes-Legacy-Server.git"
INSTALL_DIR="${INSTALL_DIR:-/www/wwwroot/Survival-Heroes-Legacy-Server}"
HOST_ARG="${1:-}"

fail() {
  echo "ERRO: $*" >&2
  exit 1
}

command -v git >/dev/null 2>&1 || fail "git não está instalado."
command -v docker >/dev/null 2>&1 || fail "Docker não está instalado. Instale o Docker pelo aaPanel primeiro."
docker compose version >/dev/null 2>&1 || fail "Docker Compose v2 não está disponível."

if [ -d "$INSTALL_DIR/.git" ]; then
  echo "Atualizando projeto existente em $INSTALL_DIR ..."
  cd "$INSTALL_DIR"
  git pull --ff-only
else
  echo "Clonando projeto para $INSTALL_DIR ..."
  mkdir -p "$(dirname "$INSTALL_DIR")"
  git clone "$REPO_URL" "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

if [ -n "$HOST_ARG" ]; then
  PUBLIC_HOST="$HOST_ARG"
else
  PUBLIC_HOST=""
  if command -v curl >/dev/null 2>&1; then
    PUBLIC_HOST="$(curl -4 -fsS --max-time 8 https://api.ipify.org 2>/dev/null || true)"
  fi
  if [ -z "$PUBLIC_HOST" ]; then
    fail "não consegui detectar o IP público. Rode novamente passando o IP ou domínio: sh scripts/install-vps.sh SEU_IP"
  fi
fi

echo "Host público detectado/configurado: $PUBLIC_HOST"

[ -f .env ] || cp .env.example .env
sh scripts/configure-public-host.sh "$PUBLIC_HOST" 7001
sh scripts/enable-legacy-flow.sh
sh scripts/setup.sh

echo
echo "================ INSTALAÇÃO CONCLUÍDA ================"
echo "Projeto:        $INSTALL_DIR"
echo "Bootstrap:      $PUBLIC_HOST:88/TCP"
echo "Login/member:   $PUBLIC_HOST:7001/TCP"
echo "Gateway dev:    somente localhost:8080"
echo "Game dev:       somente localhost:7000"
echo
echo "No aaPanel > Segurança, libere ENTRADA TCP nas portas:"
echo "  88"
echo "  7001"
echo
echo "Estado dos containers:"
docker compose ps
echo
echo "Para acompanhar o cliente depois:"
echo "  cd $INSTALL_DIR"
echo "  docker compose logs -f query member"
echo "========================================================"
