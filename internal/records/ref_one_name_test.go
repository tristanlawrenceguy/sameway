package records_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A field that holds one record takes its name as well as its id: an
// entry logged against "Running miles", a call with "Sam"; a name that is
// no one is refused, nobody made up.
func TestOneRecordIsFoundByItsName(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	habit, _, _ := records.Write(st, "created", "habit", "", map[string]any{"name": "Running miles"})
	entry, _, err := records.Write(st, "created", "entry", "", map[string]any{"habit": "running miles", "amount": 4, "at": "2026-10-16"})
	if err != nil || entry.Fields["habit"] != habit.ID {
		t.Errorf("the habit by its name: %v %v", err, entry)
	}
	sam, _, _ := records.Write(st, "created", "person", "", map[string]any{"name": "Sam"})
	call, _, err := records.Write(st, "created", "interaction", "", map[string]any{"summary": "Call", "person": "Sam"})
	if err != nil || call.Fields["person"] != sam.ID {
		t.Errorf("the person by their name: %v %v", err, call)
	}
	if _, _, err := records.Write(st, "created", "interaction", "", map[string]any{"summary": "Call", "person": "Nobody Here"}); err == nil || !strings.Contains(err.Error(), "no person is called") {
		t.Errorf("a name that is no one is refused: %v", err)
	}
}
