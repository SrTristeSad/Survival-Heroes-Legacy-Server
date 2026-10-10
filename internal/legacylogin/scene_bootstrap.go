package legacylogin

import (
	"encoding/binary"
	"fmt"
)

const (
	ServerPropertyTableOpcode = byte(0x09)
	ServerRecordTableOpcode   = byte(0x0a)
	ServerEntrySceneOpcode    = byte(0x0b)

	ServerEmptyTableSize = 3
	ServerEntrySceneSize = 15
)

// BuildEmptyServerPropertyTable returns the smallest PROPERTY_TABLE message
// accepted by the archived 3.3.3 receiver: opcode plus a zero uint16 count.
func BuildEmptyServerPropertyTable() []byte {
	return []byte{ServerPropertyTableOpcode, 0, 0}
}

// BuildEmptyServerRecordTable returns the smallest RECORD_TABLE message
// accepted by the archived 3.3.3 receiver: opcode plus a zero uint16 count.
func BuildEmptyServerRecordTable() []byte {
	return []byte{ServerRecordTableOpcode, 0, 0}
}

// EntryScenePrefix models the fixed 15-byte prefix consumed by
// GameReceiver::ServerEntryScene. Field0 is present on the wire but is not read
// by the recovered 3.3.3 handler. SceneA and SceneB are formatted by the client
// as "%u-%u" and looked up in its already-loaded scene registry.
type EntryScenePrefix struct {
	Field0        uint32
	SceneA        uint32
	SceneB        uint32
	PropertyCount uint16
}

// BuildServerEntrySceneZeroProperties constructs only the parser-valid fixed
// prefix. It must not be emitted until SceneA/SceneB are known to identify a
// scene present in the client; an unknown pair enters the client's "no scene"
// error path.
func BuildServerEntrySceneZeroProperties(field0, sceneA, sceneB uint32) []byte {
	out := make([]byte, ServerEntrySceneSize)
	out[0] = ServerEntrySceneOpcode
	binary.LittleEndian.PutUint32(out[1:5], field0)
	binary.LittleEndian.PutUint32(out[5:9], sceneA)
	binary.LittleEndian.PutUint32(out[9:13], sceneB)
	binary.LittleEndian.PutUint16(out[13:15], 0)
	return out
}

func InspectServerEntryScenePrefix(data []byte) (EntryScenePrefix, error) {
	var p EntryScenePrefix
	if len(data) < ServerEntrySceneSize {
		return p, ErrNeedMore
	}
	if data[0] != ServerEntrySceneOpcode {
		return p, fmt.Errorf("unexpected entry-scene opcode 0x%02x", data[0])
	}
	p.Field0 = binary.LittleEndian.Uint32(data[1:5])
	p.SceneA = binary.LittleEndian.Uint32(data[5:9])
	p.SceneB = binary.LittleEndian.Uint32(data[9:13])
	p.PropertyCount = binary.LittleEndian.Uint16(data[13:15])
	return p, nil
}
