package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type client struct {
	conn net.Conn
	id   string
}
type hub struct {
	mu      sync.Mutex
	clients map[*client]bool
}

type packet struct {
	Type    string  `json:"type"`
	ID      string  `json:"id,omitempty"`
	X       float64 `json:"x,omitempty"`
	Y       float64 `json:"y,omitempty"`
	Message string  `json:"message,omitempty"`
	Time    int64   `json:"time,omitempty"`
}

func (h *hub) broadcast(v any, except *client) {
	b, _ := json.Marshal(v)
	b = append(b, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if c != except {
			_, _ = c.conn.Write(b)
		}
	}
}

func (h *hub) handle(conn net.Conn) {
	c := &client{conn: conn, id: fmt.Sprintf("c-%d", time.Now().UnixNano())}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		_ = conn.Close()
		h.broadcast(packet{Type: "leave", ID: c.id, Time: time.Now().UnixMilli()}, nil)
	}()

	enc := json.NewEncoder(conn)
	_ = enc.Encode(packet{Type: "welcome", ID: c.id, Message: "Phase-0 protocol placeholder; not the original Survival Heroes protocol.", Time: time.Now().UnixMilli()})
	h.broadcast(packet{Type: "join", ID: c.id, Time: time.Now().UnixMilli()}, c)

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var p packet
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			_ = enc.Encode(packet{Type: "error", Message: "invalid json"})
			continue
		}
		if p.Type == "ping" {
			_ = enc.Encode(packet{Type: "pong", Time: time.Now().UnixMilli()})
			continue
		}
		if p.Type == "move" {
			p.ID = c.id
			p.Time = time.Now().UnixMilli()
			h.broadcast(p, c)
			continue
		}
		_ = enc.Encode(packet{Type: "error", Message: "unsupported packet type"})
	}
}

func main() {
	addr := os.Getenv("GAME_ADDR")
	if addr == "" {
		addr = ":7000"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("game server listening on %s", addr)
	h := &hub{clients: map[*client]bool{}}
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go h.handle(c)
	}
}
