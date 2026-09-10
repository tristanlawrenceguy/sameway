package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const noteYAML = `
name: note
title: title
fields:
  title: {type: string, required: true, maxLength: 20}
  body: {type: text}
  tags: {type: list, of: string}
  status: {type: enum, values: [draft, published], default: draft}
  pinned: {type: bool, default: false}
  rank: {type: int, default: 0}
  extra: {type: json}
`

func open(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "note.yaml"), []byte(noteYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", set)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestRoundTrip(t *testing.T) {
	st := open(t)
	rec, err := st.Create("note", map[string]any{"title": "Hello", "tags": "a, b", "pinned": "yes", "rank": "3", "extra": map[string]any{"k": "v"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.Get("note", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fields["title"] != "Hello" || got.Fields["status"] != "draft" || got.Fields["pinned"] != true {
		t.Errorf("unexpected fields: %#v", got.Fields)
	}
	if tags, _ := got.Fields["tags"].([]any); len(tags) != 2 || tags[0] != "a" {
		t.Errorf("tags not round-tripped: %#v", got.Fields["tags"])
	}
	if got.Fields["rank"] != int64(3) {
		t.Errorf("rank: %#v", got.Fields["rank"])
	}
	if extra, _ := got.Fields["extra"].(map[string]any); extra["k"] != "v" {
		t.Errorf("extra: %#v", got.Fields["extra"])
	}
	upd, err := st.Update("note", rec.ID, map[string]any{"status": "published"})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Fields["status"] != "published" || upd.Fields["title"] != "Hello" {
		t.Errorf("update lost fields: %#v", upd.Fields)
	}
	list, err := st.List("note", store.ListOptions{})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if err := st.Delete("note", rec.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get("note", rec.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected not found, got %v", err)
	}
}

func TestValidation(t *testing.T) {
	st := open(t)
	_, err := st.Create("note", map[string]any{"body": "no title", "status": "bogus", "nope": 1})
	var ve *schema.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
	for _, f := range []string{"title", "status", "nope"} {
		if _, ok := ve.Problems[f]; !ok {
			t.Errorf("expected a problem for %s: %v", f, ve.Problems)
		}
	}
	if _, err := st.Create("nothing", nil); err == nil {
		t.Error("expected unknown type error")
	}
}
