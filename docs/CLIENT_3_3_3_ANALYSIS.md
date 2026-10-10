# Survival Heroes 3.3.3 client analysis

This document records confirmed clean-room observations from the archived Android 3.3.3 client. It deliberately separates facts observed in the client from hypotheses that still need validation.

## Target client

- Package: `com.snailgamesusa.survival`
- Version: `3.3.3`
- Version code: `103`
- Archived XAPK SHA-256: `b536ee1bc9a37ba8198aa5e6caeb6f008dce816d52d98c616e0477182462d98e`
- Main OBB: `main.103.com.snailgamesusa.survival.obb`

The project does not redistribute this client or its assets.

## Runtime / engine

The client is not Unity. Static analysis shows a native C++ game runtime with Cocos2d-x-related components, Lua support and PhysX. Networking is implemented primarily in `libapp.so`.

Relevant native networking symbols include:

- `CNetWork::ConnectServer`
- `CNetWork::DisconnectServer`
- `CNetWork::SendData`
- `CNetWork::ConnectUdpServer`
- `CNetWork::SendUdpData`
- `CNetWork::StartUDPSession`
- `LenSplitProtocol::MakeSendData`
- `LenSplitProtocol::ProcessRecvData`

The UDP stack exposes separate channel types including reliable sequenced, unreliable and unreliable sequenced channels. The reliable sequenced channel uses KCP.

**Consequence:** the original Phase-0 TCP/JSON game socket is only a development harness. It is not the final Survival Heroes battle transport.

## Original bootstrap configuration

`assets/config.ini` contains:

```ini
[server_query]
Addr=164.52.38.62
Port=88
Issuer=10000
GameID=125
```

The old raw IP must not be probed. A compatible test client should have this address redirected to a server controlled by the tester.

Other useful client configuration strings include:

- `server_id`
- `server_port`
- `server_district`
- `server_issuer`
- `server_recommend`
- `server_page`
- `server_list.ini`

## Server-query request on TCP/88

The 3.3.3 client connects via TCP to the `[server_query]` address and sends a fixed 28-byte request.

`CMsgBuf::SetInt` writes a native 32-bit value directly. On the released little-endian ARM Android client the request is therefore seven consecutive little-endian `uint32` values:

| Index | Value | Meaning |
|---:|---:|---|
| 0 | 24 | request framing/family constant |
| 1 | 1001 | query command |
| 2 | 1 | version/family constant |
| 3 | 1 | version/family constant |
| 4 | 22222 | request magic/family constant |
| 5 | 10000 | `Issuer` from `config.ini` |
| 6 | 125 | `GameID` from `config.ini` |

For the archived client defaults, the exact request bytes are:

```text
18 00 00 00
E9 03 00 00
01 00 00 00
01 00 00 00
CE 56 00 00
10 27 00 00
7D 00 00 00
```

This request format is implemented in `internal/legacyquery` and served by `cmd/query-server`.

## Server-query response envelope

`CPackageHeadSocket::GetRecvInfo` reads this envelope:

```text
uint32 remaining_length
uint32 unpacked_size
byte   payload[]
```

All integers are little-endian for this client.

- Total packet length is `remaining_length + 4`.
- If `unpacked_size == 0`, `payload` is used as-is.
- If `unpacked_size != 0`, the client passes the payload through zlib `uncompress()` using that value as the expected output capacity.

The server implementation currently emits the simpler uncompressed form.

## Server-query response payload

The client constructs `CMsgBuf` over the decoded payload and consumes:

```text
uint32 value0
uint32 value1
uint32 value2
uint32 value3
uint32 server_line_count
char   server_line_0[] + NUL
char   server_line_1[] + NUL
...
```

The first four integers are consumed but their results are not used by the observed receive routine. The current server keeps the request-family constants for them rather than depending on this behavior.

Strings are NUL-terminated. `CMsgBuf::SetString` writes `strlen + 1`; `StringVal` reads a NUL-terminated value.

## How returned strings become `server_list.ini`

This is confirmed:

1. The query response returns a vector of strings.
2. `CParseServerListFile::WriteServerListFile` appends every string verbatim.
3. After every string it appends the two-byte delimiter `\r\n`.
4. The result is written as `server_list.ini`.
5. The generic reader uses `strtok(..., "\r\n")`, therefore each response string is one logical line of the generated file.

## Recovered `server_list.ini` record grammar

`FormLoginServers::LoadServerList` splits every line using `/` and consumes exactly three text fields followed by five decimal integers:

```text
server_id/district_key/address/port/type/open/load/priority
```

Recovered semantics:

| Field | Client behavior |
|---|---|
| `server_id` | copied to the selected-login server ID |
| `district_key` | copied as district data and also passed through the client's localization path |
| `address` | copied directly to the selected-login server address |
| `port` | copied directly to the selected-login server port |
| `type` | `0` normal, `1` recommended, `2` new |
| `open` | `0` stopped, `1` open |
| `load` | when open: `0` good, `1` normal, `2` hot |
| `priority` | lower value wins when choosing among recommended servers |

The recommendation routine explicitly filters records whose `type == 1` and selects the smallest `priority` value. The UI maps `type == 1` to `server_recommand` (the original client spelling) and `type == 2` to `server_new`. Server-state rendering maps `open == 0` to `server_stop`; when `open == 1`, load values 0/1/2 map to `server_good`, `server_normal`, and `server_hot`.

`server_issuer` is maintained separately by the login/server-selection flow; it is not one of the eight slash-delimited fields.

The grammar is implemented and validated by `internal/legacyquery.ServerRecord`. `cmd/query-server` can generate a record from the `LEGACY_SERVER_*` environment variables. The raw `QUERY_SERVER_RECORDS` variable remains only as a protocol-research override.

## OBB / PCK0 archive

The main OBB is a custom `PCK0` archive, version 15.

Observed header:

```text
char[4] magic       = "PCK0"
uint32  version     = 15
uint16  flags       = 0
uint32  file_count  = 57879
uint32  data_offset = 4973871
uint8   reserved    = 0
```

A v15 index entry begins with a little-endian `uint16` record length and contains:

```text
uint64 data_offset
uint32 unpacked_size
uint32 packed_size       # 0 means stored/raw
uint32 hash/check value
byte[5] metadata
char[] NUL-terminated path
```

The archive contains 57,879 indexed resources; 57,690 entries use compression. Deflate/zlib handling has been validated against a known 292-byte resource.

The clean-room parser/extractor lives at `tools/pck0_extract.py`.

## Update endpoints found in the archived client

The client contains historical patch URLs such as:

```text
http://up-yxxg-us.snail.com/android36/patch
http://config-yxxg-us.snail.com/android36/version
http://up-yxxg-us.snail.com/android36/min
http://up-yxxg-us.snail.com/android36/res
http://up-yxxg-us.snail.com/android36/lng_patch
http://up-yxxg-us.snail.com/android36/bank
```

These are evidence about the old client architecture only. The revival server should not depend on the historical infrastructure.

## Next protocol milestones

1. ✅ Recover the exact `server_list.ini` line grammar.
2. ✅ Generate a revival-server entry from structured server configuration.
3. Patch a tester-owned 3.3.3 APK so `[server_query].Addr` points to the revival host.
4. Recover member/login server framing and message identifiers.
5. Reach the original lobby using the unmodified game logic and assets.
6. Recover room/match negotiation.
7. Replace the development game socket with the native UDP/KCP-compatible battle transport.

## Confidence labels

Unless explicitly marked as a hypothesis, values in this document were observed in the archived 3.3.3 client through configuration, native symbols, ARM64 disassembly or deterministic archive parsing. No original server source code was used.
