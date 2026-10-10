package legacylogin

import (
	"encoding/binary"
	"fmt"
)

const (
	ClientCreateRoleOpcode = byte(0x05)
	CreateRoleNameUnits    = 33
	CreateRoleNameBytes    = CreateRoleNameUnits * 2
	CreateRoleFixedSize    = 1 + 4 + CreateRoleNameBytes + 2 // 73 bytes
)

// CreateRoleMetadata contains only non-secret structural information from the
// recovered CLIENT_CREATE_ROLE request. The character name itself is not kept.
type CreateRoleMetadata struct {
	Field0          uint32
	NameUTF16Units  int
	ExtraParamCount uint16
	PayloadBytes    int
}

// InspectCreateRoleRequest validates the fixed 73-byte prefix recovered from
// GameSender::CreateRole. Additional parameters, if ExtraParamCount > 0, are
// intentionally left opaque until their generic Lua-parameter grammar is
// independently confirmed.
func InspectCreateRoleRequest(data []byte) (CreateRoleMetadata, error) {
	var m CreateRoleMetadata
	if len(data) < CreateRoleFixedSize {
		return m, ErrNeedMore
	}
	if data[0] != ClientCreateRoleOpcode {
		return m, fmt.Errorf("unexpected create-role opcode 0x%02x", data[0])
	}

	m.Field0 = binary.LittleEndian.Uint32(data[1:5])
	m.PayloadBytes = len(data)

	name := data[5 : 5+CreateRoleNameBytes]
	for i := 0; i < CreateRoleNameUnits; i++ {
		u := binary.LittleEndian.Uint16(name[i*2 : i*2+2])
		if u == 0 {
			break
		}
		m.NameUTF16Units++
	}
	m.ExtraParamCount = binary.LittleEndian.Uint16(data[71:73])
	return m, nil
}
