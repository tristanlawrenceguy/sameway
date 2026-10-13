package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Entry list headings use natural language separators — a comma between habit
// and amount, not a colon. No "(due …)" text is appended to entry titles in
// the list. The page still lists all entries and they remain distinguishable.
func TestEntryListUsesNaturalLanguageHeadings(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	water, err := a.Store.Create(server.HabitType, map[string]any{
		"name": "8 glasses of water a day", "unit": "glasses", "target": 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	read, err := a.Store.Create(server.HabitType, map[string]any{
		"name": "Read", "unit": "minutes", "cadence": "day",
	})
	if err != nil {
		t.Fatal(err)
	}

	at1 := when.Store(time.Now().Add(-time.Minute), false)
	_, err = a.Store.Create(server.EntryType, map[string]any{
		"habit": water.ID, "at": at1, "amount": 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	at2 := when.Store(time.Now().Add(-2*time.Minute), false)
	_, err = a.Store.Create(server.EntryType, map[string]any{
		"habit": read.ID, "at": at2, "amount": 25,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/entry").Body.String()

	// Acceptance 1: natural language headings with comma separator.
	if !strings.Contains(body, "8 glasses of water a day, 1 glass") {
		t.Errorf("the entry list should use comma separators, not colons; want '8 glasses of water a day, 1 glass'\n%s", truncate(said(body)))
	}
	if strings.Contains(body, "8 glasses of water a day: 1 glass") {
		t.Error("the entry list must not show colon-separated machine format\n" + said(body))
	}

	// Acceptance 2: no "(due …)" timestamp appended to entry titles.
	if strings.Contains(said(body), "(due") || strings.Contains(said(body), "Due ") {
		t.Error("entry titles in the list must not have a spurious '(due …)' or 'Due' timestamp\n" + said(body))
	}

	// Acceptance 3: all entries listed and distinguishable.
	if !strings.Contains(said(body), "Read, 25 minutes") {
		t.Errorf("the entry list should show the second entry; got:\n%s", said(body))
	}
	if strings.Count(said(body), "glasses of water a day") < 1 ||
		strings.Count(said(body), "Read, 25 minutes") < 1 {
		t.Error("all entries must be listed and distinguishable by content\n" + said(body))
	}
}
