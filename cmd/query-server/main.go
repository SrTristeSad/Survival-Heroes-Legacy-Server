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

func envInt32(name string, fallback int32) (int32, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

func structuredServerRecord() (string, bool) {
	host := strings.TrimSpace(os.Getenv("LEGACY_SERVER_HOST"))
	portRaw := strings.TrimSpace(os.Getenv("LEGACY_SERVER_PORT"))
	if host == "" || portRaw == "" {
		return "", false
	}

	port64, err := strconv.ParseInt(portRaw, 10, 32)
	if err != nil {
		log.Printf("invalid LEGACY_SERVER_PORT=%q: %v", portRaw, err)
		return "", false
	}
	typeValue, err := envInt32("LEGACY_SERVER_TYPE", int32(legacyquery.ServerTypeRecommended))
	if err != nil {
		log.Printf("invalid LEGACY_SERVER_TYPE: %v", err)
		return "", false
	}
	openValue, err := envInt32("LEGACY_SERVER_OPEN", 1)
	if err != nil {
		log.Printf("invalid LEGACY_SERVER_OPEN: %v", err)
		return "", false
	}
	loadValue, err := envInt32("LEGACY_SERVER_LOAD", int32(legacyquery.ServerLoadGood))
	if err != nil {
		log.Printf("invalid LEGACY_SERVER_LOAD: %v", err)
		return "", false
	}
	priority, err := envInt32("LEGACY_SERVER_PRIORITY", 0)
	if err != nil {
		log.Printf("invalid LEGACY_SERVER_PRIORITY: %v", err)
		return "", false
	}

	id := strings.TrimSpace(os.Getenv("LEGACY_SERVER_ID"))
	if id == "" {
		id = "legacy-1"
	}
	district := strings.TrimSpace(os.Getenv("LEGACY_SERVER_DISTRICT"))
	if district == "" {
		district = "Legacy"
	}

	record := legacyquery.ServerRecord{
		ID:       id,
		District: district,
		Address:  host,
		Port:     int32(port64),
		Type:     legacyquery.ServerType(typeValue),
		Open:     openValue,
		Load:     legacyquery.ServerLoad(loadValue),
		Priority: priority,
	}
	line, err := record.MarshalLine()
	if err != nil {
		log.Printf("invalid structured legacy server record: %v", err)
		return "", false
	}
	return line, true
}

func serverRecords() []string {
	// Raw records remain available as an expert/debug override for protocol
	// research. Normal deployments should use the structured LEGACY_SERVER_*
	// variables below instead.
	if raw := strings.TrimSpace(os.Getenv("QUERY_SERVER_RECORDS")); raw != "" {
		parts := strings.Split(raw, "||")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				if _, err := legacyquery.ParseServerRecordLine(p); err != nil {
					log.Printf("ignoring invalid QUERY_SERVER_RECORDS entry %q: %v", p, err)
					continue
				}
				out = append(out, p)
			}
		}
		return out
	}

	if line, ok := structuredServerRecord(); ok {
		return []string{line}
	}
	return nil
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
