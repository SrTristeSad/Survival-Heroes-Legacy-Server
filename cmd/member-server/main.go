package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/SrTristeSad/survival-heroes-legacy-server/internal/legacylogin"
)

type loginReadResult struct {
	Meta      legacylogin.LoginMetadata
	Consumed  int
	Transport string
}

func looksLikeRawLogin(data []byte) bool {
	if len(data) < 5 {
		return false
	}
	return data[0] == legacylogin.PacketKind && binary.LittleEndian.Uint32(data[1:5]) == legacylogin.PacketFamily
}

func readLoginPacket(conn net.Conn) (loginReadResult, error) {
	var result loginReadResult
	// The recovered login payload itself is bounded by the client's 0x800-byte
	// builder. Allow the 4-byte LenSplit prefix in addition to that payload.
	buf := make([]byte, 0, legacylogin.MaxPacketSize+legacylogin.WireHeaderSize)
	tmp := make([]byte, 512)

	for len(buf) < cap(buf) {
		n, err := conn.Read(tmp)
		if n > 0 {
			room := cap(buf) - len(buf)
			if n > room {
				n = room
			}
			buf = append(buf, tmp[:n]...)

			if looksLikeRawLogin(buf) {
				meta, consumed, parseErr := legacylogin.InspectLoginRequest(buf)
				if parseErr == nil {
					return loginReadResult{Meta: meta, Consumed: consumed, Transport: "raw-lab"}, nil
				}
				if !errors.Is(parseErr, legacylogin.ErrNeedMore) {
					return result, parseErr
				}
			} else if len(buf) >= legacylogin.WireHeaderSize {
				payloadLen := binary.LittleEndian.Uint32(buf[:legacylogin.WireHeaderSize])
				if payloadLen == 0 || payloadLen > legacylogin.MaxPacketSize {
					return result, fmt.Errorf("invalid LenSplit payload length %d", payloadLen)
				}
				payload, consumed, frameErr := legacylogin.ParseWireFrame(buf)
				if frameErr == nil {
					meta, _, parseErr := legacylogin.InspectLoginRequest(payload)
					if parseErr != nil {
						return result, fmt.Errorf("LenSplit payload is not a recovered login request: %w", parseErr)
					}
					return loginReadResult{Meta: meta, Consumed: consumed, Transport: "len-split"}, nil
				}
				if !errors.Is(frameErr, legacylogin.ErrNeedMore) {
					return result, frameErr
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) && len(buf) > 0 {
				return result, fmt.Errorf("connection closed with incomplete login packet (%d bytes)", len(buf))
			}
			return result, err
		}
	}
	return result, fmt.Errorf("login packet exceeded recovered client buffer")
}

func handle(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	result, err := readLoginPacket(conn)
	if err != nil {
		log.Printf("member %s: login framing validation failed: %v", remote, err)
		return
	}
	meta := result.Meta
	log.Printf(
		"member %s: validated login transport=%s wire_bytes=%d account_len=%d password_len=%d login_string_len=%d login_type=%d in_stub=%t device_id_len=%d prop_md5_len=%d record_md5_len=%d name_len=%d os_type=%d",
		remote,
		result.Transport,
		result.Consumed,
		meta.AccountBytes,
		meta.PasswordBytes,
		meta.LoginStringBytes,
		meta.LoginType,
		meta.InStub,
		meta.DeviceIDBytes,
		meta.PropMD5Bytes,
		meta.RecordMD5Bytes,
		meta.NameBytes,
		meta.OSType,
	)
	log.Printf("member %s: login request accepted for protocol research; verify/encode response sequence is not emitted until its ordering is confirmed", remote)
}

func main() {
	addr := strings.TrimSpace(os.Getenv("MEMBER_ADDR"))
	if addr == "" {
		addr = ":7001"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("legacy member/login compatibility server listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn)
	}
}
