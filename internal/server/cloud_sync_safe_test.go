package server_test

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/cloudsync"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The time zone is the person's, so it is the same on each computer: set
// on one, the other's days follow it, and a change back to the computer's
// own goes across too. Settings that are the computer's own never do,
// whatever another copy sends.
func TestTheTimeZoneSetOnOneComputerIsTheOthers(t *testing.T) {
	t.Parallel()
	c := twoComputers(t)
	after(t, c.srvA, postForm(t, c.srvA, "/zone", url.Values{"zone": {"Asia/Tokyo"}}))
	c.round()
	if got := c.b.Workspace.Zone().String(); got != "Asia/Tokyo" {
		t.Fatalf("the laptop's days follow the zone set on the desktop: %q", got)
	}
	if got := c.b.Now().Location().String(); got != "Asia/Tokyo" {
		t.Errorf("the laptop's clock is said there: %q", got)
	}
	after(t, c.srvB, postForm(t, c.srvB, "/zone", url.Values{"zone": {"America/New_York"}}))
	c.round()
	if got := c.a.Workspace.Config.UI.Zone; got != "America/New_York" {
		t.Errorf("the latest choice, on either, is both's: %q", got)
	}
	after(t, c.srvA, postForm(t, c.srvA, "/zone", url.Values{"zone": {""}}))
	c.round()
	if got := c.b.Workspace.Config.UI.Zone; got != "" {
		t.Errorf("each computer's own zone again, on both: %q", got)
	}

	c.a.Store.StampSetting("notify.command", "del everything")
	c.round()
	if got := c.b.Workspace.Get("notify.command"); got != "" {
		t.Errorf("a setting that is the computer's own is never taken from another: %q", got)
	}
}

// When the workspace's folder goes from the cloud folder, nothing is
// passed and the page says so, plainly, with what to do; nothing made
// meanwhile is lost: put back from one computer, the other picks up on
// its own and passes on what it made, even what it wrote just before the
// folder went and the first had not read.
func TestAFolderThatGoesIsSaidAndNothingIsLost(t *testing.T) {
	t.Parallel()
	c := twoComputers(t)
	c.round()
	before, _ := c.b.Store.Create("note", map[string]any{"title": "Written just before"})
	c.srvB.SyncRound()
	if err := os.RemoveAll(c.folder); err != nil {
		t.Fatal(err)
	}
	if err := c.srvA.SyncRound(); err != cloudsync.ErrGone {
		t.Fatalf("the round says the folder is gone: %v", err)
	}
	if _, err := os.Stat(c.folder); err == nil {
		t.Fatal("a folder that went is not put back on its own")
	}
	meanwhile, _ := c.b.Store.Create("note", map[string]any{"title": "Written while it was gone"})
	c.srvB.SyncRound()
	page := get(t, c.srvB, "/workspaces").Body.String()
	for _, want := range []string{"is no longer in OneDrive: deleted there, or not back yet.", "What you change here is kept, and passes on once the folder is back.", "Put its folder back in OneDrive"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page says %q: %s", want, truncate(page))
		}
	}

	page = after(t, c.srvA, postForm(t, c.srvA, "/cloud-sync", url.Values{"cloud": {"OneDrive"}})).Body.String()
	if !strings.Contains(page, "Its folder is back in OneDrive") {
		t.Errorf("putting it back says so: %s", truncate(page))
	}
	if _, err := os.Stat(filepath.Join(c.folder, cloudsync.Start)); err != nil {
		t.Errorf("back under its own name, with a copy to start from: %v", err)
	}
	c.round()
	for _, n := range []string{before.ID, meanwhile.ID} {
		if _, err := c.a.Store.Get("note", n); err != nil {
			t.Errorf("the laptop's note reaches the desktop once the folder is back: %v", err)
		}
	}
	if page := get(t, c.srvB, "/workspaces").Body.String(); strings.Contains(page, "no longer in OneDrive") {
		t.Errorf("the laptop is in step again: %s", truncate(page))
	}
}

// A cloud app gone from this computer stops keeping in step, said on the
// page and as the job's error, never as silence.
func TestACloudAppGoneIsSaid(t *testing.T) {
	t.Parallel()
	c := twoComputers(t)
	c.b.Workspace.Machine.Cloud.Name = "Dropbox" // OneDrive is not on this computer now
	err := c.srvB.SyncRound()
	if err == nil || err.Error() != "OneDrive is not on this computer just now" {
		t.Fatalf("the round says why: %v", err)
	}
	page := get(t, c.srvB, "/workspaces").Body.String()
	if !strings.Contains(page, "OneDrive is not on this computer just now. What you change here is kept, and passes on once OneDrive is back.") || strings.Contains(page, "Keep it in step through") {
		t.Errorf("the page says so, and still counts it kept in step: %s", truncate(page))
	}
}

// Both computers change one note's words before either has read the
// other's: both agree on the later, and the earlier is kept as a clash on
// both, to be offered back, not lost.
func TestTheSameNoteChangedOnBothKeepsBoth(t *testing.T) {
	t.Parallel()
	c := twoComputers(t)
	n, _ := c.a.Store.Create("note", map[string]any{"title": "Plans", "body": "Saturday"})
	c.round()
	c.a.Store.Update("note", n.ID, map[string]any{"body": "Saturday at the beach"})
	c.b.Store.Update("note", n.ID, map[string]any{"body": "Saturday in the hills"})
	c.round()
	c.srvA.SyncRound()
	gotA, _ := c.a.Store.Get("note", n.ID)
	gotB, _ := c.b.Store.Get("note", n.ID)
	if gotA.Fields["body"] != gotB.Fields["body"] {
		t.Fatalf("both agree: %v / %v", gotA.Fields["body"], gotB.Fields["body"])
	}
	for name, st := range map[string]*store.Store{"desktop": c.a.Store, "laptop": c.b.Store} {
		clashes, _ := st.List(store.ClashType, store.ListOptions{})
		if len(clashes) != 1 || clashes[0].Fields["text"] == gotA.Fields["body"] {
			t.Errorf("the %s keeps the other version: %v", name, clashes)
		}
	}
}
