package legacylogin

import (
	"encoding/binary"
	"testing"
)

func TestRoleRecordWireTypes(t *testing.T) {
	record := RoleRecord{Params: []RoleParam{
		{Type: RoleParamU32, U32: 0x11223344},
		{Type: RoleParamU64A, U64: 0x0102030405060708},
		{Type: RoleParamU32B, U32: 9},
		{Type: RoleParamU64B, U64: 10},
		{Type: RoleParamString, Text: "Alex"},
		{Type: RoleParamUTF16, Text: "Herói"},
		{Type: RoleParamPairU32, Pair: [2]uint32{11, 12}},
	}}
	b, err := record.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(b[:2]); got != 7 {
		t.Fatalf("param_count=%d", got)
	}
	if b[2] != byte(RoleParamU32) || binary.LittleEndian.Uint32(b[3:7]) != 0x11223344 {
		t.Fatalf("bad first parameter: %x", b[:7])
	}
}

func TestRoleStringIncludesNULTerminator(t *testing.T) {
	b, err := (RoleParam{Type: RoleParamString, Text: "abc"}).MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if b[0] != 6 || binary.LittleEndian.Uint32(b[1:5]) != 4 || string(b[5:]) != "abc\x00" {
		t.Fatalf("string param=%x", b)
	}
}

func TestRoleUTF16LengthIsBytesAndTerminated(t *testing.T) {
	b, err := (RoleParam{Type: RoleParamUTF16, Text: "A"}).MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if b[0] != 7 || binary.LittleEndian.Uint32(b[1:5]) != 4 {
		t.Fatalf("utf16 header=%x", b[:5])
	}
	if binary.LittleEndian.Uint16(b[5:7]) != 'A' || binary.LittleEndian.Uint16(b[7:9]) != 0 {
		t.Fatalf("utf16 payload=%x", b[5:])
	}
}

func TestBuildServerLoginSuccessPacketWithRole(t *testing.T) {
	roles := []RoleRecord{{Params: []RoleParam{{Type: RoleParamU32, U32: 77}}}}
	packet, err := BuildServerLoginSuccessPacket([9]uint32{}, roles)
	if err != nil {
		t.Fatal(err)
	}
	if packet[0] != ServerLoginSuccessOpcode {
		t.Fatalf("opcode=%02x", packet[0])
	}
	if got := binary.LittleEndian.Uint32(packet[37:41]); got != 1 {
		t.Fatalf("role_count=%d", got)
	}
	if got := binary.LittleEndian.Uint16(packet[41:43]); got != 1 {
		t.Fatalf("role param_count=%d", got)
	}
	if packet[43] != 2 || binary.LittleEndian.Uint32(packet[44:48]) != 77 {
		t.Fatalf("role payload=%x", packet[43:])
	}
}

func TestRejectUnknownRoleType(t *testing.T) {
	_, err := (RoleParam{Type: 9}).MarshalBinary()
	if err == nil {
		t.Fatal("expected unsupported role type error")
	}
}
