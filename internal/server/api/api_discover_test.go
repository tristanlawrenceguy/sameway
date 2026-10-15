package api_test

import (
	"strings"
	"testing"
)

// An agent finding where blocks go is told Home is there, though it has no
// record of its own; and what describes the chat and Today says what they
// hold: a turn under way followed from /chat/live, and Today's follow-ups
// as today's, with later ones ahead.
func TestAnAgentIsToldWhatIsThere(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var tabs struct {
		Count int
		Home  struct{ ID, Name, Page string }
	}
	decode(t, get(t, h, "/api/canvas"), &tabs)
	if tabs.Home.Name != "Home" || tabs.Home.Page != "/" || tabs.Home.ID != "" {
		t.Errorf("the tabs say Home is there: %+v", tabs)
	}
	routes := get(t, h, "/api/describe/routes").Body.String()
	for _, want := range []string{"/chat/live", "follow-up is today"} {
		if !strings.Contains(routes, want) {
			t.Errorf("the routes say %q", want)
		}
	}
}
