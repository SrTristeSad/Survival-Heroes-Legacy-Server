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

const (
	maxMemberWirePayload = 64 * 1024
	memberIdleTimeout    = 5 * time.Minute
	maxObservedFrames    = 64
)

type loginReadResult struct {
	Meta      legacylogin.LoginMetadata
	Consumed  int
	Transport string
}

type verifyReplyConfig struct {
	Enabled          bool
	SendSetEncode    bool
	SendLoginSuccess bool
	SendWorldInfo    bool
	WorldInfoField0  uint16
	WorldInfoText    string
	SendSceneTables  bool
	SendEntryScene   bool
	EntrySceneField0 uint32
	EntrySceneA      uint32
	EntrySceneB      uint32
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

func envUint16(name string, fallback uint16) (uint16, error) {
	v, err := envUint32(name, uint32(fallback))
	if err != nil {
		return 0, err
	}
	if v > 0xffff {
		return 0, fmt.Errorf("%s=%d exceeds uint16", name, v)
	}
	return uint16(v), nil
}

func loadVerifyReplyConfig() (verifyReplyConfig, error) {
	var cfg verifyReplyConfig
	cfg.Enabled = envBool("MEMBER_SEND_SET_VERIFY", false)
	cfg.SendSetEncode = envBool("MEMBER_SEND_SET_ENCODE", false)
	cfg.SendLoginSuccess = envBool("MEMBER_SEND_LOGIN_SUCCESS", false)
	cfg.SendWorldInfo = envBool("MEMBER_SEND_WORLD_INFO", false)
	cfg.WorldInfoText = os.Getenv("MEMBER_WORLD_INFO_TEXT")
	cfg.SendSceneTables = envBool("MEMBER_SEND_SCENE_TABLES", false)
	cfg.SendEntryScene = envBool("MEMBER_SEND_ENTRY_SCENE", false)

	worldField, err := envUint16("MEMBER_WORLD_INFO_FIELD0", 0)
	if err != nil {
		return cfg, err
	}
	cfg.WorldInfoField0 = worldField

	entryField, err := envUint32("MEMBER_ENTRY_SCENE_FIELD0", 0)
	if err != nil {
		return cfg, err
	}
	entryA, err := envUint32("MEMBER_ENTRY_SCENE_A", 119)
	if err != nil {
		return cfg, err
	}
	entryB, err := envUint32("MEMBER_ENTRY_SCENE_B", 0)
	if err != nil {
		return cfg, err
	}
	cfg.EntrySceneField0 = entryField
	cfg.EntrySceneA = entryA
	cfg.EntrySceneB = entryB

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

func sendWorldInfo(conn net.Conn, remote string, cfg verifyReplyConfig) error {
	payload := legacylogin.BuildServerWorldInfo(cfg.WorldInfoField0, cfg.WorldInfoText)
	wireBytes, err := writeFramedPayload(conn, payload)
	if err != nil {
		return err
	}
	log.Printf(
		"member %s: sent experimental SERVER_WORLD_INFO field0=%d text_runes=%d payload_bytes=%d wire_bytes=%d",
		remote, cfg.WorldInfoField0, len([]rune(cfg.WorldInfoText)), len(payload), wireBytes,
	)
	return nil
}

func sendSceneTables(conn net.Conn, remote string) error {
	property := legacylogin.BuildEmptyServerPropertyTable()
	propertyWire, err := writeFramedPayload(conn, property)
	if err != nil {
		return err
	}
	record := legacylogin.BuildEmptyServerRecordTable()
	recordWire, err := writeFramedPayload(conn, record)
	if err != nil {
		return err
	}
	log.Printf(
		"member %s: sent experimental empty SERVER_PROPERTY_TABLE and SERVER_RECORD_TABLE wire_bytes=%d/%d",
		remote, propertyWire, recordWire,
	)
	return nil
}

func sendEntryScene(conn net.Conn, remote string, cfg verifyReplyConfig) error {
	payload := legacylogin.BuildServerEntrySceneZeroProperties(
		cfg.EntrySceneField0,
		cfg.EntrySceneA,
		cfg.EntrySceneB,
	)
	wireBytes, err := writeFramedPayload(conn, payload)
	if err != nil {
		return err
	}
	log.Printf(
		"member %s: sent experimental SERVER_ENTRY_SCENE key=%d-%d field0=%d properties=0 payload_bytes=%d wire_bytes=%d",
		remote, cfg.EntrySceneA, cfg.EntrySceneB, cfg.EntrySceneField0, len(payload), wireBytes,
	)
	return nil
}

func handle(conn net.Conn, verifyCfg verifyReplyConfig) {
	remote := conn.RemoteAddr().String()
	defer conn.Close()

	for frameNo := 1; frameNo <= maxObservedFrames; frameNo++ {
		_ = conn.SetDeadline(time.Now().Add(memberIdleTimeout))
		payload, wireBytes, err := readMemberFrame(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("member %s: session idle after %d frame(s)", remote, frameNo-1)
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

		case legacylogin.ClientWorldInfoOpcode:
			meta, parseErr := legacylogin.InspectClientWorldInfo(payload)
			if parseErr != nil {
				log.Printf("member %s: CLIENT_WORLD_INFO validation failed: %v", remote, parseErr)
				return
			}
			log.Printf(
				"member %s: CLIENT_WORLD_INFO wire_bytes=%d field0=%d text_utf32_units=%d payload_bytes=%d",
				remote, wireBytes, meta.Field0, meta.TextUTF32Units, meta.PayloadBytes,
			)
			if verifyCfg.SendWorldInfo {
				if err := sendWorldInfo(conn, remote, verifyCfg); err != nil {
					log.Printf("member %s: SERVER_WORLD_INFO write failed: %v", remote, err)
					return
				}
			}

		case legacylogin.ClientChooseRoleOpcode:
			meta, parseErr := legacylogin.InspectChooseRole(payload)
			if parseErr != nil {
				log.Printf("member %s: CLIENT_CHOOSE_ROLE validation failed: %v", remote, parseErr)
				return
			}
			log.Printf(
				"member %s: CLIENT_CHOOSE_ROLE wire_bytes=%d role_utf16_units=%d scene_info_bytes=%d payload_bytes=%d",
				remote, wireBytes, meta.RoleUTF16Units, meta.SceneInfoBytes, meta.PayloadBytes,
			)
			if verifyCfg.SendSceneTables || verifyCfg.SendEntryScene {
				if err := sendSceneTables(conn, remote); err != nil {
					log.Printf("member %s: scene table bootstrap failed: %v", remote, err)
					return
				}
			}
			if verifyCfg.SendEntryScene {
				if err := sendEntryScene(conn, remote, verifyCfg); err != nil {
					log.Printf("member %s: SERVER_ENTRY_SCENE write failed: %v", remote, err)
					return
				}
			}

		case legacylogin.ClientCreateRoleOpcode:
			meta, parseErr := legacylogin.InspectCreateRoleRequest(payload)
			if parseErr != nil {
				log.Printf("member %s: CLIENT_CREATE_ROLE validation failed: %v", remote, parseErr)
				return
			}
			log.Printf(
				"member %s: CLIENT_CREATE_ROLE wire_bytes=%d field0=%d name_utf16_units=%d extra_params=%d payload_bytes=%d",
				remote, wireBytes, meta.Field0, meta.NameUTF16Units, meta.ExtraParamCount, meta.PayloadBytes,
			)

		default:
			log.Printf("member %s: unknown client opcode=0x%02x payload_size=%d", remote, payload[0], len(payload))
		}
	}
	log.Printf("member %s: observation frame limit reached (%d)", remote, maxObservedFrames)
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
	log.Printf(
		"legacy member/login compatibility server listening on %s (set_verify=%t set_encode=%t login_success=%t world_info=%t scene_tables=%t entry_scene=%t scene=%d-%d mode=%d field0=%d)",
		addr,
		verifyCfg.Enabled,
		verifyCfg.SendSetEncode,
		verifyCfg.SendLoginSuccess,
		verifyCfg.SendWorldInfo,
		verifyCfg.SendSceneTables,
		verifyCfg.SendEntryScene,
		verifyCfg.EntrySceneA,
		verifyCfg.EntrySceneB,
		verifyCfg.Packet.Mode,
		verifyCfg.Packet.Field0,
	)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn, verifyCfg)
	}
}
