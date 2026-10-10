package legacylogin

import (
	"errors"
	"testing"
)

func TestLoginRequestRoundTrip(t *testing.T) {
	in := LoginRequest{
		Account:     "alex",
		Password:    "secret",
		LoginString: "token",
		LoginType:   201,
		InStub:      true,
		DeviceID:    "device-123",
		PropMD5:     "0123456789abcdef",
		RecordMD5:   "fedcba9876543210",
		Name:        "android",
		OSType:      4,
	}
	packet, err := BuildLoginRequest(in)
	if err != nil {
		t.Fatal(err)
	}
	if packet[0] != 2 || len(packet) > MaxPacketSize {
		t.Fatalf("bad packet shape")
	}
	got, consumed, err := ParseLoginRequest(packet)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != len(packet) {
		t.Fatalf("consumed=%d len=%d", consumed, len(packet))
	}
	if got.Account != in.Account || got.Password != in.Password || got.LoginString != in.LoginString || got.LoginType != in.LoginType || got.InStub != in.InStub || got.DeviceID != in.DeviceID || got.PropMD5 != in.PropMD5 || got.RecordMD5 != in.RecordMD5 || got.Name != in.Name || got.OSType != in.OSType {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

func TestInspectLoginRequest(t *testing.T) {
	in := LoginRequest{
		Account:     "alex",
		Password:    "secret",
		LoginString: "token",
		LoginType:   201,
		InStub:      true,
		DeviceID:    "device-123",
		PropMD5:     "0123456789abcdef",
		RecordMD5:   "fedcba9876543210",
		Name:        "android",
		OSType:      4,
	}
	packet, err := BuildLoginRequest(in)
	if err != nil {
		t.Fatal(err)
	}
	meta, consumed, err := InspectLoginRequest(packet)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != len(packet) {
		t.Fatalf("consumed=%d len=%d", consumed, len(packet))
	}
	if meta.AccountBytes != len(in.Account) || meta.PasswordBytes != len(in.Password) || meta.LoginStringBytes != len(in.LoginString) || meta.DeviceIDBytes != len(in.DeviceID) || meta.PropMD5Bytes != len(in.PropMD5) || meta.RecordMD5Bytes != len(in.RecordMD5) || meta.NameBytes != len(in.Name) {
		t.Fatalf("unexpected metadata lengths: %#v", meta)
	}
	if meta.LoginType != in.LoginType || meta.InStub != in.InStub || meta.OSType != in.OSType {
		t.Fatalf("unexpected metadata values: %#v", meta)
	}
}

func TestLoginRequestNeedsMore(t *testing.T) {
	packet, err := BuildLoginRequest(LoginRequest{Account: "a", Password: "b", LoginString: "c", DeviceID: "d", PropMD5: "e", RecordMD5: "f", Name: "g"})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, 20, 21, len(packet) - 1} {
		_, _, err := ParseLoginRequest(packet[:n])
		if !errors.Is(err, ErrNeedMore) {
			t.Fatalf("n=%d err=%v", n, err)
		}
		_, _, err = InspectLoginRequest(packet[:n])
		if !errors.Is(err, ErrNeedMore) {
			t.Fatalf("inspect n=%d err=%v", n, err)
		}
	}
}

func TestRejectsBadHeaderAndDeviceID(t *testing.T) {
	packet, err := BuildLoginRequest(LoginRequest{})
	if err != nil {
		t.Fatal(err)
	}
	packet[0] = 9
	if _, _, err := ParseLoginRequest(packet); err == nil {
		t.Fatal("expected bad header error")
	}
	if _, _, err := InspectLoginRequest(packet); err == nil {
		t.Fatal("expected inspector bad header error")
	}
	if _, err := BuildLoginRequest(LoginRequest{DeviceID: string(make([]byte, MaxDeviceIDBytes+1))}); err == nil {
		t.Fatal("expected device id error")
	}
}
