package core

import "testing"

func TestTwoPlayersFormMatch(t *testing.T) {
	s := NewState()
	a := s.CreateGuest("A")
	b := s.CreateGuest("B")

	m, _, err := s.Enqueue(a.ID, "legacy-duo", 2, "127.0.0.1", 7000)
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("match formed too early")
	}

	m, queued, err := s.Enqueue(b.ID, "legacy-duo", 2, "127.0.0.1", 7000)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected match")
	}
	if queued != 0 {
		t.Fatalf("expected empty queue, got %d", queued)
	}
	if len(m.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(m.Players))
	}

	got, ok := s.MatchForPlayer(a.ID)
	if !ok || got.ID != m.ID {
		t.Fatal("player A not attached to match")
	}
}
