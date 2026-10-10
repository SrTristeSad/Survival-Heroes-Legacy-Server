package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
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

type verifyReplyConfig struct {
	Enabled          bool
	SendSetEncode    bool
	SendLoginSuccess bool
	Packet           legacylogin.SetVerifyPacket
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("invalid %s=%q; using %t", name, raw, fallback)
		return fallback
	}
	return v
}

func envUint32(name string, fallback uint32) (uint32, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseUint(raw, 0, 32)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", name, raw, err)
	}
	return uint32(v), nil
}

func loadVerifyReplyConfig() (verifyReplyConfig, error) {
	var cfg verifyReplyConfig
	cfg.Enabled = envBool("MEMBER_SEND_SET_VERIFY", false)
	cfg.SendSetEncode = envBool("MEMBER_SEND_SET_ENCODE", false)
	cfg.SendLoginSuccess = envBool("MEMBER_SEND_LOGIN_SUCCESS", false)
	if !cfg.Enabled {
		return cfg, nil
	}

	field0, err := envUint32("MEMBER_SET_VERIFY_FIELD0", 0)
	if err != nil {
		return cfg, err
	}
	mode, err := envUint32("MEMBER_SET_VERIFY_MODE", 0)
	if err != nil {
		return cfg, err
	}
	cfg.Packet.Field0 = field0
	cfg.Packet.Mode = mode

	if mode == 1 {
		raw := strings.TrimSpace(os.Getenv("MEMBER_SET_VERIFY_KEY_HEX"))
		if raw == "" {
			return cfg, errors.New("MEMBER_SET_VERIFY_MODE=1 requires MEMBER_SET_VERIFY_KEY_HEX (16 bytes / 32 hex chars)")
		}
		key, err := hex.DecodeString(raw)
		if err != nil {
			return cfg, fmt.Errorf("MEMBER_SET_VERIFY_KEY_HEX: %w", err)
		}
		if len(key) != legacylogin.ServerSetVerifyKeySize {
			return cfg, fmt.Errorf("MEMBER_SET_VERIFY_KEY_HEX must decode to %d bytes, got %d", legacylogin.ServerSetVerifyKeySize, len(key))
		}
		copy(cfg.Packet.Key[:], key)
	}
	return cfg, nil
}

func looksLikeRawLogin(data []byte) bool {
	if len(data) < 5 {
		return false
	}
	return data[0] == legacylogin.PacketKind && binary.LittleEndian.Uint32(data[1:5]) == legacylogin.PacketFamily
}

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

func writeFramedPayload(conn net.Conn, payload []byte) (int, error) {
	wire, err := legacylogin.FrameWirePayload(payload)
	if err != nil {
		return 0, err
	}
	n, err := conn.Write(wire)
	return n, err
}

func sendSetVerify(conn net.Conn, remote string, cfg verifyReplyConfig) error {
	payload, err := legacylogin.BuildServerSetVerify(cfg.Packet)
	if err != nil {
		return err
	}
	wireBytes, err := writeFramedPayload(conn, payload)
	if err != nil {
		return err
	}
	log.Printf("member %s: sent experimental SERVER_SET_VERIFY field0=%d mode=%d payload_bytes=%d wire_bytes=%d", remote, cfg.Packet.Field0, cfg.Packet.Mode, len(payload), wireBytes)
	return nil
}

func sendSetEncode(conn net.Conn, remote string) error {
	payload := legacylogin.BuildServerSetEncode()
	wireBytes, err := writeFramedPayload(conn, payload)
	if err != nil {
		return err
	}
	log.Printf("member %s: sent experimental SERVER_SET_ENCODE payload_bytes=%d wire_bytes=%d", remote, len(payload), wireBytes)
	return nil
}

func sendZeroRoleLoginSuccess(conn net.Conn, remote string) error {
	// The archived client consumes a 41-byte fixed prefix. The tenth uint32 is
	// role_count. Keeping all fixed values at zero is a controlled experiment
	// intended to reach the client's no-role / create-role path without making
	// up a proprietary character record.
	var header legacylogin.LoginSuccessHeader
	payload, err := legacylogin.BuildServerLoginSuccessHeader(header)
	if err != nil {
		return err
	}
	wireBytes, err := writeFramedPayload(conn, payload)
	if err != nil {
		return err
	}
	log.Printf("member %s: sent experimental SERVER_LOGIN_SUCCESS role_count=0 payload_bytes=%d wire_bytes=%d", remote, len(payload), wireBytes)
	return nil
}

func handle(conn net.Conn, verifyCfg verifyReplyConfig) {
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
			if verifyCfg.Enabled {
				if err := sendSetVerify(conn, remote, verifyCfg); err != nil {
					log.Printf("member %s: SERVER_SET_VERIFY write failed: %v", remote, err)
					return
				}
			} else {
				log.Printf("member %s: SERVER_SET_VERIFY emission disabled; set MEMBER_SEND_SET_VERIFY=true for controlled client tests", remote)
			}

		case legacylogin.ClientRetEncodeOpcode:
			log.Printf("member %s: CLIENT_RET_ENCODE observed payload_size=%d", remote, len(payload))
			if verifyCfg.SendSetEncode {
				if err := sendSetEncode(conn, remote); err != nil {
					log.Printf("member %s: SERVER_SET_ENCODE write failed: %v", remote, err)
					return
				}
			}

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
			if verifyCfg.SendLoginSuccess {
				if err := sendZeroRoleLoginSuccess(conn, remote); err != nil {
					log.Printf("member %s: SERVER_LOGIN_SUCCESS write failed: %v", remote, err)
					return
				}
			}

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
	verifyCfg, err := loadVerifyReplyConfig()
	if err != nil {
		log.Fatalf("invalid verify reply configuration: %v", err)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("legacy member/login compatibility server listening on %s (experimental_set_verify=%t set_encode=%t login_success=%t mode=%d field0=%d)", addr, verifyCfg.Enabled, verifyCfg.SendSetEncode, verifyCfg.SendLoginSuccess, verifyCfg.Packet.Mode, verifyCfg.Packet.Field0)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn, verifyCfg)
	}
}
