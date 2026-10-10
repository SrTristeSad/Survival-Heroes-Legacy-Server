# Survival Heroes Legacy Server

Clean-room community server reimplementation scaffold for preserving **Survival Heroes** multiplayer after shutdown.

> Status: **Phase 0 / protocol discovery pending**. The included services are functional scaffolding, not yet compatible with the original 3.3.3 client.

## Goals

- Target the final known Android client line, starting with **3.3.3**.
- Reimplement server behavior from observations of the client; do not distribute proprietary game assets.
- Prioritize real players; bots, if implemented later, must be explicit and optional.
- Make self-hosting simple on a small VPS/aaPanel using Docker Compose.
- Document the protocol so preservation does not depend on one person.

## What already works in Phase 0

- HTTP health endpoint.
- Local guest account creation and bearer-token session.
- Profile endpoint.
- In-memory matchmaking queue.
- Automatic two-player match allocation (configurable with `MATCH_SIZE`).
- Separate TCP game service with a tiny JSON development protocol (`welcome`, `ping/pong`, `move`).
- Docker Compose deployment.
- Static APK/XAPK/OBB network-string scanner.

## Quick start

```bash
cp .env.example .env
./scripts/setup.sh
./scripts/smoke-test.sh
```

Services:

- Gateway/API: `http://localhost:8080`
- Development game socket: `tcp://localhost:7000`

## API (development only)

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

Do **not** commit APK, XAPK or OBB files. Put them outside the repository or in ignored local directories. See `docs/REVERSE_ENGINEERING.md`.

## Roadmap

1. Inspect pristine Survival Heroes 3.3.3 APK/XAPK + OBB.
2. Identify bootstrap/login endpoints and wire format.
3. Make the original client complete bootstrap against the local gateway.
4. Recreate account/profile/lobby responses.
5. Discover matchmaking/session negotiation.
6. Recreate battle connection and movement synchronization.
7. Add combat, loot, shrinking zone and authoritative validation incrementally.
8. Add PostgreSQL/Redis only after protocol behavior is known well enough to justify persistence.

## Legal / preservation note

This project is intended as an independent interoperability/preservation reimplementation. It contains no original server source code and should not redistribute proprietary client assets, trademarks or copyrighted game data.
