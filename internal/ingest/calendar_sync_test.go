package ingest

import (
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A sync adds what is new, changes what changed and takes away what the
// calendar no longer has, of the events it brought; nothing else.
func TestACalendarIsKeptInStep(t *testing.T) {
	types, err := schema.LoadFS(examples.FS, filepath.ToSlash(filepath.Join(examples.StarterRoot, "schema")))
	if err != nil {
		t.Skip("starter schema: ", err)
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "data.db"), types)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ev, _ := types.Get("event")
	st.Create("event", map[string]any{"title": "Mine", "starts": "2026-10-14 10:00"})
	table := func(rows ...map[string]string) *Table {
		return &Table{Columns: []string{"title", "starts", "uid"}, Rows: rows}
	}
	got, _ := SyncCalendar(st, ev, table(map[string]string{"uid": "a", "title": "A", "starts": "2026-10-12 09:00"}, map[string]string{"uid": "b", "title": "B", "starts": "2026-10-13 09:00"}), nil)
	if got.Added != 2 || len(got.UIDs) != 2 {
		t.Fatalf("first sync adds: %+v", got)
	}
	got, _ = SyncCalendar(st, ev, table(map[string]string{"uid": "a", "title": "A moved", "starts": "2026-10-12 10:00"}), got.UIDs)
	if got.Changed != 1 || got.Removed != 1 || got.Added != 0 {
		t.Errorf("then changes one and takes away the one gone: %+v", got)
	}
	if n, _ := st.Count("event"); n != 2 {
		t.Errorf("leaving the person's own: %d events", n)
	}
	if got.String() != "1 changed, 1 removed" {
		t.Errorf("said: %q", got.String())
	}
}
