package main

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/SrTristeSad/survival-heroes-legacy-server/internal/legacylogin"
)

func TestLoadVerifyReplyConfigLoginSuccessFlag(t *testing.T) {
	t.Setenv("MEMBER_SEND_SET_VERIFY", "true")
	t.Setenv("MEMBER_SET_VERIFY_MODE", "0")
	t.Setenv("MEMBER_SEND_LOGIN_SUCCESS", "true")
	cfg, err := loadVerifyReplyConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SendLoginSuccess {
		t.Fatal("login success experiment was not enabled")
	}
}

func TestSendZeroRoleLoginSuccess(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() { done <- sendZeroRoleLoginSuccess(server, "test") }()

	var header [legacylogin.WireHeaderSize]byte
	if _, err := client.Read(header[:]); err != nil {
		t.Fatal(err)
	}
	n := binary.LittleEndian.Uint32(header[:])
	if n != legacylogin.ServerLoginSuccessSize {
		t.Fatalf("payload length=%d want=%d", n, legacylogin.ServerLoginSuccessSize)
	}
	payload := make([]byte, int(n))
	if _, err := client.Read(payload); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if payload[0] != legacylogin.ServerLoginSuccessOpcode {
		t.Fatalf("opcode=%02x", payload[0])
	}
	for i, b := range payload[1:] {
		if b != 0 {
			t.Fatalf("byte %d=%02x; zero-role fixed header should be zero-filled", i+1, b)
		}
	}
}
