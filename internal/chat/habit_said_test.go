package chat_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An agent logs and reads a habit as the person does: an entry made says
// where its habit stands now, as the Log press does, and get_record on a
// habit says how it has gone, so no agent works it out from the entries.
func TestAnAgentLogsAndReadsAHabitInWords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	now := time.Date(2026, 3, 11, 15, 0, 0, 0, time.Local)
	svc.Book.Clock = func() time.Time { return now }
	habit, err := svc.Store.Create("habit", map[string]any{"name": "Water", "cadence": "day", "target": 8, "unit": "glasses"})
	if err != nil {
		t.Fatal(err)
	}
	for _, days := range []int{1, 2, 4} {
		svc.Store.Create("entry", map[string]any{"habit": habit.ID, "at": when.Store(now.AddDate(0, 0, -days), false), "amount": 8})
	}
	args, _ := json.Marshal(map[string]any{"type": "entry", "fields": map[string]any{"habit": habit.ID, "at": when.Store(now, false), "amount": 3}})
	text, isErr := svc.Call("create_record", args)
	if isErr || !strings.Contains(text, "Water now: 3 of 8 glasses today. 2 days in a row.") {
		t.Errorf("logging through create_record says where the habit stands now:\n%s", text)
	}
	args, _ = json.Marshal(map[string]any{"type": "habit", "id": habit.ID})
	text, _ = svc.Call("get_record", args)
	var out struct {
		Standing string `json:"standing"`
	}
	json.Unmarshal([]byte(text), &out)
	// From the first entry, four days ago: the days before it are not misses.
	if want := "Reached the target of 8 glasses on 3 of the 5 days since Sat 7 Mar. Best run: 2 days in a row."; !strings.Contains(out.Standing, want) {
		t.Errorf("get_record on a habit says how it has gone, want %q in %q", want, out.Standing)
	}
}
