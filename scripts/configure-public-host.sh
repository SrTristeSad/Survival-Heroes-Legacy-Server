#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <public-ip-or-domain> [member-port]"
  exit 2
fi

HOST="$1"
PORT="${2:-7001}"

case "$PORT" in
  ''|*[!0-9]*) echo "member port must be numeric" >&2; exit 2 ;;
esac
if [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
  echo "member port must be between 1 and 65535" >&2
  exit 2
fi

[ -f .env ] || cp .env.example .env

python3 - "$HOST" "$PORT" <<'PY'
from pathlib import Path
import sys
host, port = sys.argv[1], sys.argv[2]
p = Path('.env')
lines = p.read_text(encoding='utf-8').splitlines()
updates = {
    'LEGACY_SERVER_HOST': host,
    'LEGACY_SERVER_PORT': port,
    'MEMBER_PORT': port,
}
seen = set()
out = []
for line in lines:
    if '=' in line and not line.lstrip().startswith('#'):
        key = line.split('=', 1)[0].strip()
        if key in updates:
            out.append(f'{key}={updates[key]}')
            seen.add(key)
            continue
    out.append(line)
for key, value in updates.items():
    if key not in seen:
        out.append(f'{key}={value}')
p.write_text('\n'.join(out) + '\n', encoding='utf-8')
PY

echo "Configured revival host: $HOST"
echo "Server-list member endpoint: $HOST:$PORT"
echo "Bootstrap query endpoint remains TCP/88 unless QUERY_PORT is changed."
echo "Run: sh scripts/setup.sh"
