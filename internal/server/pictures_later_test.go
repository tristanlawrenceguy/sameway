package server_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A photo of a letter shared while the model cannot see stays a note
// named IMG_2041, its picture unread. While the model still cannot see,
// trying again changes nothing; once one can, the picture is read, the
// note takes the name of what it is and is sorted again, so it is tagged
// to do, each change in Activity.
func TestAPictureNotReadWhenItCameIsReadLater(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	a, h := newAppWith(t, app.Options{Clock: clock})
	model := &readsLetter{blind: true}
	a.Chat.Provider = model
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files", "IMG_2041.png")
	fw.Write(pngOf(t, 40, 30))
	mw.Close()
	if rec := do(t, h, http.MethodPost, "/share", &buf, mw.FormDataContentType()); rec.Code != http.StatusSeeOther {
		t.Fatalf("shared: %d", rec.Code)
	}
	note := lastNote(t, a)
	mu.Lock()
	now = now.Add(time.Hour)
	mu.Unlock()
	job := a.Automate.PicturesJob()

	job.Run(context.Background(), clock())
	if got, _ := a.Store.Get("note", note.ID); got.Fields["title"] != "IMG_2041" {
		t.Errorf("while the model cannot see, nothing changes: %v", got.Fields["title"])
	}

	model.mu.Lock()
	model.blind = false
	model.mu.Unlock()
	job.Run(context.Background(), clock())
	got, _ := a.Store.Get("note", note.ID)
	if got.Fields["title"] != "Parking fine from the council" || !strings.Contains(strings.Join(records.StringList(got.Fields["tags"]), ","), "to do") {
		t.Errorf("once a model can see, it is read, named and tagged: %v %v", got.Fields["title"], got.Fields["tags"])
	}
	logged, _ := a.Store.List(records.ActivityType, store.ListOptions{})
	var renamed, tagged bool
	for _, l := range logged {
		if l.Fields["target_id"] == note.ID && l.Fields["via"] == "because its pictures could be read now" {
			renamed = renamed || l.Fields["action"] == "updated"
			tagged = tagged || l.Fields["action"] == "tagged"
		}
	}
	if !renamed || !tagged {
		t.Errorf("the renaming and the tagging are each in Activity: renamed %v, tagged %v", renamed, tagged)
	}
	job.Run(context.Background(), clock())
	if model.saw != 1 {
		t.Errorf("a picture read is not read again: saw %d", model.saw)
	}
}
