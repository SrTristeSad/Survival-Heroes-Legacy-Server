#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

SERVER_FILE="$ROOT_DIR/client-config/server.txt"
HOST="${1:-}"
PORT="${2:-88}"

# Prioridade: argumento explicito -> client-config/server.txt -> .env.
if [ -z "$HOST" ] && [ -f "$SERVER_FILE" ]; then
  HOST=$(sed -e 's/#.*$//' -e 's/[[:space:]]//g' "$SERVER_FILE" | sed -n '/./{p;q;}')
fi
if [ -z "$HOST" ] && [ -f .env ]; then
  HOST=$(sed -n 's/^LEGACY_SERVER_HOST=//p' .env | tail -n1)
fi
if [ -z "$HOST" ]; then
  echo "Uso: sh scripts/build-client-vps.sh [IP-ou-dominio] [porta-bootstrap]" >&2
  echo "Ou preencha client-config/server.txt." >&2
  exit 2
fi
case "$PORT" in
  ''|*[!0-9]*) echo "A porta precisa ser numerica." >&2; exit 2 ;;
esac
if [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
  echo "A porta precisa estar entre 1 e 65535." >&2
  exit 2
fi

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO=sudo
  else
    echo "Este script precisa de root ou sudo para instalar as ferramentas Android." >&2
    exit 1
  fi
fi

faltando=""
for cmd in curl unzip zip python3 keytool zipalign apksigner; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    faltando="$faltando $cmd"
  fi
done

if [ -n "$faltando" ]; then
  echo "Instalando ferramentas necessarias:$faltando"
  $SUDO apt-get update -qq
  $SUDO DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
    curl unzip zip python3 openjdk-17-jdk-headless zipalign apksigner
fi

BUILD_DIR="$ROOT_DIR/build/cliente"
CACHE_DIR="$ROOT_DIR/build/cache"
WORK_DIR="$ROOT_DIR/build/cliente-work"
KEYSTORE="$BUILD_DIR/survival-heroes-teste.jks"
XAPK_ORIGINAL="$CACHE_DIR/Survival-Heroes-3.3.3-original.xapk"
SAFE_HOST=$(printf '%s' "$HOST" | tr '/: ' '___')
OUT_XAPK="$BUILD_DIR/Survival-Heroes-3.3.3-Legacy-${SAFE_HOST}.xapk"
OUT_APK="$BUILD_DIR/Survival-Heroes-3.3.3-Legacy-${SAFE_HOST}.apk"

mkdir -p "$BUILD_DIR" "$CACHE_DIR"
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR/xapk"

# Guarda junto do build o endereco usado, para ficar facil trocar/repetir depois.
printf '%s\n' "$HOST" > "$BUILD_DIR/server.txt"

if [ ! -s "$XAPK_ORIGINAL" ]; then
  echo "Localizando o XAPK que voce enviou para a Release 'apk'..."
  RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/SrTristeSad/Survival-Heroes-Legacy-Server/releases/tags/apk")
  ASSET_URL=$(printf '%s' "$RELEASE_JSON" | python3 -c '
import json,sys
r=json.load(sys.stdin)
for a in r.get("assets",[]):
    n=a.get("name","").lower()
    if n.endswith(".xapk"):
        print(a["browser_download_url"])
        break
')
  if [ -z "$ASSET_URL" ]; then
    echo "Nao encontrei um .xapk na Release de tag 'apk'." >&2
    exit 1
  fi
  echo "Baixando o XAPK original para o cache privado da VPS..."
  curl -fL --retry 3 --progress-bar "$ASSET_URL" -o "$XAPK_ORIGINAL"
fi

if ! unzip -tq "$XAPK_ORIGINAL" >/dev/null; then
  echo "O XAPK baixado esta corrompido." >&2
  exit 1
fi

echo "Extraindo apenas a estrutura necessaria..."
unzip -q "$XAPK_ORIGINAL" -d "$WORK_DIR/xapk"

APK_ORIGINAL=$(find "$WORK_DIR/xapk" -type f -name '*.apk' | head -n1 || true)
if [ -z "$APK_ORIGINAL" ]; then
  echo "Nenhum APK encontrado dentro do XAPK." >&2
  exit 1
fi
APK_REL=${APK_ORIGINAL#"$WORK_DIR/xapk/"}

UNSIGNED="$WORK_DIR/cliente-patched-unsigned.apk"
ALIGNED="$WORK_DIR/cliente-patched-aligned.apk"
SIGNED="$WORK_DIR/cliente-patched-signed.apk"

python3 tools/patch_client.py "$APK_ORIGINAL" "$UNSIGNED" --host "$HOST" --port "$PORT"
zipalign -f -p 4 "$UNSIGNED" "$ALIGNED"

if [ ! -f "$KEYSTORE" ]; then
  echo "Criando chave local de teste (fica somente nesta VPS)..."
  keytool -genkeypair -noprompt \
    -keystore "$KEYSTORE" \
    -storepass android -keypass android \
    -alias survival-heroes-legacy \
    -keyalg RSA -keysize 2048 -validity 3650 \
    -dname "CN=Survival Heroes Legacy Test, OU=Preservacao, O=Community, C=BR" >/dev/null 2>&1
  chmod 600 "$KEYSTORE" || true
fi

apksigner sign \
  --ks "$KEYSTORE" \
  --ks-key-alias survival-heroes-legacy \
  --ks-pass pass:android \
  --key-pass pass:android \
  --v1-signing-enabled true \
  --v2-signing-enabled true \
  --out "$SIGNED" "$ALIGNED"

apksigner verify --verbose "$SIGNED" >/dev/null

if ! unzip -p "$SIGNED" assets/config.ini | grep -F "Addr=$HOST" >/dev/null; then
  echo "Falha de validacao: o host nao apareceu no config.ini final." >&2
  exit 1
fi
if ! unzip -p "$SIGNED" assets/config.ini | grep -F "Port=$PORT" >/dev/null; then
  echo "Falha de validacao: a porta nao apareceu no config.ini final." >&2
  exit 1
fi

cp "$SIGNED" "$OUT_APK"

# Reaproveita o XAPK fornecido pelo proprio usuario e troca somente o APK base.
# O OBB e os demais arquivos permanecem byte a byte como estavam.
cp "$XAPK_ORIGINAL" "$OUT_XAPK"
zip -q -d "$OUT_XAPK" "$APK_REL" >/dev/null || true
cp "$SIGNED" "$WORK_DIR/xapk/$APK_REL"
(
  cd "$WORK_DIR/xapk"
  zip -q -0 "$OUT_XAPK" "$APK_REL"
)

cat > "$BUILD_DIR/LEIA-ME.txt" <<EOF
Survival Heroes 3.3.3 - cliente de teste Legacy

Bootstrap configurado: $HOST:$PORT
Login/member anunciado pelo servidor: $HOST:7001
Arquivo de endereco usado no build: server.txt

O APK atual foi modificado em assets/config.ini e reassinado com uma chave de teste
criada nesta VPS. O projeto tambem esta analisando o carregamento nativo para fazer
o server.txt ser lido diretamente pelo aplicativo em uma versao posterior, sem
precisar reassinar ao trocar o servidor.

Por causa da nova assinatura, ele nao pode ser instalado por cima de uma versao
assinada pela Snail Games. Desinstale uma instalacao antiga antes deste teste.

Arquivo recomendado para instalar:
$(basename "$OUT_XAPK")

APK separado (para depuracao/ADB):
$(basename "$OUT_APK")
EOF

printf '\n================ CLIENTE GERADO ================\n'
printf 'server.txt: %s\n' "$BUILD_DIR/server.txt"
printf 'Bootstrap:  %s:%s\n' "$HOST" "$PORT"
printf 'XAPK:       %s\n' "$OUT_XAPK"
printf 'APK:        %s\n' "$OUT_APK"
printf 'SHA-256 XAPK: '
sha256sum "$OUT_XAPK" | awk '{print $1}'
printf 'SHA-256 APK:  '
sha256sum "$OUT_APK" | awk '{print $1}'
printf '\nNo aaPanel, abra Arquivos e entre em:\n%s\n' "$BUILD_DIR"
printf 'Baixe primeiro o arquivo .xapk.\n'
printf '=================================================\n'
