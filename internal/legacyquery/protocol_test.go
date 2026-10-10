package legacyquery

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestKnownClientRequest(t *testing.T) {
	want := Request{
		Header0: 24,
		Command: 1001,
		Ver0:    1,
		Ver1:    1,
		Header1: 22222,
		Issuer:  10000,
		GameID:  125,
	}
	wire := want.MarshalBinary()
	if len(wire) != 28 {
		t.Fatalf("request len=%d", len(wire))
	}
	got, err := ParseRequest(wire)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if !got.IsKnownShape() {
		t.Fatal("known request shape rejected")
	}

	wantHex := []byte{
		0x18, 0x00, 0x00, 0x00,
		0xe9, 0x03, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00,
		0xce, 0x56, 0x00, 0x00,
		0x10, 0x27, 0x00, 0x00,
		0x7d, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(wire, wantHex) {
		t.Fatalf("wire=%x want=%x", wire, wantHex)
	}
}

func TestEmptyServerResponse(t *testing.T) {
	wire, err := BuildResponse(nil)
	if err != nil {
		t.Fatal(err)
	}
	// 8-byte envelope + five uint32 values.
	if len(wire) != 28 {
		t.Fatalf("response len=%d", len(wire))
	}
	if got := binary.LittleEndian.Uint32(wire[0:4]); got != 24 {
		t.Fatalf("remaining len=%d", got)
	}
	if got := binary.LittleEndian.Uint32(wire[4:8]); got != 0 {
		t.Fatalf("unpacked size=%d", got)
	}
	if got := binary.LittleEndian.Uint32(wire[24:28]); got != 0 {
		t.Fatalf("server count=%d", got)
	}
}

func TestServerStringsAreNULTerminated(t *testing.T) {
	wire, err := BuildResponse([]string{"one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	body := wire[8:]
	if count := binary.LittleEndian.Uint32(body[16:20]); count != 2 {
		t.Fatalf("server count=%d", count)
	}
	if !bytes.Equal(body[20:], []byte("one\x00two\x00")) {
		t.Fatalf("strings=%q", body[20:])
	}
}
