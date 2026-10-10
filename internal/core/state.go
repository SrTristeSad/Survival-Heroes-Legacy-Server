package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Player struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type QueueEntry struct {
	PlayerID string    `json:"player_id"`
	Mode     string    `json:"mode"`
	JoinedAt time.Time `json:"joined_at"`
}

type Match struct {
	ID        string    `json:"id"`
	Mode      string    `json:"mode"`
	Players   []string  `json:"players"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
}

type State struct {
	mu       sync.RWMutex
	players  map[string]Player
	byToken  map[string]string
	queue    []QueueEntry
	matches  map[string]Match
	matchFor map[string]string
}

func NewState() *State {
	return &State{
		players:  map[string]Player{},
		byToken:  map[string]string{},
		matches:  map[string]Match{},
		matchFor: map[string]string{},
	}
}

func randomID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return prefix + "fallback"
	}
	return prefix + hex.EncodeToString(b)
}

func (s *State) CreateGuest(name string) Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := Player{ID: randomID("p_"), Name: name, Token: randomID("t_"), CreatedAt: time.Now().UTC()}
	s.players[p.ID] = p
	s.byToken[p.Token] = p.ID
	return p
}

func (s *State) PlayerByToken(token string) (Player, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byToken[token]
	if !ok {
		return Player{}, false
	}
	p, ok := s.players[id]
	return p, ok
}

func (s *State) Enqueue(playerID, mode string, matchSize int, host string, port int) (*Match, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.players[playerID]; !ok {
		return nil, 0, errors.New("unknown player")
	}
	if _, ok := s.matchFor[playerID]; ok {
		return nil, len(s.queue), errors.New("player already matched")
	}
	for _, q := range s.queue {
		if q.PlayerID == playerID {
			return nil, len(s.queue), errors.New("player already queued")
		}
	}
	s.queue = append(s.queue, QueueEntry{PlayerID: playerID, Mode: mode, JoinedAt: time.Now().UTC()})

	indexes := make([]int, 0, matchSize)
	players := make([]string, 0, matchSize)
	for i, q := range s.queue {
		if q.Mode == mode {
			indexes = append(indexes, i)
			players = append(players, q.PlayerID)
			if len(players) == matchSize {
				break
			}
		}
	}
	if len(players) < matchSize {
		return nil, len(s.queue), nil
	}

	remove := map[int]bool{}
	for _, i := range indexes {
		remove[i] = true
	}
	kept := s.queue[:0]
	for i, q := range s.queue {
		if !remove[i] {
			kept = append(kept, q)
		}
	}
	s.queue = kept

	m := Match{ID: randomID("m_"), Mode: mode, Players: players, Host: host, Port: port, CreatedAt: time.Now().UTC()}
	s.matches[m.ID] = m
	for _, id := range players {
		s.matchFor[id] = m.ID
	}
	return &m, len(s.queue), nil
}

func (s *State) MatchForPlayer(playerID string) (Match, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mid, ok := s.matchFor[playerID]
	if !ok {
		return Match{}, false
	}
	m, ok := s.matches[mid]
	return m, ok
}
