package server

import (
	"testing"
	"time"
)

// TestInLastTurn pins the rule behind the glow: a change marker reports the
// exchange the page is showing, and nothing older. Without the upper bound a
// block edited once would keep glowing on every load until the next message,
// which would turn a transient signal back into permanent chrome.
func TestInLastTurn(t *testing.T) {
	now := time.Now()
	// A slow local model can take minutes to build a page, so the whole
	// exchange counts, not just the instant the reply landed.
	fresh := &conversation{
		LastTurn: now.Add(-90 * time.Second), // the person's message
		TurnEnd:  now.Add(-20 * time.Second), // the reply that closed it
	}
	cases := []struct {
		name string
		when time.Time
		want bool
	}{
		{"early in a slow turn", now.Add(-80 * time.Second), true},
		{"exactly at the person's message", fresh.LastTurn, true},
		{"just after the reply", fresh.TurnEnd.Add(time.Second), true},
		{"before the turn began", now.Add(-10 * time.Minute), false},
		{"a moment ago, still being watched", now.Add(-2 * time.Second), true},
	}
	for _, c := range cases {
		if got := inLastTurn(c.when, fresh); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}

	// The same turn, revisited much later: the page must be calm. Without
	// this the glow would become permanent chrome for anyone who leaves a
	// tab open and comes back.
	stale := &conversation{
		LastTurn: now.Add(-2 * time.Hour),
		TurnEnd:  now.Add(-2*time.Hour + time.Minute),
	}
	if inLastTurn(stale.LastTurn.Add(30*time.Second), stale) {
		t.Errorf("a change from an old exchange should no longer glow")
	}

	// A conversation that has never had a turn marks nothing old.
	empty := &conversation{}
	if inLastTurn(now.Add(-time.Hour), empty) {
		t.Errorf("with no turns, an old change should not be marked")
	}
	if !inLastTurn(now, empty) {
		t.Errorf("a change happening right now should still be marked")
	}
}
