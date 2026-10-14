package media_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A file left "Being read" when Sameway closed is put right when it
// opens, so no page says it is being read days later: a recording waits
// to be written down again, any other file says it was not read and what
// to do. A file still being read in this Sameway is left as it is.
func TestAFileLeftBeingReadIsPutRightWhenSamewayOpens(t *testing.T) {
	t.Parallel()
	var now atomic.Int64
	now.Store(time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC).UnixNano())
	a, h := newAppWith(t, app.Options{Clock: func() time.Time { return time.Unix(0, now.Load()) }})
	doc, _ := a.Store.Create("file", map[string]any{"title": "Lease", "name": "lease.pages", "kind": "document", "status": "converting"})
	rec, _ := a.Store.Create("file", map[string]any{"title": "Call", "name": "call.m4a", "kind": "audio", "status": "converting", "note": "Being written down on this computer: 2 of 5 parts done."})

	now.Add(int64(8 * 24 * time.Hour)) // Sameway opens again eight days on
	jobs := h.(*server.Server).BackgroundJobs()
	fresh, _ := a.Store.Create("file", map[string]any{"title": "New", "name": "new.pages", "kind": "document", "status": "converting"})
	for _, j := range jobs {
		if j.Name == "Files left half read when Sameway closed" {
			j.Run(context.Background(), time.Unix(0, now.Load()))
		}
	}
	got := func(id string) (string, string) {
		r, _ := a.Store.Get("file", id)
		s, _ := r.Fields["status"].(string)
		n, _ := r.Fields["note"].(string)
		return s, n
	}
	if s, n := got(doc.ID); s != "failed" || !strings.Contains(n, "Add the file again") {
		t.Errorf("a document left being read should say it was not read and what to do: %s %q", s, n)
	}
	if page := get(t, h, "/t/file/"+doc.ID).Body.String(); strings.Contains(page, "Reading the file") || !strings.Contains(page, "Sameway closed before it was read") {
		t.Errorf("its page should no longer say it is being read")
	}
	if s, n := got(rec.ID); s != "ready" || n != "" {
		t.Errorf("a recording left being written down should wait to be written down again: %s %q", s, n)
	}
	if s, _ := got(fresh.ID); s != "converting" {
		t.Errorf("a file being read now is left as it is, got %s", s)
	}
}
