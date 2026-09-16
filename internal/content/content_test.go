package content_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/content"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func starter(t *testing.T) (*schema.Set, *store.Store) {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return types, st
}

// A record and its file say the same thing: the body field is the Markdown
// body, everything else is front matter, and reading the file back gives
// the record again, times included.
func TestARecordRoundTripsThroughItsFile(t *testing.T) {
	types, _ := starter(t)
	note, _ := types.Get("note")
	created := time.Date(2026, 9, 16, 15, 4, 5, 0, time.UTC)
	rec := &store.Record{ID: "abc", Type: "note", CreatedAt: created, UpdatedAt: created.Add(time.Hour),
		Fields: map[string]any{"title": "Water the plants", "body": "Every Sunday.\n\nAnd Wednesdays.", "tags": []any{"garden"}}}
	data, err := content.Encode(note, rec)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") || !strings.Contains(text, "title: Water the plants\n") || !strings.HasSuffix(text, "---\nEvery Sunday.\n\nAnd Wednesdays.\n") {
		t.Fatalf("unexpected file:\n%s", text)
	}
	if strings.Contains(text, "body:") {
		t.Error("the body field belongs in the body, not the front matter")
	}
	fields, c, u, err := content.Decode(note, data)
	if err != nil {
		t.Fatal(err)
	}
	if fields["title"] != "Water the plants" || fields["body"] != "Every Sunday.\n\nAnd Wednesdays." || !c.Equal(created) || !u.Equal(created.Add(time.Hour)) {
		t.Errorf("decoded %v %v %v", fields, c, u)
	}

	// A block has no body field: its props are front matter, whole.
	block, _ := types.Get("block")
	brec := &store.Record{ID: "b1", Type: "block", CreatedAt: created, UpdatedAt: created,
		Fields: map[string]any{"component": "card", "props": map[string]any{"title": "Plan", "items": []any{"a", "b"}}, "position": int64(3), "span": int64(6)}}
	data, _ = content.Encode(block, brec)
	fields, _, _, err = content.Decode(block, data)
	if err != nil {
		t.Fatal(err)
	}
	props, _ := fields["props"].(map[string]any)
	if fields["component"] != "card" || props["title"] != "Plan" {
		t.Errorf("a block's props should survive the round trip, got %v", fields)
	}
}

// The folder follows the database: written as records change, exported
// whole on demand, and imported back with every change logged.
func TestTheFolderFollowsTheDatabaseAndBack(t *testing.T) {
	types, st := starter(t)
	m := content.Mirror{Dir: t.TempDir(), Types: types, Skip: []string{"message", "activity", "proposal"}}
	st.AfterWrite = m.Changed

	rec, err := st.Create("note", map[string]any{"title": "Water the plants", "body": "Every Sunday."})
	if err != nil {
		t.Fatal(err)
	}
	path := m.Path("note", rec.ID)
	if data, err := os.ReadFile(path); err != nil || !strings.Contains(string(data), "Every Sunday.") {
		t.Fatalf("creating a record should write its file: %v", err)
	}
	st.Update("note", rec.ID, map[string]any{"title": "Water the garden"})
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "title: Water the garden") {
		t.Error("updating a record should rewrite its file")
	}
	if _, err := st.Create("message", map[string]any{"role": "user", "content": "hi"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.Dir, "message")); err == nil {
		t.Error("history types are not content")
	}

	// A stray file goes on export; a missing one is written.
	os.WriteFile(m.Path("note", "stray"), []byte("---\ntitle: Old\n---\n"), 0o644)
	os.Remove(path)
	rep, err := m.Export(st)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Written != 1 || rep.Removed != 1 {
		t.Errorf("export should write 1 and remove 1, got %+v", rep)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("export should write the missing file")
	}

	// Import: a changed file updates, a new file creates, a gone file deletes,
	// and each is logged with what was there before.
	os.WriteFile(path, []byte("---\ntitle: Water everything\ncreated: 2026-09-16T15:04:05Z\nupdated: 2026-09-16T16:04:05Z\n---\nDaily.\n"), 0o644)
	os.WriteFile(m.Path("note", "new1"), []byte("---\ntitle: New from a pull\n---\nHello.\n"), 0o644)
	extra, _ := st.Create("note", map[string]any{"title": "Local only"})
	os.Remove(m.Path("note", extra.ID))
	var logged []string
	rep, err = m.Import(st, func(action string, r *store.Record, before map[string]any) {
		logged = append(logged, action+":"+r.ID+":"+strings.TrimSpace(strings.Join([]string{before2(before)}, "")))
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Created != 1 || rep.Updated != 1 || rep.Deleted != 1 || len(rep.Problems) != 0 {
		t.Errorf("import should create 1, update 1, delete 1, got %+v", rep)
	}
	got, _ := st.Get("note", rec.ID)
	if got.Fields["title"] != "Water everything" || got.Fields["body"] != "Daily." || !got.UpdatedAt.Equal(time.Date(2026, 9, 16, 16, 4, 5, 0, time.UTC)) {
		t.Errorf("import should take the file's fields and times, got %v %v", got.Fields, got.UpdatedAt)
	}
	if _, err := st.Get("note", "new1"); err != nil {
		t.Error("import should create the record a new file describes")
	}
	if _, err := st.Get("note", extra.ID); err == nil {
		t.Error("import should delete the record whose file is gone")
	}
	want := []string{"updated:" + rec.ID + ":Water the garden", "created:new1:", "deleted:" + extra.ID + ":Local only"}
	sort.Strings(logged)
	sort.Strings(want)
	if strings.Join(logged, "|") != strings.Join(want, "|") {
		t.Errorf("logged %v, want %v", logged, want)
	}

	// An import with nothing new changes nothing.
	rep, _ = m.Import(st, nil)
	if rep.Created+rep.Updated+rep.Deleted != 0 {
		t.Errorf("a second import should be quiet, got %+v", rep)
	}
}

func before2(before map[string]any) string {
	if before == nil {
		return ""
	}
	s, _ := before["title"].(string)
	return s
}
