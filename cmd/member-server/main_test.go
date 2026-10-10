package main

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/SrTristeSad/survival-heroes-legacy-server/internal/legacylogin"
)

func testLoginPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := legacylogin.BuildLoginRequest(legacylogin.LoginRequest{
		Account:     "tester",
		Password:    "not-logged",
		LoginString: "research-token",
		LoginType:   100,
		DeviceID:    "test-device",
		PropMD5:     "prop",
		RecordMD5:   "record",
		Name:        "android",
		OSType:      4,
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func runRead(t *testing.T, wire []byte) loginReadResult {
	t.Helper()
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		_, err := client.Write(wire)
		done <- err
	}()

	got, err := readLoginPacket(server)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	return got
}

func TestReadMemberFrameGetVerify(t *testing.T) {
	wire, err := legacylogin.FrameWirePayload(legacylogin.BuildClientGetVerify())
	if err != nil {
		t.Fatal(err)
	}
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		_, err := client.Write(wire)
		done <- err
	}()

	payload, consumed, err := readMemberFrame(server)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if consumed != len(wire) || len(payload) != 1 || payload[0] != legacylogin.ClientGetVerifyOpcode {
		t.Fatalf("consumed=%d wire=%d payload=%x", consumed, len(wire), payload)
	}
}

func TestReadLoginPacketLenSplit(t *testing.T) {
	payload := testLoginPayload(t)
	wire, err := legacylogin.FrameWirePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	got := runRead(t, wire)
	if got.Transport != "len-split" {
		t.Fatalf("transport=%q", got.Transport)
	}
	if got.Consumed != len(wire) || got.Meta.LoginType != 100 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestReadLoginPacketRawLab(t *testing.T) {
	payload := testLoginPayload(t)
	got := runRead(t, payload)
	if got.Transport != "raw-lab" {
		t.Fatalf("transport=%q", got.Transport)
	}
	if got.Consumed != len(payload) || got.Meta.LoginType != 100 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestLoadVerifyReplyConfigDisabledByDefault(t *testing.T) {
	t.Setenv("MEMBER_SEND_SET_VERIFY", "")
	cfg, err := loadVerifyReplyConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatal("experimental verify reply unexpectedly enabled")
	}
}

func TestLoadVerifyReplyConfigModeOneRequiresKey(t *testing.T) {
	t.Setenv("MEMBER_SEND_SET_VERIFY", "true")
	t.Setenv("MEMBER_SET_VERIFY_MODE", "1")
	t.Setenv("MEMBER_SET_VERIFY_KEY_HEX", "")
	if _, err := loadVerifyReplyConfig(); err == nil {
		t.Fatal("expected missing mode-1 key to fail")
	}
}

func TestSendSetVerifyFramesModeZero(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	cfg := verifyReplyConfig{
		Enabled: true,
		Packet: legacylogin.SetVerifyPacket{
			Field0: 7,
			Mode:   0,
		},
	}

	done := make(chan error, 1)
	go func() { done <- sendSetVerify(server, "test", cfg) }()

	var header [4]byte
	if _, err := client.Read(header[:]); err != nil {
		t.Fatal(err)
	}
	n := binary.LittleEndian.Uint32(header[:])
	if n != legacylogin.ServerSetVerifyBaseSize {
		t.Fatalf("payload length=%d want=%d", n, legacylogin.ServerSetVerifyBaseSize)
	}
	payload := make([]byte, n)
	if _, err := client.Read(payload); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if payload[0] != legacylogin.ServerSetVerifyOpcode {
		t.Fatalf("opcode=%02x", payload[0])
	}
	if got := binary.LittleEndian.Uint32(payload[1:5]); got != 7 {
		t.Fatalf("field0=%d", got)
	}
	if got := binary.LittleEndian.Uint32(payload[5:9]); got != 0 {
		t.Fatalf("mode=%d", got)
	}
}
