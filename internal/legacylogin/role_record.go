package legacylogin

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// RoleParamType is the one-byte tag consumed by GameReceiver::ServerLoginSucceed.
// The archived 3.3.3 client accepts exactly tags 2 through 8.
type RoleParamType byte

const (
	RoleParamU32     RoleParamType = 2
	RoleParamU64A    RoleParamType = 3
	RoleParamU32B    RoleParamType = 4
	RoleParamU64B    RoleParamType = 5
	RoleParamString  RoleParamType = 6
	RoleParamUTF16   RoleParamType = 7
	RoleParamPairU32 RoleParamType = 8
)

type RoleParam struct {
	Type RoleParamType
	U32  uint32
	U64  uint64
	Text string
	Pair [2]uint32
}

type RoleRecord struct {
	Params []RoleParam
}

func appendU32(dst []byte, v uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return append(dst, b[:]...)
}

func appendU64(dst []byte, v uint64) []byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	return append(dst, b[:]...)
}

func encodeUTF16LE(text string) []byte {
	units := utf16.Encode([]rune(text))
	units = append(units, 0)
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], u)
	}
	return out
}

func (p RoleParam) MarshalBinary() ([]byte, error) {
	out := []byte{byte(p.Type)}
	switch p.Type {
	case RoleParamU32, RoleParamU32B:
		out = appendU32(out, p.U32)
	case RoleParamU64A, RoleParamU64B:
		out = appendU64(out, p.U64)
	case RoleParamString:
		if len(p.Text) >= int(^uint32(0)) {
			return nil, fmt.Errorf("legacy login: role string too large")
		}
		raw := append([]byte(p.Text), 0)
		out = appendU32(out, uint32(len(raw)))
		out = append(out, raw...)
	case RoleParamUTF16:
		raw := encodeUTF16LE(p.Text)
		out = appendU32(out, uint32(len(raw)))
		out = append(out, raw...)
	case RoleParamPairU32:
		out = appendU32(out, p.Pair[0])
		out = appendU32(out, p.Pair[1])
	default:
		return nil, fmt.Errorf("legacy login: unsupported role parameter type %d", p.Type)
	}
	return out, nil
}

func (r RoleRecord) MarshalBinary() ([]byte, error) {
	if len(r.Params) > int(^uint16(0)) {
		return nil, fmt.Errorf("legacy login: too many role parameters: %d", len(r.Params))
	}
	out := make([]byte, 2)
	binary.LittleEndian.PutUint16(out, uint16(len(r.Params)))
	for i, p := range r.Params {
		b, err := p.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("legacy login: role parameter %d: %w", i, err)
		}
		out = append(out, b...)
	}
	return out, nil
}

// BuildServerLoginSuccessPacket constructs the full opcode-0x04 packet using
// the recovered 41-byte fixed prefix followed by zero or more typed role records.
func BuildServerLoginSuccessPacket(values [9]uint32, roles []RoleRecord) ([]byte, error) {
	if len(roles) > int(^uint32(0)) {
		return nil, fmt.Errorf("legacy login: too many roles: %d", len(roles))
	}
	out := make([]byte, ServerLoginSuccessSize)
	out[0] = ServerLoginSuccessOpcode
	for i, v := range values {
		off := 1 + i*4
		binary.LittleEndian.PutUint32(out[off:off+4], v)
	}
	binary.LittleEndian.PutUint32(out[37:41], uint32(len(roles)))
	for i, role := range roles {
		b, err := role.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("legacy login: role %d: %w", i, err)
		}
		out = append(out, b...)
	}
	return out, nil
}
