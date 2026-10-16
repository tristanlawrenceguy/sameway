package records_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// An event moved to a day said in words keeps its time of day; a date
// written as one, as a page sends it when the time is cleared, is a whole
// day.
func TestADaySaidInWordsKeepsTheTime(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ev, _, err := records.Write(st, "created", "event", "", map[string]any{"title": "Cleanup", "starts": "2026-10-22T09:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	rec, _, err := records.Write(st, "updated", "event", ev.ID, map[string]any{"starts": "October 25"})
	if err != nil {
		t.Fatal(err)
	}
	at, _ := time.Parse(time.RFC3339, rec.Fields["starts"].(string))
	if at.In(st.Now().Location()).Hour() != time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC).In(st.Now().Location()).Hour() || at.Day() != 25 {
		t.Errorf("moved in words, it keeps 9am: %v", rec.Fields["starts"])
	}
	rec, _, _ = records.Write(st, "updated", "event", ev.ID, map[string]any{"starts": "2026-10-26"})
	if s, _ := rec.Fields["starts"].(string); !strings.HasPrefix(s, "2026-10-26T00:00:00") {
		t.Errorf("a date written as one is a whole day: %v", s)
	}
}
