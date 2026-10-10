# Client compatibility work

Target client: Survival Heroes 3.3.3 (`com.snailgamesusa.survival`).

This repository intentionally does **not** include the proprietary APK/OBB/assets. The operator supplies a legitimately obtained client package locally.

## Phase A — static inventory

1. Preserve untouched APK/XAPK/OBB and record SHA-256.
2. Run `python3 tools/apk-scan/scan.py <client.xapk>`.
3. Inventory AndroidManifest permissions, activities, native `.so` libraries and Unity metadata.
4. Identify hard-coded bootstrap/login/content endpoints.
5. Determine serialization/protocol candidates (JSON, protobuf, flatbuffers, custom binary).

## Phase B — network observation

Run only against a client/device you control. Record DNS lookups, connection destinations and TLS/plaintext characteristics. Do not intercept third-party credentials. We are interested in reconstructing the historical service interface, not accessing any surviving private service.

## Phase C — clean-room compatibility

Implement observed request/response semantics in `gateway` and packet semantics in `game-server`, documenting every field as confirmed, inferred, or unknown. Keep placeholder protocol code separate from confirmed compatibility code.

## First compatibility milestone

`3.3.3 client -> local bootstrap -> local login -> lobby shell`

Second milestone:

`two real clients -> local matchmaking -> same match -> movement visible on both clients`
