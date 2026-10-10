package legacylogin

import (
	"encoding/binary"
	"fmt"
)

const (
	// LenSplitProtocol::m_nProtocolSize is 4 in the archived 3.3.3 client.
	WireHeaderSize = 4

	// Client opcodes are zero-based. The archived enum/string table starts with
	// GET_VERIFY, RET_ENCODE, LOGIN, and the native senders independently confirm
	// a one-byte 0x00 GET_VERIFY packet and the already-recovered 0x02 LOGIN.
	ClientGetVerifyOpcode = byte(0x00)
	ClientRetEncodeOpcode = byte(0x01)
	ClientLoginOpcode     = byte(0x02)

	// Exact GameReceiver dispatch opcodes recovered from the client jump table.
	ServerSetVerifyOpcode    = byte(0x01)
	ServerSetEncodeOpcode    = byte(0x02)
	ServerLoginSuccessOpcode = byte(0x04)

	ServerSetVerifyBaseSize = 13
	ServerSetVerifyKeySize  = 16
	ServerSetEncodeMinSize  = 62
	ServerLoginSuccessSize  = 41
)

// FrameWirePayload applies the LenSplitProtocol outer framing used by the
// legacy TCP game/member connection: a little-endian uint32 payload length
// followed immediately by the payload.
func FrameWirePayload(payload []byte) ([]byte, error) {
	if uint64(len(payload)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("payload too large: %d", len(payload))
	}
	out := make([]byte, WireHeaderSize+len(payload))
	binary.LittleEndian.PutUint32(out[:WireHeaderSize], uint32(len(payload)))
	copy(out[WireHeaderSize:], payload)
	return out, nil
}

// ParseWireFrame returns one complete LenSplitProtocol payload from data.
// consumed is zero when more bytes are required.
func ParseWireFrame(data []byte) (payload []byte, consumed int, err error) {
	if len(data) < WireHeaderSize {
		return nil, 0, ErrNeedMore
	}
	n := uint64(binary.LittleEndian.Uint32(data[:WireHeaderSize]))
	total := uint64(WireHeaderSize) + n
	if total > uint64(len(data)) {
		return nil, 0, ErrNeedMore
	}
	return data[WireHeaderSize:int(total)], int(total), nil
}

// BuildClientGetVerify returns the exact raw one-byte payload emitted by the
// original client's pre-login GameSender method. It is useful in protocol
// tests and packet-capture comparisons; the compatibility server never sends
// this client-to-server packet itself.
func BuildClientGetVerify() []byte {
	return []byte{ClientGetVerifyOpcode}
}

// SetVerifyPacket is the exact fixed portion consumed by
// GameReceiver::ServerSetVerify. Field0 is intentionally unnamed until its
// higher-level meaning is confirmed from the client flow.
type SetVerifyPacket struct {
	Field0 uint32
	Mode   uint32
	Key    [ServerSetVerifyKeySize]byte
}

// BuildServerSetVerify constructs opcode 0x01. Mode 1 carries a 16-byte key;
// other modes retain the 13-byte minimum shape required by the receiver.
func BuildServerSetVerify(v SetVerifyPacket) ([]byte, error) {
	if v.Mode == 1 {
		out := make([]byte, 9+ServerSetVerifyKeySize)
		out[0] = ServerSetVerifyOpcode
		binary.LittleEndian.PutUint32(out[1:5], v.Field0)
		binary.LittleEndian.PutUint32(out[5:9], v.Mode)
		copy(out[9:], v.Key[:])
		return out, nil
	}

	out := make([]byte, ServerSetVerifyBaseSize)
	out[0] = ServerSetVerifyOpcode
	binary.LittleEndian.PutUint32(out[1:5], v.Field0)
	binary.LittleEndian.PutUint32(out[5:9], v.Mode)
	// Bytes 9..12 are retained as zero padding because the receiver enforces a
	// 13-byte minimum even when it does not consume the 16-byte key.
	return out, nil
}

// LoginSuccessHeader models only the 41-byte fixed prefix consumed before the
// role list. The meanings of Values[0..8] are deliberately not guessed.
// Values[9] is the role count at offset +37.
type LoginSuccessHeader struct {
	Values [10]uint32
}

func (h LoginSuccessHeader) RoleCount() uint32 { return h.Values[9] }

// BuildServerLoginSuccessHeader constructs the fixed 41-byte opcode-0x04
// prefix. Role records, when RoleCount > 0, must be appended separately only
// after their grammar has been recovered.
func BuildServerLoginSuccessHeader(h LoginSuccessHeader) ([]byte, error) {
	if h.RoleCount() != 0 {
		return nil, fmt.Errorf("role_count=%d requires role records that are not implemented yet", h.RoleCount())
	}
	out := make([]byte, ServerLoginSuccessSize)
	out[0] = ServerLoginSuccessOpcode
	for i, v := range h.Values {
		off := 1 + i*4
		binary.LittleEndian.PutUint32(out[off:off+4], v)
	}
	return out, nil
}
