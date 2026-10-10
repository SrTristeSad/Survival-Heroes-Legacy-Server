package legacylogin

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

const (
	ClientWorldInfoOpcode  = byte(0x03)
	ClientChooseRoleOpcode = byte(0x04)
	ServerWorldInfoOpcode  = byte(0x05)

	// ChooseRole and ChooseRoleAndScene share the same 83-byte prefix. The
	// optional scene string starts at offset 83 and is always NUL-terminated;
	// an empty scene therefore produces an 84-byte packet.
	ChooseRoleUTF16Units  = 41
	ChooseRoleUTF16Bytes  = ChooseRoleUTF16Units * 2
	ChooseRoleSceneOffset = 1 + ChooseRoleUTF16Bytes
	ChooseRoleMinSize     = ChooseRoleSceneOffset + 1

	ClientWorldInfoMinSize = 9 // opcode + uint32 + one UTF-32LE NUL
	ServerWorldInfoMinSize = 5 // opcode + uint16 + one UTF-16LE NUL
)

type WorldInfoRequestMetadata struct {
	Field0         uint32
	TextUTF32Units int
	PayloadBytes   int
}

type ChooseRoleMetadata struct {
	RoleUTF16Units int
	SceneInfoBytes int
	PayloadBytes   int
}

// InspectClientWorldInfo validates GameSender's opcode-0x03 request. Android
// wchar_t is 32-bit in this client, so the trailing text is UTF-32LE / native
// wchar_t data terminated by a four-byte zero code point. The text itself is
// deliberately not retained.
func InspectClientWorldInfo(data []byte) (WorldInfoRequestMetadata, error) {
	var m WorldInfoRequestMetadata
	if len(data) < ClientWorldInfoMinSize {
		return m, ErrNeedMore
	}
	if data[0] != ClientWorldInfoOpcode {
		return m, fmt.Errorf("unexpected world-info opcode 0x%02x", data[0])
	}
	if (len(data)-5)%4 != 0 {
		return m, fmt.Errorf("legacy login: CLIENT_WORLD_INFO wchar payload is not 4-byte aligned")
	}
	m.Field0 = binary.LittleEndian.Uint32(data[1:5])
	m.PayloadBytes = len(data)
	terminated := false
	for off := 5; off+4 <= len(data); off += 4 {
		v := binary.LittleEndian.Uint32(data[off : off+4])
		if v == 0 {
			terminated = true
			if off+4 != len(data) {
				return m, fmt.Errorf("legacy login: CLIENT_WORLD_INFO has bytes after wchar terminator")
			}
			break
		}
		if v > utf8.MaxRune || (v >= 0xD800 && v <= 0xDFFF) {
			return m, fmt.Errorf("legacy login: CLIENT_WORLD_INFO invalid wchar code point 0x%x", v)
		}
		m.TextUTF32Units++
	}
	if !terminated {
		return m, fmt.Errorf("legacy login: CLIENT_WORLD_INFO missing wchar terminator")
	}
	return m, nil
}

// InspectChooseRole validates both the fixed ChooseRole form and the
// ChooseRoleAndScene variant. The role selector occupies a fixed UTF-16LE
// area; scene_info is a C string starting at byte offset 83. Neither value is
// retained by the compatibility server.
func InspectChooseRole(data []byte) (ChooseRoleMetadata, error) {
	var m ChooseRoleMetadata
	if len(data) < ChooseRoleMinSize {
		return m, ErrNeedMore
	}
	if data[0] != ClientChooseRoleOpcode {
		return m, fmt.Errorf("unexpected choose-role opcode 0x%02x", data[0])
	}
	m.PayloadBytes = len(data)

	role := data[1:ChooseRoleSceneOffset]
	for i := 0; i < ChooseRoleUTF16Units; i++ {
		if binary.LittleEndian.Uint16(role[i*2:i*2+2]) == 0 {
			break
		}
		m.RoleUTF16Units++
	}

	scene := data[ChooseRoleSceneOffset:]
	end := -1
	for i, b := range scene {
		if b == 0 {
			end = i
			break
		}
	}
	if end < 0 {
		return m, fmt.Errorf("legacy login: CLIENT_CHOOSE_ROLE scene_info is not NUL-terminated")
	}
	if end+1 != len(scene) {
		return m, fmt.Errorf("legacy login: CLIENT_CHOOSE_ROLE has bytes after scene_info terminator")
	}
	m.SceneInfoBytes = end
	return m, nil
}

// BuildServerWorldInfo constructs the exact simple shape consumed by
// GameReceiver::ServerWorldInfo: opcode 0x05, uint16 field, then a NUL-
// terminated UTF-16LE string. The higher-level meanings of field0/text remain
// intentionally unnamed until proven by the surrounding game flow.
func BuildServerWorldInfo(field0 uint16, text string) []byte {
	raw := encodeUTF16LE(text)
	out := make([]byte, 3, 3+len(raw))
	out[0] = ServerWorldInfoOpcode
	binary.LittleEndian.PutUint16(out[1:3], field0)
	out = append(out, raw...)
	return out
}
