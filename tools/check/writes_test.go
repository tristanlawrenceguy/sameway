package main

import (
	"reflect"
	"testing"
)

const storeImportLine = "import \"github.com/tristanlawrenceguy/sameway/internal/store\"\n"

// Every way of holding the store is seen, in every package but store and
// records, and a test file is left alone.
func TestStoreWritesAreFoundEverywhere(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/chat/a.go", "package chat\n"+storeImportLine+
		"type Service struct{ Store *store.Store }\n"+
		"func (s *Service) made() { s.Store.Create(\"note\", nil) }\n"+
		"func (s *Service) read() { s.Store.Get(\"note\", \"1\") }\n")
	write(t, root, "internal/ingest/b.go", "package ingest\n"+storeImportLine+
		"type linker struct{ st *store.Store }\n"+
		"func (l *linker) find() { l.st.Update(\"person\", \"1\", nil) }\n"+
		"func Sync(db *store.Store) { db.Delete(\"event\", \"1\") }\n"+
		"func Later(x *store.Store) { st := x; _ = st }\n")
	write(t, root, "internal/server/media/c.go", "package media\n"+
		"var Op = struct{ Run func(a *App) }{Run: func(a *App) { a.Store.Put(\"file\", \"1\", nil) }}\n"+
		"func (s *Service) Update(x int) {}\n"+
		"func (s *Service) self() { s.Update(1) }\n")
	write(t, root, "internal/server/media/c_test.go", "package media\nfunc seed(a *App) { a.Store.Create(\"file\", nil) }\n")
	write(t, root, "internal/records/d.go", "package records\nfunc w(b *Book) { b.Store.Create(\"note\", nil) }\n")
	write(t, root, "internal/store/e.go", "package store\nfunc (s *Store) Create() { s.Store.Create() }\n")
	got := storeWritesIn(root)
	want := map[string]bool{
		"internal/chat/a.go Service.made":  true,
		"internal/ingest/b.go linker.find": true,
		"internal/ingest/b.go Sync":        true,
		"internal/server/media/c.go Op":    true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("found %v, want %v", got, want)
	}
}

// A write not on the list fails, and so does an entry that no longer writes.
func TestStoreWritesListed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/chat/a.go", "package chat\nfunc (s *Service) made() { s.Store.Create(\"note\", nil) }\n")
	r := checkStoreWrites(root, map[string]string{"internal/chat/b.go Service.gone": "why"})
	if !has(r.problems, "internal/chat/a.go Service.made writes the store itself") || !has(r.problems, "internal/chat/b.go Service.gone no longer writes") || len(r.problems) != 2 {
		t.Errorf("problems = %q", r.problems)
	}
	if r := checkStoreWrites(root, map[string]string{"internal/chat/a.go Service.made": "why"}); len(r.problems) != 0 {
		t.Errorf("listed with its reason, problems = %q", r.problems)
	}
}

// The list only goes down from base: an entry added fails.
func TestStoreWritesMayOnlyGoDown(t *testing.T) {
	t.Parallel()
	was, err := parseDebt([]byte("package main\n\nvar storeWrites = map[string]string{\n\t\"internal/chat/a.go Service.made\": \"why\",\n}\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := compareDebt(was, map[string]map[string]int{"storeWrites": {"internal/chat/a.go Service.made": 1, "internal/chat/b.go Service.more": 1}})
	if !has(r.problems, "internal/chat/b.go Service.more: added to storeWrites") || len(r.problems) != 1 {
		t.Errorf("problems = %q", r.problems)
	}
	if r := compareDebt(was, map[string]map[string]int{"storeWrites": {}}); len(r.problems) != 0 {
		t.Errorf("shrunk to nothing, problems = %q", r.problems)
	}
	if _, ok := mustParse(t, baseDebt)["storeWrites"]; ok {
		t.Error("a debt.go without storeWrites reads as having it, so the first change to add the list would fail as adding every entry")
	}
}

func mustParse(t *testing.T, src string) map[string]map[string]int {
	t.Helper()
	lists, err := parseDebt([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return lists
}
