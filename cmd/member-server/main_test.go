package main

import (
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
