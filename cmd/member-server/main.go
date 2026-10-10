package main

import (
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

func readLoginPacket(conn net.Conn) (legacylogin.LoginMetadata, int, error) {
	var meta legacylogin.LoginMetadata
	buf := make([]byte, 0, legacylogin.MaxPacketSize)
	tmp := make([]byte, 512)
	for len(buf) < legacylogin.MaxPacketSize {
		n, err := conn.Read(tmp)
		if n > 0 {
			room := legacylogin.MaxPacketSize - len(buf)
			if n > room {
				n = room
			}
			buf = append(buf, tmp[:n]...)
			parsed, consumed, parseErr := legacylogin.InspectLoginRequest(buf)
			if parseErr == nil {
				return parsed, consumed, nil
			}
			if !errors.Is(parseErr, legacylogin.ErrNeedMore) {
				return meta, 0, parseErr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) && len(buf) > 0 {
				return meta, 0, fmt.Errorf("connection closed with incomplete login packet (%d bytes)", len(buf))
			}
			return meta, 0, err
		}
	}
	return meta, 0, fmt.Errorf("login packet exceeded %d-byte recovered client buffer", legacylogin.MaxPacketSize)
}

func handle(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	meta, consumed, err := readLoginPacket(conn)
	if err != nil {
		log.Printf("member %s: login framing validation failed: %v", remote, err)
		return
	}
	log.Printf(
		"member %s: validated login bytes=%d account_len=%d password_len=%d login_string_len=%d login_type=%d in_stub=%t device_id_len=%d prop_md5_len=%d record_md5_len=%d name_len=%d os_type=%d",
		remote,
		consumed,
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
	log.Printf("member %s: request framing recovered; server response protocol not implemented yet", remote)
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
