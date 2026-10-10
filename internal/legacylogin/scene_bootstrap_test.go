package legacylogin

import (
	"encoding/binary"
	"testing"
)

func TestEmptyPropertyTable(t *testing.T) {
	p := BuildEmptyServerPropertyTable()
	if len(p) != ServerEmptyTableSize || p[0] != ServerPropertyTableOpcode || binary.LittleEndian.Uint16(p[1:3]) != 0 {
		t.Fatalf("packet=%x", p)
	}
}

func TestEmptyRecordTable(t *testing.T) {
	p := BuildEmptyServerRecordTable()
	if len(p) != ServerEmptyTableSize || p[0] != ServerRecordTableOpcode || binary.LittleEndian.Uint16(p[1:3]) != 0 {
		t.Fatalf("packet=%x", p)
	}
}

func TestEntrySceneZeroProperties(t *testing.T) {
	p := BuildServerEntrySceneZeroProperties(7, 12, 34)
	if len(p) != ServerEntrySceneSize || p[0] != ServerEntrySceneOpcode {
		t.Fatalf("shape=%x", p)
	}
	got, err := InspectServerEntryScenePrefix(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Field0 != 7 || got.SceneA != 12 || got.SceneB != 34 || got.PropertyCount != 0 {
		t.Fatalf("prefix=%#v", got)
	}
}

func TestEntrySceneWireOffsets(t *testing.T) {
	p := BuildServerEntrySceneZeroProperties(0x11223344, 0x55667788, 0x99aabbcc)
	if binary.LittleEndian.Uint32(p[1:5]) != 0x11223344 ||
		binary.LittleEndian.Uint32(p[5:9]) != 0x55667788 ||
		binary.LittleEndian.Uint32(p[9:13]) != 0x99aabbcc ||
		binary.LittleEndian.Uint16(p[13:15]) != 0 {
		t.Fatalf("packet=%x", p)
	}
}
