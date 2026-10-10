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

const maxMemberWirePayload = 64 * 1024

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

// readMemberFrame reads one real LenSplitProtocol frame from the legacy TCP
// connection. It does not interpret or retain credential fields.
func readMemberFrame(conn net.Conn) ([]byte, int, error) {
	var header [legacylogin.WireHeaderSize]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, 0, err
	}
	n := binary.LittleEndian.Uint32(header[:])
	if n == 0 || n > maxMemberWirePayload {
		return nil, 0, fmt.Errorf("invalid LenSplit payload length %d", n)
	}
	payload := make([]byte, int(n))
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, 0, err
	}
	return payload, legacylogin.WireHeaderSize + int(n), nil
}

// readLoginPacket remains available for deterministic protocol tests and the
// raw laboratory harness. Real archived clients enter through readMemberFrame.
func readLoginPacket(conn net.Conn) (loginReadResult, error) {
	var result loginReadResult
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

func logLoginMetadata(remote string, wireBytes int, meta legacylogin.LoginMetadata) {
	log.Printf(
		"member %s: CLIENT_LOGIN wire_bytes=%d account_len=%d password_len=%d login_string_len=%d login_type=%d in_stub=%t device_id_len=%d prop_md5_len=%d record_md5_len=%d name_len=%d os_type=%d",
		remote,
		wireBytes,
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
}

func handle(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	for frameNo := 1; frameNo <= 8; frameNo++ {
		payload, wireBytes, err := readMemberFrame(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("member %s: handshake idle after %d frame(s)", remote, frameNo-1)
				return
			}
			if errors.Is(err, io.EOF) {
				return
			}
			log.Printf("member %s: wire frame failed: %v", remote, err)
			return
		}
		if len(payload) == 0 {
			log.Printf("member %s: empty payload", remote)
			return
		}

		switch payload[0] {
		case legacylogin.ClientGetVerifyOpcode:
			if len(payload) != 1 {
				log.Printf("member %s: CLIENT_GET_VERIFY unexpected payload size=%d", remote, len(payload))
				return
			}
			log.Printf("member %s: CLIENT_GET_VERIFY validated wire_bytes=%d", remote, wireBytes)
			log.Printf("member %s: waiting to emit SERVER_SET_VERIFY until recovered field semantics/order are confirmed", remote)

		case legacylogin.ClientRetEncodeOpcode:
			// The body grammar is still under recovery. Do not print its bytes: it
			// may include key/transform material.
			log.Printf("member %s: CLIENT_RET_ENCODE observed payload_size=%d", remote, len(payload))

		case legacylogin.ClientLoginOpcode:
			meta, consumed, parseErr := legacylogin.InspectLoginRequest(payload)
			if parseErr != nil {
				log.Printf("member %s: CLIENT_LOGIN validation failed: %v", remote, parseErr)
				return
			}
			if consumed != len(payload) {
				log.Printf("member %s: CLIENT_LOGIN trailing_bytes=%d", remote, len(payload)-consumed)
			}
			logLoginMetadata(remote, wireBytes, meta)

		default:
			log.Printf("member %s: unknown client opcode=0x%02x payload_size=%d", remote, payload[0], len(payload))
		}
	}
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
