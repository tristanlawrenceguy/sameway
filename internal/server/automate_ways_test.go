package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Actions hear a record however it came: imported from a file, which is
// logged as one change, and a reminder ringing, which is the system's own
// and never logged. Both are heard through records.Listen, now the one
// way to hear a change, where they used to be heard by a listener on the
// store itself.
func TestActionsHearRecordsImportedAndRemindersRinging(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Automate.StartWorker()
	got, hook := listen(t)
	a.Store.Create("action", map[string]any{"title": "New task", "kind": "webhook", "url": hook + "/new?t={{title}}", "when": "added", "what": "task"})
	a.Store.Create("action", map[string]any{"title": "Rang", "kind": "webhook", "url": hook + "/rang?t={{title}}", "when": "changed", "what": server.ReminderType, "only": []any{"state=rang"}})

	up, ct := multipartFile(t, "Tasks.csv", "Title\nSow the beans\n", nil)
	file := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/task/import", up, ct).Header().Get("Location"), "/t/task/import?file=")
	wantStatus(t, postForm(t, h, "/t/task/import/"+file+"/run", url.Values{}), http.StatusSeeOther)
	if heard := got.wait(t, 1); !strings.Contains(heard[0], "t=Sow+the+beans") {
		t.Errorf("a task imported sets off what waits for a task added: %q", heard)
	}

	written(t, a.Store, "created", server.ReminderType, "", map[string]any{"title": "Tea", "at": when.Store(time.Now().Add(-time.Minute), false)})
	if rang := h.(*server.Server).Ring(time.Now()); len(rang) != 1 {
		t.Fatalf("the reminder did not ring: %v", rang)
	}
	if heard := got.wait(t, 1); !strings.Contains(heard[1], "t=Tea") {
		t.Errorf("a reminder ringing sets off what waits for it to ring: %q", heard)
	}
	got.quiet(t)
}
