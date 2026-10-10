package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SrTristeSad/survival-heroes-legacy-server/internal/legacyquery"
)

func envUint32(name string, fallback uint32) uint32 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		log.Printf("invalid %s=%q; using %d", name, raw, fallback)
		return fallback
	}
	return uint32(v)
}

func serverRecords() []string {
	// The exact textual server-record format is being recovered from
	// WriteServerListFile. Until then the safe default is an empty list.
	// This override is intentionally an expert/debug escape hatch only.
	raw := os.Getenv("QUERY_SERVER_RECORDS")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "||")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func handle(conn net.Conn, expectedIssuer, expectedGameID uint32, records []string) {
	remote := conn.RemoteAddr().String()
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))

	buf := make([]byte, legacyquery.RequestSize)
	if _, err := io.ReadFull(conn, buf); err != nil {
		log.Printf("query %s: read request: %v", remote, err)
		return
	}

	req, err := legacyquery.ParseRequest(buf)
	if err != nil {
		log.Printf("query %s: parse: %v", remote, err)
		return
	}
	log.Printf(
		"query %s: header=%d cmd=%d v=%d/%d magic=%d issuer=%d game=%d",
		remote, req.Header0, req.Command, req.Ver0, req.Ver1, req.Header1, req.Issuer, req.GameID,
	)

	if !req.IsKnownShape() {
		log.Printf("query %s: unknown request shape; closing without response", remote)
		return
	}
	if req.Issuer != expectedIssuer || req.GameID != expectedGameID {
		log.Printf(
			"query %s: unexpected issuer/game (got %d/%d want %d/%d); responding for compatibility",
			remote, req.Issuer, req.GameID, expectedIssuer, expectedGameID,
		)
	}

	response, err := legacyquery.BuildResponse(records)
	if err != nil {
		log.Printf("query %s: build response: %v", remote, err)
		return
	}
	if _, err := conn.Write(response); err != nil {
		log.Printf("query %s: write response: %v", remote, err)
		return
	}
	log.Printf("query %s: sent %d bytes with %d server record(s)", remote, len(response), len(records))
}

func main() {
	addr := strings.TrimSpace(os.Getenv("QUERY_ADDR"))
	if addr == "" {
		addr = ":8088"
	}
	expectedIssuer := envUint32("QUERY_ISSUER", 10000)
	expectedGameID := envUint32("QUERY_GAME_ID", 125)
	records := serverRecords()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf(
		"legacy query server listening on %s (issuer=%d game=%d records=%d)",
		addr, expectedIssuer, expectedGameID, len(records),
	)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn, expectedIssuer, expectedGameID, records)
	}
}
