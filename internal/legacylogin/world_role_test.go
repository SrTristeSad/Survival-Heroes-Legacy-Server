package legacylogin

import (
	"encoding/binary"
	"testing"
)

func TestInspectClientWorldInfo(t *testing.T) {
	p := make([]byte, 5+3*4+4)
	p[0] = ClientWorldInfoOpcode
	binary.LittleEndian.PutUint32(p[1:5], 99)
	for i, r := range []uint32{'B', 'R', '1'} {
		binary.LittleEndian.PutUint32(p[5+i*4:9+i*4], r)
	}
	m, err := InspectClientWorldInfo(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.Field0 != 99 || m.TextUTF32Units != 3 || m.PayloadBytes != len(p) {
		t.Fatalf("metadata=%#v", m)
	}
}

func TestInspectChooseRoleEmptyScene(t *testing.T) {
	p := make([]byte, ChooseRoleMinSize)
	p[0] = ClientChooseRoleOpcode
	for i, u := range []uint16{'A', 'l', 'e', 'x', 0} {
		binary.LittleEndian.PutUint16(p[1+i*2:1+i*2+2], u)
	}
	m, err := InspectChooseRole(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.RoleUTF16Units != 4 || m.SceneInfoBytes != 0 || m.PayloadBytes != ChooseRoleMinSize {
		t.Fatalf("metadata=%#v", m)
	}
}

func TestInspectChooseRoleWithScene(t *testing.T) {
	p := make([]byte, ChooseRoleSceneOffset)
	p[0] = ClientChooseRoleOpcode
	p = append(p, []byte("scene-a\x00")...)
	m, err := InspectChooseRole(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.SceneInfoBytes != len("scene-a") {
		t.Fatalf("scene bytes=%d", m.SceneInfoBytes)
	}
}

func TestBuildServerWorldInfo(t *testing.T) {
	p := BuildServerWorldInfo(7, "BR")
	if len(p) != 3+6 { // B,R,NUL => three UTF-16 units
		t.Fatalf("len=%d", len(p))
	}
	if p[0] != ServerWorldInfoOpcode || binary.LittleEndian.Uint16(p[1:3]) != 7 {
		t.Fatalf("header=%x", p[:3])
	}
	if binary.LittleEndian.Uint16(p[3:5]) != 'B' || binary.LittleEndian.Uint16(p[5:7]) != 'R' || binary.LittleEndian.Uint16(p[7:9]) != 0 {
		t.Fatalf("utf16=%x", p[3:])
	}
}

func TestBuildServerWorldInfoEmptyIsMinimumSize(t *testing.T) {
	p := BuildServerWorldInfo(0, "")
	if len(p) != ServerWorldInfoMinSize {
		t.Fatalf("len=%d want=%d", len(p), ServerWorldInfoMinSize)
	}
}
