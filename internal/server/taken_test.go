package server_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// photoTakenAt is a JPEG's start as a phone writes it, saying when it
// was taken, with no offset.
func photoTakenAt(at time.Time) string {
	le := binary.LittleEndian
	stamp := at.Format("2006:01:02 15:04:05") + "\x00"
	var b bytes.Buffer
	b.WriteString("\xff\xd8\xff\xe1\x00\x60Exif\x00\x00II*\x00")
	binary.Write(&b, le, uint32(8))
	binary.Write(&b, le, uint16(1))
	binary.Write(&b, le, []uint16{0x8769, 4})
	binary.Write(&b, le, []uint32{1, 26})
	binary.Write(&b, le, uint32(0))
	binary.Write(&b, le, uint16(1))
	binary.Write(&b, le, []uint16{0x9003, 2})
	binary.Write(&b, le, []uint32{uint32(len(stamp)), 44})
	binary.Write(&b, le, uint32(0))
	b.WriteString(stamp)
	return b.String()
}

// A photo shared today from last summer is of last summer: when it was
// taken is read from it as it is kept, and photos are sorted and
// narrowed by that, looking back (Taken, newest first; Taken in the last
// 30 days), not ahead as a day something is due. Photos kept before are
// read for it once.
func TestPhotosAreSortedAndNarrowedByWhenTheyWereTaken(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := a.Store.Now()
	ids := map[string]string{}
	for name, at := range map[string]time.Time{"Beach.jpg": now.AddDate(0, 0, -3), "Wedding.jpg": time.Date(2024, 5, 4, 14, 0, 0, 0, now.Location())} {
		body, ct := multipartFile(t, name, photoTakenAt(at), nil)
		ids[name] = strings.TrimPrefix(do(t, h, "POST", "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	}
	wedding, _ := a.Store.Get(records.FileType, ids["Wedding.jpg"])
	if got, _ := wedding.Fields["taken"].(string); !strings.HasPrefix(got, "2024-05-04") {
		t.Fatalf("when it was taken is kept on the file: %v", wedding.Fields)
	}
	recent := get(t, h, "/t/file?where=taken%3E%3D-30d").Body.String()
	row := func(id string) string { return `class="sw-row__link" href="/t/file/` + id + `"` }
	if !strings.Contains(recent, row(ids["Beach.jpg"])) || strings.Contains(recent, row(ids["Wedding.jpg"])) {
		t.Errorf("narrowed to the last 30 days, only the beach")
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "file", "as": "gallery", "order": "-taken", "label": "Photos", "controls": true},
	}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	if b, w := strings.Index(page, ">Beach<"), strings.Index(page, ">Wedding<"); b < 0 || w < b {
		t.Errorf("the newest taken first: %.3000s", page)
	}
	for _, said := range []string{"Taken, newest first", "Taken, oldest first", "Taken in the last 30 days", "Taken more than a year ago"} {
		if !strings.Contains(page, said) {
			t.Errorf("the choices look back: no %q", said)
		}
	}
	if strings.Contains(page, "Taken in the next 7 days") || strings.Contains(page, "Taken soonest first") {
		t.Errorf("not ahead, as a day something is due")
	}

	records.ApplyOps(a.Store, records.Op{Type: records.FileType, ID: ids["Wedding.jpg"], After: map[string]any{"taken": nil}})
	for _, j := range h.(*server.Server).BackgroundJobs() {
		if j.Name == "When photos already kept were taken, read once" {
			j.Run(context.Background(), now)
		}
	}
	if again, _ := a.Store.Get(records.FileType, ids["Wedding.jpg"]); again.Fields["taken"] == nil {
		t.Errorf("a photo kept before is read for when it was taken, once")
	}
}
