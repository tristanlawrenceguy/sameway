package export

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A record as text names each field by its label, a choice by its label,
// a day in words and a ref by its title; its writing is apart; with no
// way to name a ref, the ref is left out rather than given as an id.
func TestTextSaysARecordAsAReaderReadsIt(t *testing.T) {
	t.Parallel()
	typ, err := schema.Parse([]byte("name: task\ntitle: title\nfields:\n  title: {type: string}\n  status: {type: enum, values: [to_do, in_progress]}\n  due: {type: datetime}\n  done: {type: bool}\n  project: {type: ref, to: project}\n  notes: {type: markdown}\n"))
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]any{"title": "Café", "status": "in_progress", "due": "2026-10-09T00:00:00Z", "done": false, "project": "p1", "notes": " Bring 🎉 \n"}
	facts, body := Text(typ, fields, func(schema.Field, string) string { return "Garden" }, false)
	var said []string
	for _, f := range facts {
		said = append(said, f.Name+": "+f.Value)
	}
	if got, want := strings.Join(said, "; "), "Status: In progress; Due: Friday 9 October 2026; Project: Garden"; got != want {
		t.Errorf("facts = %q, want %q", got, want)
	}
	if len(body) != 1 || body[0] != "Bring 🎉" {
		t.Errorf("body = %q", body)
	}
	facts, _ = Text(typ, fields, nil, false)
	for _, f := range facts {
		if f.Field.Name == "project" {
			t.Errorf("with no titles a ref is left out, got %v", f)
		}
	}
}
