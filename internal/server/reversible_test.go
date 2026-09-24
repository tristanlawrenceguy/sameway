package server_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Everything reversible. A delete through the API is logged, as the
// person's, with how it came and everything the record had, so it can be
// put back like any other.
func TestADeleteThroughTheAPICanBeUndone(t *testing.T) {
	a, h := newApp(t)
	note, _ := a.Store.Create("note", map[string]any{"title": "Water the plants", "body": "Every Sunday."})
	wantStatus(t, do(t, h, http.MethodDelete, "/api/note/"+note.ID, nil, ""), http.StatusOK)

	log := get(t, h, "/activity").Body.String()
	if !strings.Contains(log, "You deleted note Water the plants, through the API") {
		t.Fatalf("the delete is in the log, saying how it came: %s", truncate(log))
	}
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)"`).FindStringSubmatch(log)
	if undo == nil {
		t.Fatal("the delete offers Undo")
	}
	postForm(t, h, undo[1], url.Values{"from": {"/activity"}})
	if got, err := a.Store.Get("note", note.ID); err != nil || got.Fields["body"] != "Every Sunday." {
		t.Errorf("undo brings the note back whole, got %v %v", got, err)
	}
}

// The activity log is what makes every change reversible: it is read by
// anyone and changed by nobody.
func TestTheActivityLogCannotBeChangedThroughTheAPI(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("note", map[string]any{"title": "Seed"})
	var log struct{ Records []struct{ ID string } }
	decode(t, get(t, h, "/api/activity"), &log)
	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Logged"})
	decode(t, get(t, h, "/api/activity"), &log)
	if len(log.Records) == 0 {
		t.Fatal("a change through the API is logged")
	}
	id := log.Records[0].ID
	for _, rec := range []int{
		do(t, h, http.MethodDelete, "/api/activity/"+id, nil, "").Code,
		postJSON(t, h, http.MethodPatch, "/api/activity/"+id, map[string]any{"summary": "nothing happened"}).Code,
		postJSON(t, h, http.MethodPost, "/api/activity", map[string]any{"actor": "human", "action": "said"}).Code,
	} {
		if rec != http.StatusMethodNotAllowed {
			t.Errorf("the log refuses to be changed, got %d", rec)
		}
	}
	if _, err := a.Store.Get("activity", id); err != nil {
		t.Error("the entry is still there")
	}
}

// A deleted workspace goes to Sameway's trash with everything in its
// folder, and comes back from the Workspaces page.
func TestADeletedWorkspaceGoesToTheTrashAndComesBack(t *testing.T) {
	// The test app keeps its own list of workspaces, and so its own trash.
	_, h := newApp(t)
	dir := filepath.Join(t.TempDir(), "garden")
	os.MkdirAll(filepath.Join(dir, "content"), 0o755)
	os.WriteFile(filepath.Join(dir, "my-own-notes.txt"), []byte("kept"), 0o644)

	gone, err := workspace.Trash(dir, "Garden")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("the folder has left where it was")
	}
	list := workspace.TrashedWorkspaces()
	if len(list) != 1 || list[0].Name != "Garden" {
		t.Fatalf("the trash lists it, got %v", list)
	}

	page := get(t, h, "/workspaces").Body.String()
	if !strings.Contains(page, "Recently deleted") || !strings.Contains(page, "Garden") {
		t.Errorf("the Workspaces page offers it back: %s", truncate(page))
	}
	r := postForm(t, h, "/workspaces/restore", url.Values{"now": {gone.Now}})
	if body := after(t, h, r).Body.String(); !strings.Contains(body, "Garden is back at") {
		t.Errorf("restoring says so: %s", truncate(body))
	}
	if kept, err := os.ReadFile(filepath.Join(dir, "my-own-notes.txt")); err != nil || string(kept) != "kept" {
		t.Error("everything that was in the folder comes back with it")
	}
}

// Logging a habit can be taken back: the entry it made goes.
func TestLoggingAHabitCanBeUndone(t *testing.T) {
	a, h := newApp(t)
	habit, err := a.Store.Create("habit", map[string]any{"name": "Water", "cadence": "day", "target": 1})
	if err != nil {
		t.Fatal(err)
	}
	postForm(t, h, "/habit/"+habit.ID+"/log", url.Values{"from": {"/t/habit/" + habit.ID}})
	if n, _ := a.Store.Count("entry"); n != 1 {
		t.Fatalf("logging makes an entry, got %d", n)
	}
	log := get(t, h, "/activity").Body.String()
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)"`).FindStringSubmatch(log)
	if undo == nil {
		t.Fatalf("the log offers Undo: %s", truncate(log))
	}
	postForm(t, h, undo[1], url.Values{"from": {"/activity"}})
	if n, _ := a.Store.Count("entry"); n != 0 {
		t.Errorf("undo takes the logged amount back, %d left", n)
	}
}
