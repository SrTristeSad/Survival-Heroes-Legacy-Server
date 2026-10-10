package legacylogin

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	HeaderSize       = 21
	MaxPacketSize    = 0x800
	PacketKind       = byte(2)
	PacketFamily     = uint32(0x300)
	FixedInfoSize    = uint32(16)
	MaxDeviceIDBytes = 64
)

var ErrNeedMore = errors.New("legacy login packet incomplete")

type LoginRequest struct {
	Account     string
	Password    string
	LoginString string
	LoginType   uint32
	InStub      bool
	DeviceID    string
	PropMD5     string
	RecordMD5   string
	Name        string
	Info        [16]byte
	OSType      uint32
}

func align16Plus16(n int) uint32 {
	v := n + 16
	return uint32((v + 15) &^ 15)
}

func appendBlock(dst []byte, s string) []byte {
	n := align16Plus16(len(s))
	var lenbuf [4]byte
	binary.LittleEndian.PutUint32(lenbuf[:], n)
	dst = append(dst, lenbuf[:]...)
	start := len(dst)
	dst = append(dst, make([]byte, int(n))...)
	copy(dst[start:], s)
	return dst
}

func BuildLoginRequest(r LoginRequest) ([]byte, error) {
	if len(r.DeviceID) > MaxDeviceIDBytes {
		return nil, fmt.Errorf("device id exceeds %d bytes", MaxDeviceIDBytes)
	}
	out := make([]byte, HeaderSize)
	out[0] = PacketKind
	binary.LittleEndian.PutUint32(out[1:5], PacketFamily)
	out = appendBlock(out, r.Account)
	out = appendBlock(out, r.Password)
	out = appendBlock(out, r.LoginString)

	var u32 [4]byte
	binary.LittleEndian.PutUint32(u32[:], r.LoginType)
	out = append(out, u32[:]...)
	if r.InStub {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	out = appendBlock(out, r.DeviceID)
	out = appendBlock(out, r.PropMD5)
	out = appendBlock(out, r.RecordMD5)
	out = appendBlock(out, r.Name)

	binary.LittleEndian.PutUint32(u32[:], FixedInfoSize)
	out = append(out, u32[:]...)
	out = append(out, r.Info[:]...)
	binary.LittleEndian.PutUint32(u32[:], r.OSType)
	out = append(out, u32[:]...)

	if len(out) > MaxPacketSize {
		return nil, fmt.Errorf("login packet size %d exceeds recovered client buffer %d", len(out), MaxPacketSize)
	}
	return out, nil
}

func need(data []byte, offset, n int) error {
	if n < 0 || offset < 0 || offset+n > len(data) {
		return ErrNeedMore
	}
	return nil
}

func readBlock(data []byte, off *int, label string) (string, error) {
	if err := need(data, *off, 4); err != nil {
		return "", err
	}
	n := binary.LittleEndian.Uint32(data[*off : *off+4])
	*off += 4
	if n < 16 || n%16 != 0 {
		return "", fmt.Errorf("%s padded length %d is not a recovered 16-byte block", label, n)
	}
	if n > MaxPacketSize {
		return "", fmt.Errorf("%s padded length %d exceeds packet buffer", label, n)
	}
	if err := need(data, *off, int(n)); err != nil {
		return "", err
	}
	block := data[*off : *off+int(n)]
	*off += int(n)
	end := 0
	for end < len(block) && block[end] != 0 {
		end++
	}
	return string(block[:end]), nil
}

func ParseLoginRequest(data []byte) (LoginRequest, int, error) {
	var r LoginRequest
	if len(data) < HeaderSize {
		return r, 0, ErrNeedMore
	}
	if data[0] != PacketKind {
		return r, 0, fmt.Errorf("unexpected login packet kind %d", data[0])
	}
	if got := binary.LittleEndian.Uint32(data[1:5]); got != PacketFamily {
		return r, 0, fmt.Errorf("unexpected login packet family 0x%x", got)
	}
	for i := 5; i < HeaderSize; i++ {
		if data[i] != 0 {
			return r, 0, fmt.Errorf("unexpected non-zero recovered header byte at offset %d", i)
		}
	}

	off := HeaderSize
	var err error
	if r.Account, err = readBlock(data, &off, "account"); err != nil {
		return r, 0, err
	}
	if r.Password, err = readBlock(data, &off, "password"); err != nil {
		return r, 0, err
	}
	if r.LoginString, err = readBlock(data, &off, "login string"); err != nil {
		return r, 0, err
	}
	if err = need(data, off, 5); err != nil {
		return r, 0, err
	}
	r.LoginType = binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	r.InStub = data[off] != 0
	off++
	if r.DeviceID, err = readBlock(data, &off, "device id"); err != nil {
		return r, 0, err
	}
	if len(r.DeviceID) > MaxDeviceIDBytes {
		return r, 0, fmt.Errorf("device id exceeds %d bytes", MaxDeviceIDBytes)
	}
	if r.PropMD5, err = readBlock(data, &off, "prop md5"); err != nil {
		return r, 0, err
	}
	if r.RecordMD5, err = readBlock(data, &off, "record md5"); err != nil {
		return r, 0, err
	}
	if r.Name, err = readBlock(data, &off, "name"); err != nil {
		return r, 0, err
	}

	if err = need(data, off, 4); err != nil {
		return r, 0, err
	}
	infoLen := binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	if infoLen != FixedInfoSize {
		return r, 0, fmt.Errorf("unexpected info block length %d", infoLen)
	}
	if err = need(data, off, int(FixedInfoSize)+4); err != nil {
		return r, 0, err
	}
	copy(r.Info[:], data[off:off+int(FixedInfoSize)])
	off += int(FixedInfoSize)
	r.OSType = binary.LittleEndian.Uint32(data[off : off+4])
	off += 4
	return r, off, nil
}
