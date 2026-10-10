#!/usr/bin/env sh
set -eu

[ -f .env ] || cp .env.example .env

python3 - <<'PY'
from pathlib import Path

p = Path('.env')
lines = p.read_text(encoding='utf-8').splitlines()
updates = {
    'MEMBER_SEND_SET_VERIFY': 'true',
    'MEMBER_SET_VERIFY_FIELD0': '0',
    'MEMBER_SET_VERIFY_MODE': '0',
    'MEMBER_SEND_SET_ENCODE': 'true',
    'MEMBER_SEND_LOGIN_SUCCESS': 'true',
    'MEMBER_SEND_WORLD_INFO': 'true',
    'MEMBER_WORLD_INFO_FIELD0': '0',
    'MEMBER_WORLD_INFO_TEXT': '',
    'MEMBER_SEND_SCENE_TABLES': 'true',
    'MEMBER_SEND_ENTRY_SCENE': 'true',
    'MEMBER_ENTRY_SCENE_FIELD0': '0',
    'MEMBER_ENTRY_SCENE_A': '119',
    'MEMBER_ENTRY_SCENE_B': '0',
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

cat <<'EOF'
Recovered 3.3.3 compatibility experiment enabled.

Flow enabled:
  GET_VERIFY -> SET_VERIFY
  RET_ENCODE -> SET_ENCODE
  LOGIN -> LOGIN_SUCCESS (zero roles)
  WORLD_INFO -> WORLD_INFO
  CHOOSE_ROLE -> empty PROPERTY/RECORD tables -> ENTRY_SCENE 119-0

This is a protocol-research profile for a tester-owned patched client. It does
not claim that the original lobby/map has been reached until verified live.

Next:
  sh scripts/setup.sh
  docker compose logs -f query member
EOF
