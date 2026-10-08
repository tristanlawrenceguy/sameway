package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// TestGoHttpClientProvenanceSaysAnAgent checks that an activity entry
// attributed to "Go-http-client" is resolved to "an agent" in provenance
// text. This covers acceptance item 2 — detail page ledes show "From: an
// agent" instead of "From: Go-http-client".

func TestGoHttpClientProvenanceSaysAnAgent(t *testing.T) {
	svc := newFullService(t)
	noteID := func(title string) string {
		rec, err := svc.Store.Create("note", map[string]any{"title": title})
		if err != nil {
			t.Fatal(err)
		}
		return rec.ID
	}

	id := noteID("Go-Client Note")

	// Record an activity entry with Go-http-client as the agent.
	records.Record(svc.Store, records.ActorAgent, records.Change{
		Action:    "created",
		Component: "note",
		ID:        id,
		Detail:    "Go-Client Note",
		By:        "Go-http-client",
		Via:       records.ThroughAPI,
	})

	w := svc.Writers()
	got := w.OfID("note", id)

	// Acceptance 2: provenance must not say "Go-http-client".
	if got.Words == "" {
		t.Fatal("provenance for Go-http-client entry is empty")
	}

	// The words should be marked as Outside (an agent wrote it).
	if got.Outside != true {
		t.Errorf("Go-http-client provenance should be marked as Outside\n%+v", got)
	}

	// Specifically verify it does NOT contain the machine-language identifier.
	for _, bad := range []string{"Go-http-client"} {
		if strings.Contains(got.Words, bad) {
			t.Errorf("provenance must not say %q: got %q\n%+v", bad, got.Words, got)
		}
	}

	// The provenance words should mention "agent".
	lower := strings.ToLower(got.Words)
	if !strings.Contains(lower, "agent") {
		t.Errorf("provenance should mention 'agent', got %q\n%+v", got.Words, got)
	}
}
