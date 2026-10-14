package chat_test

import (
	"strings"
	"testing"
)

// Entries are called by their habit and how much ("Running miles, 5
// miles"): several made at once are told apart by semicolons in what the
// log and the receipt say, so three are not read as six names; titles
// with no comma keep commas between them.
func TestSeveralNamesWithCommasAreToldApart(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	habit, _ := svc.Store.Create("habit", map[string]any{"name": "Running miles", "cadence": "day", "unit": "miles"})
	run(t, svc, "create_records", map[string]any{"type": "entry", "records": []map[string]any{
		{"habit": habit.ID, "amount": 5, "at": "2026-10-12T08:00:00Z"},
		{"habit": habit.ID, "amount": 3, "at": "2026-10-13T08:00:00Z"},
	}})
	run(t, svc, "create_records", map[string]any{"type": "task", "records": []map[string]any{{"title": "Email Sarah"}, {"title": "Call mum"}}})
	made := entries(t, svc, "created")
	if len(made) != 2 {
		t.Fatalf("two entries in the log, got %d", len(made))
	}
	if s, _ := made[0].Fields["summary"].(string); !strings.Contains(s, "Running miles, 5 miles; Running miles, 3 miles") {
		t.Errorf("entries are set apart by a semicolon: %q", s)
	}
	if s, _ := made[1].Fields["summary"].(string); !strings.Contains(s, "Email Sarah, Call mum") {
		t.Errorf("titles with no comma keep commas: %q", s)
	}
}
