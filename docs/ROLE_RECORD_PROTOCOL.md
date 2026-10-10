# Survival Heroes 3.3.3 login role-record protocol

This note records the role-list grammar consumed by the archived Android 3.3.3 client's `GameReceiver::ServerLoginSucceed` path.

## Login-success prefix

The server message uses opcode `0x04`. The client requires a fixed 41-byte prefix. A little-endian `uint32` at offset `37` is the number of role records that immediately follow the prefix.

A zero role count is valid at the parser level. The compatibility server can therefore emit a controlled zero-role packet to test the original no-role / create-role client path without inventing character data.

## Role record

Each role begins with:

```text
uint16 parameter_count
```

Then `parameter_count` typed parameters follow. Every parameter begins with a one-byte type tag. The 3.3.3 receiver accepts exactly tags `2` through `8`:

| Tag | Wire value |
|---:|---|
| `2` | `uint32` |
| `3` | `uint64` |
| `4` | `uint32` |
| `5` | `uint64` |
| `6` | `uint32 byte_length` + NUL-terminated byte string; the length includes the final NUL |
| `7` | `uint32 byte_length` + NUL-terminated UTF-16LE string; the length is measured in bytes and includes the final `0x0000` |
| `8` | two consecutive `uint32` values |

Other tags enter the client's `(GameReceiver::ServerLoginSucceed)role parameter type error` path.

The wire encoder is implemented in `internal/legacylogin/role_record.go` and deliberately uses neutral parameter names until higher-level field meanings are proven.

## Observed client-side materialization

After parsing the generic typed parameter vector, the client copies selected indexes into an internal role-summary object. Confirmed observations include:

- parameter index `0`, when tag `2`, is stored as a `uint32` at object offset `0`;
- parameter index `1`, when tag `2`, is stored as a `uint32` at offset `4`;
- parameter index `2`, when tag `7`, is copied as a wide string at offset `8`;
- parameter index `3`, when tag `7`, is copied as a wide string at offset `0x68`;
- parameter index `4`, when tag `2`, is stored at offset `0xF8` when enough parameters are present;
- parameter index `5`, when tag `5`, is stored as an eight-byte value at offset `0x100`;
- parameter index `7`, when tag `6`, is copied as a byte string at offset `0xC8` when present.

The business meanings of these fields are intentionally not named yet. Recovering wire type and location is not sufficient evidence to label a value as role ID, level, name, or another account/game field.

## Current compatibility experiment

`MEMBER_SEND_LOGIN_SUCCESS=true` makes `cmd/member-server` answer a validated `CLIENT_LOGIN` with the recovered 41-byte login-success prefix and `role_count=0`. This is intended to determine whether the original client transitions into its character-creation UI.

The next clean-room target is the create/select-role request and response sequence.
