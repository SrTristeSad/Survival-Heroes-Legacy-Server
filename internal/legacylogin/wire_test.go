package legacylogin

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestRecoveredHandshakeOpcodes(t *testing.T) {
	if ClientGetVerifyOpcode != 0x00 || ClientRetEncodeOpcode != 0x01 || ClientLoginOpcode != 0x02 {
		t.Fatalf("unexpected client opcodes get_verify=%02x ret_encode=%02x login=%02x", ClientGetVerifyOpcode, ClientRetEncodeOpcode, ClientLoginOpcode)
	}
	if ServerSetVerifyOpcode != 0x01 || ServerSetEncodeOpcode != 0x02 || ServerLoginSuccessOpcode != 0x04 {
		t.Fatalf("unexpected server opcodes set_verify=%02x set_encode=%02x login_success=%02x", ServerSetVerifyOpcode, ServerSetEncodeOpcode, ServerLoginSuccessOpcode)
	}
	if ServerSetEncodeMinSize != 62 {
		t.Fatalf("unexpected SERVER_SET_ENCODE minimum size %d", ServerSetEncodeMinSize)
	}

	payload := BuildClientGetVerify()
	if len(payload) != 1 || payload[0] != ClientGetVerifyOpcode {
		t.Fatalf("GET_VERIFY payload=%x", payload)
	}
	frame, err := FrameWirePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(frame[:4]); got != 1 || len(frame) != 5 || frame[4] != 0 {
		t.Fatalf("GET_VERIFY frame=%x", frame)
	}
}

func TestWireFrameRoundTrip(t *testing.T) {
	payload := []byte{0x04, 1, 2, 3, 4}
	frame, err := FrameWirePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(frame[:4]); got != uint32(len(payload)) {
		t.Fatalf("wire length=%d want=%d", got, len(payload))
	}
	got, consumed, err := ParseWireFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != len(frame) {
		t.Fatalf("consumed=%d len=%d", consumed, len(frame))
	}
	if string(got) != string(payload) {
		t.Fatalf("payload=%x want=%x", got, payload)
	}
}

func TestWireFrameNeedsMore(t *testing.T) {
	frame, err := FrameWirePayload([]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, 1, 3, 4, len(frame) - 1} {
		_, consumed, err := ParseWireFrame(frame[:n])
		if !errors.Is(err, ErrNeedMore) || consumed != 0 {
			t.Fatalf("n=%d consumed=%d err=%v", n, consumed, err)
		}
	}
}

func TestBuildServerSetVerifyModeZero(t *testing.T) {
	payload, err := BuildServerSetVerify(SetVerifyPacket{Field0: 0x11223344, Mode: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) != ServerSetVerifyBaseSize || payload[0] != ServerSetVerifyOpcode {
		t.Fatalf("bad verify shape len=%d opcode=%02x", len(payload), payload[0])
	}
	if got := binary.LittleEndian.Uint32(payload[1:5]); got != 0x11223344 {
		t.Fatalf("field0=%08x", got)
	}
	if got := binary.LittleEndian.Uint32(payload[5:9]); got != 0 {
		t.Fatalf("mode=%d", got)
	}
}

func TestBuildServerSetVerifyModeOne(t *testing.T) {
	var key [ServerSetVerifyKeySize]byte
	for i := range key {
		key[i] = byte(i + 1)
	}
	payload, err := BuildServerSetVerify(SetVerifyPacket{Field0: 7, Mode: 1, Key: key})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) != 25 || payload[0] != ServerSetVerifyOpcode {
		t.Fatalf("bad verify shape len=%d opcode=%02x", len(payload), payload[0])
	}
	if string(payload[9:25]) != string(key[:]) {
		t.Fatalf("key=%x want=%x", payload[9:25], key)
	}
}

func TestBuildMinimalServerLoginSuccess(t *testing.T) {
	var h LoginSuccessHeader
	for i := 0; i < 9; i++ {
		h.Values[i] = uint32(100 + i)
	}
	payload, err := BuildServerLoginSuccessHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) != ServerLoginSuccessSize || payload[0] != ServerLoginSuccessOpcode {
		t.Fatalf("bad success shape len=%d opcode=%02x", len(payload), payload[0])
	}
	for i := 0; i < 10; i++ {
		off := 1 + i*4
		if got := binary.LittleEndian.Uint32(payload[off : off+4]); got != h.Values[i] {
			t.Fatalf("value[%d]=%d want=%d", i, got, h.Values[i])
		}
	}
}

func TestLoginSuccessRejectsUnimplementedRoleRecords(t *testing.T) {
	var h LoginSuccessHeader
	h.Values[9] = 1
	if _, err := BuildServerLoginSuccessHeader(h); err == nil {
		t.Fatal("expected role-record error")
	}
}
