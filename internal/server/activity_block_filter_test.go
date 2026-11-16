package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// seedLogWithBlocks makes a log with user-friendly content types and block
// component targets: card, button. The content-type entries should appear as
// filter options; the block components should not.
func seedLogWithBlocks(t *testing.T, a *app.App, h http.Handler) {
	t.Helper()
	records.Record(a.Store, "human", records.Change{Action: "created", Component: "task", ID: "t1", Detail: "Dig the pond"})
	records.Record(a.Store, "assistant", records.Change{Action: "added", Component: "card", ID: "c1", Detail: "Plan the beds"})
	records.Record(a.Store, "human", records.Change{Action: "added", Component: "note", ID: "n1", Detail: "Fern cuttings"})
	records.Record(a.Store, "assistant", records.Change{Action: "created", Component: "reminder", ID: "r1", Detail: "Water the plants"})
	records.Record(a.Store, "assistant", records.Change{Action: "added", Component: "button", ID: "b1", Detail: "Submit button"})
	records.Record(a.Store, "system", records.Change{Action: "failed", Detail: "could not reach the model"})
}

// seedLogWithBlocksRecord also creates a block record change (target="block").
func seedLogWithBlocksRecord(t *testing.T, a *app.App, h http.Handler) {
	t.Helper()
	records.Record(a.Store, "assistant", records.Change{Action: "removed", Component: "block", Detail: "arranged heading"})
}

// Block component names do not appear as What-filter options on /activity.
func TestActivityBlockComponentsNotInWhatFilter(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLogWithBlocks(t, a, h)
	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, `<option value="card">Cards</option>`) {
		t.Error("block component 'card' should not appear as a What filter option\n" + truncate(body))
	}
	if strings.Contains(body, `<option value="button">Buttons</option>`) {
		t.Error("block component 'button' should not appear as a What filter option\n" + truncate(body))
	}

	// User-friendly content types should still appear.
	for _, want := range []string{
		`<option value="task">Tasks</option>`,
		`<option value="note">Notes</option>`,
		`<option value="reminder">Reminders</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("user-friendly type %s should appear as a What filter option\n%s", want, truncate(body))
		}
	}

}

// The block record type (canvas blocks stored as records) does not appear as
// a What-filter option on /activity. A person never uses "Blocks" to describe
// their activity, so it is hidden like other canvas implementation details.
func TestActivityBlockRecordNotInWhatFilter(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLogWithBlocks(t, a, h)
	seedLogWithBlocksRecord(t, a, h)
	body := get(t, h, "/activity").Body.String()

	if strings.Contains(body, `<option value="block">Blocks</option>`) {
		t.Error("block record type should not appear as a What filter option\n" + truncate(body))
	}
}
