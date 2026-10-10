package legacylogin

import (
	"encoding/binary"
	"fmt"
)

// LoginMetadata describes the recovered packet shape without retaining
// account credentials, login tokens or device identifiers.
type LoginMetadata struct {
	AccountBytes     int
	PasswordBytes    int
	LoginStringBytes int
	LoginType        uint32
	InStub           bool
	DeviceIDBytes    int
	PropMD5Bytes     int
	RecordMD5Bytes   int
	NameBytes        int
	OSType           uint32
}

func inspectBlock(data []byte, off *int, label string) (int, error) {
	if err := need(data, *off, 4); err != nil {
		return 0, err
	}
	n := binary.LittleEndian.Uint32(data[*off : *off+4])
	*off += 4
	if n < 16 || n%16 != 0 {
		return 0, fmt.Errorf("%s padded length %d is not a recovered 16-byte block", label, n)
	}
	if n > MaxPacketSize {
		return 0, fmt.Errorf("%s padded length %d exceeds packet buffer", label, n)
	}
	if err := need(data, *off, int(n)); err != nil {
		return 0, err
	}
	block := data[*off : *off+int(n)]
	*off += int(n)
	length := 0
	for length < len(block) && block[length] != 0 {
		length++
	}
	return length, nil
}

// InspectLoginRequest validates the recovered 3.3.3 member-login request and
// returns only non-secret metadata. It never converts credential/token/device
// blocks into strings.
func InspectLoginRequest(data []byte) (LoginMetadata, int, error) {
	var m LoginMetadata
	if len(data) < HeaderSize {
		return m, 0, ErrNeedMore
	}
	if data[0] != PacketKind {
		return m, 0, fmt.Errorf("unexpected login packet kind %d", data[0])
	}
	if got := binary.LittleEndian.Uint32(data[1:5]); got != PacketFamily {
		return m, 0, fmt.Errorf("unexpected login packet family 0x%x", got)
	}
	for i := 5; i < HeaderSize; i++ {
		if data[i] != 0 {
			return m, 0, fmt.Errorf("unexpected non-zero recovered header byte at offset %d", i)
		}
	}

	off := HeaderSize
	var err error
	if m.AccountBytes, err = inspectBlock(data, &off, "account"); err != nil {
		return m, 0, err
	}
	if m.PasswordBytes, err = inspectBlock(data, &off, "password"); err != nil {
		return m, 0, err
	}
	if m.LoginStringBytes, err = inspectBlock(data, &off, "login string"); err != nil {
		return m, 0, err
	}
	if err = need(data, off, 5); err != nil {
		return m, 0, err
	}
	m.LoginType = binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	m.InStub = data[off] != 0
	off++
	if m.DeviceIDBytes, err = inspectBlock(data, &off, "device id"); err != nil {
		return m, 0, err
	}
	if m.DeviceIDBytes > MaxDeviceIDBytes {
		return m, 0, fmt.Errorf("device id exceeds %d bytes", MaxDeviceIDBytes)
	}
	if m.PropMD5Bytes, err = inspectBlock(data, &off, "prop md5"); err != nil {
		return m, 0, err
	}
	if m.RecordMD5Bytes, err = inspectBlock(data, &off, "record md5"); err != nil {
		return m, 0, err
	}
	if m.NameBytes, err = inspectBlock(data, &off, "name"); err != nil {
		return m, 0, err
	}

	if err = need(data, off, 4); err != nil {
		return m, 0, err
	}
	infoLen := binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	if infoLen != FixedInfoSize {
		return m, 0, fmt.Errorf("unexpected info block length %d", infoLen)
	}
	if err = need(data, off, int(FixedInfoSize)+4); err != nil {
		return m, 0, err
	}
	off += int(FixedInfoSize)
	m.OSType = binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	return m, off, nil
}
