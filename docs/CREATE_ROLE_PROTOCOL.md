# Survival Heroes 3.3.3 create-role request

The archived client's `GameSender::CreateRole` path has been recovered far enough to validate the fixed request prefix without guessing game semantics.

## Fixed request

The request uses client opcode `0x05` and has a minimum fixed size of 73 bytes:

```text
offset 0     uint8   opcode = 0x05
offset 1     uint32  field0
offset 5     uint16  name_utf16le[33]
offset 71    uint16  extra_parameter_count
offset 73    ...     optional generic parameters
```

The name area is produced from the wide-string argument and is capped to 33 UTF-16 code units in the released ARM64 client. The second script-side argument is required to be the client's wide-string variant type (`7`).

The first argument is converted to a 32-bit value and stored at offset `1`. Its higher-level meaning is intentionally left unnamed until a caller or data-flow path proves it.

The sender writes `argument_count - 2` into the 16-bit extra-parameter count. Optional extra parameters are serialized after the 73-byte fixed prefix by a shared generic parameter helper whose complete wire grammar is still being mapped.

`internal/legacylogin/create_role.go` validates the fixed prefix and reports only structural metadata; `cmd/member-server` deliberately does not log the requested character name itself.

## Message-table context

The original client message table confirms this early sequence:

```text
client 0x00 CLIENT_GET_VERIFY
client 0x01 CLIENT_RET_ENCODE
client 0x02 CLIENT_LOGIN
client 0x03 CLIENT_WORLD_INFO
client 0x04 CLIENT_CHOOSE_ROLE
client 0x05 CLIENT_CREATE_ROLE
client 0x06 CLIENT_DELETE_ROLE

server 0x01 SERVER_SET_VERIFY
server 0x02 SERVER_SET_ENCODE
server 0x03 SERVER_ERROR_CODE
server 0x04 SERVER_LOGIN_SUCCEED
server 0x05 SERVER_WORLD_INFO
```

There is no create-role-specific success message in the recovered server message table. Failures can use the generic `SERVER_ERROR_CODE`; successful creation/selection therefore continues through the normal role/world-entry flow rather than a dedicated `SERVER_CREATE_ROLE_SUCCESS` packet.

The next clean-room target is `CLIENT_CHOOSE_ROLE` and the corresponding `SERVER_WORLD_INFO` payload.
