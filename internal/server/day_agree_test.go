package server_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A meeting lists its ends before its starts. Its day is when it starts
// (schema DayField), and every surface says so: the calendar, the export,
// what else is on its day, and its list's groups. Once the calendar,
// related things and the list took the first datetime, ends, and the
// export took starts, so one meeting sat on two days.
func TestEverySurfaceAgreesOnARecordsDay(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	known := filepath.Join(t.TempDir(), "known.json")
	os.WriteFile(known, []byte("[]"), 0o644)
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	meeting := "name: meeting\ntitle: title\nfields:\n  title:\n    type: string\n  ends:\n    type: datetime\n  starts:\n    type: datetime\n"
	if err := os.WriteFile(filepath.Join(dir, "schema", "meeting.yaml"), []byte(meeting), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(dir, app.Options{Machine: workspace.Machine{Known: known}})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h := server.New(a)
	typ, _ := a.Types.Get("meeting")
	if typ.DayField() != "starts" {
		t.Fatalf("a meeting's day is its starts, got %q", typ.DayField())
	}
	rec, err := a.Store.Create("meeting", map[string]any{"title": "Harvest planning", "starts": "2026-09-12T00:00:00Z", "ends": "2026-09-25T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}

	if got := relate.Day(typ, rec); got != "2026-09-12" {
		t.Errorf("related: its day is %q, want 2026-09-12", got)
	}

	var ics bytes.Buffer
	if err := export.Write(&ics, export.ICS, typ, []*store.Record{rec}, func(schema.Field, string) string { return "" }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ics.String(), "DTSTART;VALUE=DATE:20260912") {
		t.Errorf("export: DTSTART is its starts\n%s", ics.String())
	}

	id := addCalendar(t, h, map[string]any{"type": "meeting", "month": "2026-09", "detail": "page"})
	page := get(t, h, "/canvas/"+id).Body.String()
	if cell := dayCell(page, "2026-09-12"); !strings.Contains(cell, "Harvest planning") {
		t.Errorf("calendar: the meeting is on 12 September\n%s", cell)
	}
	if cell := dayCell(page, "2026-09-25"); strings.Contains(cell, "Harvest planning") {
		t.Errorf("calendar: the meeting is not on the day it ends\n%s", cell)
	}

	// Its list groups it by when it starts, tomorrow, not when it ends.
	now := time.Now()
	if _, err := a.Store.Create("meeting", map[string]any{"title": "Seed swap", "starts": now.AddDate(0, 0, 1).UTC().Format(time.RFC3339), "ends": now.AddDate(0, 2, 0).UTC().Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	list := get(t, h, "/t/meeting").Body.String()
	week, later := strings.Index(list, ">This week "), strings.Index(list, ">Later ")
	swap := strings.Index(list, "Seed swap")
	if week < 0 || swap < week || (later >= 0 && later < swap) {
		t.Errorf("list: Seed swap is grouped under This week (week %d, later %d, at %d)", week, later, swap)
	}
}

// dayCell is a calendar day's cell, from its data-date to its end.
func dayCell(page, day string) string {
	i := strings.Index(page, `data-date="`+day+`"`)
	if i < 0 {
		return ""
	}
	rest := page[i+1:]
	if j := strings.Index(rest, `</td>`); j >= 0 {
		rest = rest[:j]
	}
	return rest
}
