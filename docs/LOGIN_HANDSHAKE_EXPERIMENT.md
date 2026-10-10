# Survival Heroes 3.3.3 — member/login handshake experiment

This document describes only behavior recovered from the archived 3.3.3 Android client plus explicitly marked test hypotheses.

## Confirmed wire sequence

The member/login connection uses `LenSplitProtocol`:

```text
uint32_le payload_length
byte      payload[payload_length]
```

Recovered client opcodes:

```text
0x00 CLIENT_GET_VERIFY
0x01 CLIENT_RET_ENCODE
0x02 CLIENT_LOGIN
```

Recovered server opcodes:

```text
0x01 SERVER_SET_VERIFY
0x02 SERVER_SET_ENCODE
0x04 SERVER_LOGIN_SUCCEED
```

## SERVER_SET_VERIFY

`GameReceiver::ServerSetVerify` requires at least 13 payload bytes.

```text
offset 0   uint8     opcode = 0x01
offset 1   uint32_le field0
offset 5   uint32_le mode
offset 9   byte[4]   padding when mode != 1
```

When `mode == 1`, the receiver requires at least 25 bytes and copies exactly 16 bytes from offset 9:

```text
offset 9   byte[16] key
```

The higher-level meanings of `field0` and `mode` are still being validated and are deliberately not named in code.

## SERVER_SET_ENCODE

The 3.3.3 ARM64 receiver checks only that the payload length is greater than 61 bytes. In the observed `GameReceiver::ServerSetEncode` routine, the packet body is not read before the `on_set_encode` callback is invoked.

The compatibility server therefore has a controlled experimental builder for the minimum accepted shape:

```text
offset 0   uint8     opcode = 0x02
offset 1   byte[61]  zero-filled research payload
```

This is **62 payload bytes**, plus the normal four-byte LenSplit length prefix.

## Controlled test switches

The experimental replies are disabled by default.

```env
MEMBER_SEND_SET_VERIFY=true
MEMBER_SET_VERIFY_FIELD0=0
MEMBER_SET_VERIFY_MODE=0
MEMBER_SET_VERIFY_KEY_HEX=
MEMBER_SEND_SET_ENCODE=true
```

For `MEMBER_SET_VERIFY_MODE=1`, `MEMBER_SET_VERIFY_KEY_HEX` must contain exactly 16 bytes / 32 hexadecimal characters.

## Intended tester-owned client experiment

With a 3.3.3 client repacked to point at a revival host and the server list pointed at TCP/7001, observe this sequence in `docker compose logs -f member`:

```text
CLIENT_GET_VERIFY
SERVER_SET_VERIFY
CLIENT_RET_ENCODE        # if the selected mode causes the client to emit it
SERVER_SET_ENCODE
CLIENT_LOGIN
```

The server intentionally logs only metadata for `CLIENT_LOGIN`; it does not print the account/password/login-string contents.

## Current boundary

`SERVER_LOGIN_SUCCEED` opcode and its fixed 41-byte prefix are recovered, including the role-count position, but the role-record grammar is not complete. The server must not invent role records merely to force the client forward.
