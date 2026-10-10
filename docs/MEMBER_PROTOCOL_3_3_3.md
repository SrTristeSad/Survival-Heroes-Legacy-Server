# Survival Heroes 3.3.3 member/login wire protocol

This document contains only protocol details confirmed from the archived Android 3.3.3 client. Unknown field meanings are intentionally left unnamed rather than guessed.

## Outer TCP framing

The old member/game TCP path uses `LenSplitProtocol`.

`LenSplitProtocol::m_nProtocolSize` is `4`, and the receive routine interprets the first four bytes as the exact payload length.

```text
uint32_le payload_length
byte      payload[payload_length]
```

The first byte of each decoded payload is the old `GameReceiver` / `GameSender` opcode.

This framing is implemented by `internal/legacylogin.FrameWirePayload` and `ParseWireFrame`.

## Confirmed server-to-client opcodes

The ARM64 `GameReceiver` jump table was decoded directly rather than inferred from string order.

| Opcode | Handler |
|---:|---|
| `0x01` | `SERVER_SET_VERIFY` / `GameReceiver::ServerSetVerify` |
| `0x02` | `SERVER_SET_ENCODE` / `GameReceiver::ServerSetEncode` |
| `0x04` | `SERVER_LOGIN_SUCCEED` / `GameReceiver::ServerLoginSucceed` |

Opcode `0x03` dispatches to another handler whose role has not yet been named with enough confidence.

## `SERVER_SET_VERIFY` (`0x01`)

The receiver enforces a minimum payload size of 13 bytes.

```text
offset  size  type       meaning
0       1     uint8      opcode = 0x01
1       4     uint32_le  field0 (meaning not yet confirmed)
5       4     uint32_le  verify_mode
9       ...              mode-dependent data
```

If `verify_mode == 1`, the receiver requires at least 25 bytes and consumes a 16-byte key beginning at offset 9.

For modes other than 1, four bytes at offsets 9..12 are retained as zero padding by the current clean-room builder because the original receiver still enforces the 13-byte minimum.

Implemented by `legacylogin.BuildServerSetVerify`.

## `SERVER_SET_ENCODE` (`0x02`)

The receiver requires a payload larger than `0x3d`, therefore at least 62 bytes, before emitting its `on_set_encode` event.

The exact internal field grammar and the point at which its transform becomes active are still being recovered. The server must not invent this packet merely to advance the UI.

## `SERVER_LOGIN_SUCCEED` (`0x04`)

The receiver requires at least `0x29` (41) bytes.

The fixed prefix contains the opcode followed by ten little-endian 32-bit slots:

```text
offset  size  type
0       1     uint8      opcode = 0x04
1       4     uint32_le  value0
5       4     uint32_le  value1
9       4     uint32_le  value2
13      4     uint32_le  value3
17      4     uint32_le  value4
21      4     uint32_le  value5
25      4     uint32_le  value6
29      4     uint32_le  value7
33      4     uint32_le  value8 / fixed slot (semantic use still under review)
37      4     uint32_le  role_count
```

Role records begin at offset 41 when `role_count > 0`. Their grammar is not implemented yet.

`BuildServerLoginSuccessHeader` deliberately accepts only `role_count == 0` until those role records are recovered.

## Client login request

The already-recovered raw `CLIENT_LOGIN` payload begins with opcode `0x02`, family `0x300`, and the padded account/password/login-string structure documented in `CLIENT_3_3_3_ANALYSIS.md`.

The client enum/string table places `CLIENT_GET_VERIFY` immediately before `CLIENT_LOGIN`, strongly suggesting client opcode `0x01`; however this document will mark that value confirmed only after the actual sender/connection-success path is located in native code.

## Safety / privacy in the compatibility server

`cmd/member-server` validates login framing without logging or retaining account passwords, login tokens, or device identifier contents. It records only lengths and non-secret protocol selectors required for reverse engineering.
