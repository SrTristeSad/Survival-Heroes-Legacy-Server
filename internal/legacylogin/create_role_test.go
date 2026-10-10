package legacylogin

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestInspectCreateRoleRequest(t *testing.T) {
	p := make([]byte, CreateRoleFixedSize)
	p[0] = ClientCreateRoleOpcode
	binary.LittleEndian.PutUint32(p[1:5], 77)
	name := []uint16{'A', 'l', 'e', 'x', 0}
	for i, u := range name {
		binary.LittleEndian.PutUint16(p[5+i*2:5+i*2+2], u)
	}
	binary.LittleEndian.PutUint16(p[71:73], 0)

	m, err := InspectCreateRoleRequest(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.Field0 != 77 || m.NameUTF16Units != 4 || m.ExtraParamCount != 0 || m.PayloadBytes != CreateRoleFixedSize {
		t.Fatalf("unexpected metadata: %#v", m)
	}
}

func TestInspectCreateRoleAllowsFullFixedNameArea(t *testing.T) {
	p := make([]byte, CreateRoleFixedSize)
	p[0] = ClientCreateRoleOpcode
	for i := 0; i < CreateRoleNameUnits; i++ {
		binary.LittleEndian.PutUint16(p[5+i*2:5+i*2+2], uint16('A'+i%26))
	}
	m, err := InspectCreateRoleRequest(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.NameUTF16Units != CreateRoleNameUnits {
		t.Fatalf("name units=%d want=%d", m.NameUTF16Units, CreateRoleNameUnits)
	}
}

func TestInspectCreateRoleNeedsFixedPrefix(t *testing.T) {
	p := make([]byte, CreateRoleFixedSize-1)
	p[0] = ClientCreateRoleOpcode
	if _, err := InspectCreateRoleRequest(p); !errors.Is(err, ErrNeedMore) {
		t.Fatalf("err=%v want ErrNeedMore", err)
	}
}
