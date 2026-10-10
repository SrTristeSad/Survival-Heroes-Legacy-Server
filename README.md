# Survival Heroes Legacy Server

Clean-room community server reimplementation for preserving **Survival Heroes** multiplayer after shutdown.

> Status: **Phase 1 / original 3.3.3 bootstrap protocol recovered.** The TCP server-query service now implements the packet shape used by the archived client. Login/member-server and battle protocols are still being recovered.

## Goals

- Target the final known Android client line, starting with **3.3.3**.
- Reimplement server behavior from observations of the client; do not distribute proprietary game assets.
- Prioritize real players; bots, if implemented later, must be explicit and optional.
- Make self-hosting simple on a small VPS/aaPanel using Docker Compose.
- Document the protocol so preservation does not depend on one person.

## Current progress

### Recovered from the original 3.3.3 client

- Original server-query bootstrap endpoint and packet framing.
- Exact 28-byte little-endian query request used on TCP port **88**.
- Query response envelope, raw/zlib behavior and NUL-terminated server-line payload.
- `server_list.ini` generation behavior (`one returned string = one CRLF-terminated line`).
- Main OBB archive format: **PCK0 v15**, including a working clean-room index/extraction tool.
- Battle networking architecture: native UDP channels with **KCP** used by the reliable-sequenced channel.
- Runtime identified as native C++ with Cocos2d-x-related components, Lua and PhysX; it is not Unity.

See [`docs/CLIENT_3_3_3_ANALYSIS.md`](docs/CLIENT_3_3_3_ANALYSIS.md) for confirmed details.

### Server components

- `query-server`: client-compatible TCP/88 bootstrap query implementation.
- `gateway`: Phase-0 HTTP guest/profile/matchmaking laboratory.
- `game-server`: Phase-0 TCP/JSON synchronization laboratory; **not** the final battle protocol.
- Docker Compose deployment.
- Automated Go tests/build checks.
- APK/XAPK/OBB static-analysis workflows.
- PCK0 v15 parser/extractor at `tools/pck0_extract.py`.

## Quick start

```bash
cp .env.example .env
sh scripts/setup.sh
sh scripts/smoke-test.sh
```

Default services:

- Original-compatible server query: `tcp://localhost:88`
- Development Gateway/API: `http://localhost:8080`
- Development game socket: `tcp://localhost:7000`

The archived client still contains its historical raw query IP. For interoperability testing, a tester-owned APK must be repackaged so `[server_query].Addr` in `assets/config.ini` points at the revival host. Do not probe the historical IP.

## Development API

The HTTP API and TCP/JSON game socket are scaffolding used while the real protocols are recovered. They are intentionally kept separate from the legacy-compatible implementations.

### Create guest

```bash
curl -X POST http://localhost:8080/v1/auth/guest \
  -H 'content-type: application/json' \
  -d '{"name":"Alex"}'
```

### Join matchmaking

```bash
curl -X POST http://localhost:8080/v1/matchmaking/enqueue \
  -H 'authorization: Bearer YOUR_TOKEN' \
  -H 'content-type: application/json' \
  -d '{"mode":"legacy-duo"}'
```

## Client files

Do **not** commit APK, XAPK or OBB files to this repository. Analysis/repacking tools operate on a client supplied separately by its owner. See `docs/REVERSE_ENGINEERING.md`.

## Roadmap

1. ✅ Inspect pristine Survival Heroes 3.3.3 APK/XAPK + OBB.
2. ✅ Recover the TCP/88 server-query request/response protocol.
3. 🚧 Recover the exact `server_list.ini` line schema and return the revival server.
4. Patch a tester-owned 3.3.3 client to use the revival bootstrap host.
5. Recover member/login server framing and authentication flow.
6. Reach the original lobby.
7. Recover room/match negotiation.
8. Implement the native UDP/KCP battle transport and movement synchronization.
9. Add combat, loot, shrinking zone and authoritative validation incrementally.
10. Add persistence only after the original behavior is understood well enough to model it correctly.

## Legal / preservation note

This project is intended as an independent interoperability/preservation reimplementation. It contains no original server source code and should not redistribute proprietary client assets, trademarks or copyrighted game data.
