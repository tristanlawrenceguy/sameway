package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// seedLogWithInternalTypes creates activity entries for every target that
// must be excluded from the What filter dropdown: conversation, entry, field,
// type, action, settings (ui./llm.), and other changes (empty target). It also
// seeds human-readable types so we can verify they remain visible.
func seedLogWithInternalTypes(t *testing.T, a *app.App, h http.Handler) {
	t.Helper()
	records.Record(a.Store, "human", records.Change{Action: "created", Component: "task", ID: "t1", Detail: "Dig the pond"})
	records.Record(a.Store, "human", records.Change{Action: "added", Component: "note", ID: "n1", Detail: "Fern cuttings"})
	records.Record(a.Store, "assistant", records.Change{Action: "added", Component: "reminder", ID: "r1", Detail: "Water the plants"})

	// Targets that must NOT appear in the What filter dropdown.
	records.Record(a.Store, "system", records.Change{Action: "updated", Component: "conversation", ID: "conv1", Detail: "Chat history updated"})
	records.Record(a.Store, "system", records.Change{Action: "created", Component: "entry", ID: "e1", Detail: "Habit log entry created"})
	records.Record(a.Store, "system", records.Change{Action: "added", Component: "field", ID: "f1", Detail: "Field added to schema"})
	records.Record(a.Store, "system", records.Change{Action: "updated", Component: "type", ID: "typ1", Detail: "Content type updated"})
	records.Record(a.Store, "assistant", records.Change{Action: "created", Component: "action", ID: "act1", Detail: "Automation action created"})

	// Settings (ui./llm. targets).
	records.Record(a.Store, "system", records.Change{Action: "set", Component: "ui.text", Detail: "large"})
	records.Record(a.Store, "system", records.Change{Action: "set", Component: "llm.gpt4o", Detail: "model changed"})

	// Other changes (empty target).
	records.Record(a.Store, "assistant", records.Change{Action: "added", Component: "", ID: "x1", Detail: "Misc change with no target"})
}

// The What filter dropdown must not show internal type names.
func TestActivityInternalTypesNotInWhatFilter(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLogWithInternalTypes(t, a, h)
	body := get(t, h, "/activity").Body.String()

	for _, bad := range []string{
		"Conversations", "Entries", "Fields", "Types", "Actions",
		"Settings", "Other changes",
	} {
		if strings.Contains(body, ">"+bad+"</option>") || strings.Contains(body, `"`+bad+`"`) {
			t.Errorf("What filter must not show %q\n%s", bad, truncate(body))
		}
	}
}

// Human-readable labels remain visible and selectable in the What filter.
func TestActivityHumanReadableLabelsStillVisible(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLogWithInternalTypes(t, a, h)
	body := get(t, h, "/activity").Body.String()

	for _, want := range []string{
		`<option value="task">Tasks</option>`,
		`<option value="note">Notes</option>`,
		`<option value="reminder">Reminders</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("human-readable label %q should appear as a What filter option\n%s", want, truncate(body))
		}
	}
}

// The ?kind= URL parameter still works for the now-hidden types.
func TestActivityKindParamStillWorksForHiddenTypes(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLogWithInternalTypes(t, a, h)

	for _, kind := range []string{"conversation", "entry", "field", "type", "action"} {
		body := get(t, h, "/activity?kind="+kind).Body.String()
		if !strings.Contains(body, "<title>Activity: to "+schema.Plural(kind)) {
			t.Errorf("filtering by ?kind=%s should show a result summary with title\n%s", kind, truncate(body))
		}
		// The page should still return 200 and contain the activity entries.
		if strings.Contains(body, "No changes match") {
			t.Errorf("?kind=%s unexpectedly matched zero results\n%s", kind, truncate(body))
		}
	}
}
