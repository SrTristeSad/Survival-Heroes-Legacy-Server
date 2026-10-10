package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SrTristeSad/survival-heroes-legacy-server/internal/core"
)

type server struct{ state *core.State }

type guestReq struct {
	Name string `json:"name"`
}
type queueReq struct {
	Mode string `json:"mode"`
}

func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func token(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (s *server) player(r *http.Request) (core.Player, bool) {
	t := token(r)
	if t == "" {
		return core.Player{}, false
	}
	return s.state.PlayerByToken(t)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, 200, map[string]any{"ok": true, "service": "gateway", "phase": 0, "time": time.Now().UTC()})
}

func (s *server) guest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonOut(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req guestReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "Survivor"
	}
	if len(req.Name) > 24 {
		req.Name = req.Name[:24]
	}
	jsonOut(w, 201, s.state.CreateGuest(req.Name))
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	p, ok := s.player(r)
	if !ok {
		jsonOut(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	p.Token = ""
	jsonOut(w, 200, p)
}

func (s *server) enqueue(w http.ResponseWriter, r *http.Request) {
	p, ok := s.player(r)
	if !ok {
		jsonOut(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	var req queueReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Mode == "" {
		req.Mode = "legacy-duo"
	}
	size := envInt("MATCH_SIZE", 2)
	host := env("GAME_PUBLIC_HOST", "127.0.0.1")
	port := envInt("GAME_PORT", 7000)
	m, queued, err := s.state.Enqueue(p.ID, req.Mode, size, host, port)
	if err != nil {
		jsonOut(w, 409, map[string]any{"error": err.Error(), "queued": queued})
		return
	}
	jsonOut(w, 200, map[string]any{"queued": queued, "match": m})
}

func (s *server) queueStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := s.player(r)
	if !ok {
		jsonOut(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	if m, found := s.state.MatchForPlayer(p.ID); found {
		jsonOut(w, 200, map[string]any{"status": "matched", "match": m})
		return
	}
	jsonOut(w, 200, map[string]string{"status": "waiting"})
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			return n
		}
	}
	return d
}

func main() {
	s := &server{state: core.NewState()}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/v1/auth/guest", s.guest)
	mux.HandleFunc("/v1/me", s.me)
	mux.HandleFunc("/v1/matchmaking/enqueue", s.enqueue)
	mux.HandleFunc("/v1/matchmaking/status", s.queueStatus)
	addr := env("GATEWAY_ADDR", ":8080")
	log.Printf("gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(mux)))
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
