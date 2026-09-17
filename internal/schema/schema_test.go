package schema_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

const everyType = `
name: sample
description: One field of every type.
fields:
  name: {type: string, required: true, maxLength: 5}
  body: {type: text}
  notes: {type: markdown}
  count: {type: int, default: 2}
  ratio: {type: float}
  on: {type: bool}
  kind: {type: enum, values: [a, b], default: a}
  tags: {type: list, of: string}
  meta: {type: json}
  when: {type: datetime}
`

func parse(t *testing.T, src string) *schema.Type {
	t.Helper()
	typ, err := schema.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return typ
}

func TestParsePreservesFieldOrderAndTitle(t *testing.T) {
	typ := parse(t, everyType)
	var names []string
	for _, f := range typ.Fields {
		names = append(names, f.Name)
	}
	if strings.Join(names, ",") != "name,body,notes,count,ratio,on,kind,tags,meta,when" {
		t.Errorf("field order lost: %v", names)
	}
	if typ.Title != "name" {
		t.Errorf("title should default to the first string field, got %q", typ.Title)
	}
}

func TestParseRejectsBadDefinitions(t *testing.T) {
	cases := map[string]string{
		"name: Bad Name\nfields:\n  a: {type: string}":      "lowercase",
		"name: ok\nfields: {}":                              "no fields",
		"name: ok\nfields:\n  id: {type: string}":           "reserved",
		"name: ok\nfields:\n  x: {type: blob}":              "unknown type",
		"name: ok\nfields:\n  x: {type: enum}":              "needs values",
		"name: ok\nfields:\n  created_at: {type: datetime}": "reserved",
	}
	for src, want := range cases {
		_, err := schema.Parse([]byte(src))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: want error containing %q, got %v", src, want, err)
		}
	}
}

// TestNormalizeAcceptsFormsCLIAndJSON feeds each field the shapes it gets
// from a browser form (strings), the CLI (strings), and the API (JSON types).
func TestNormalizeAcceptsFormsCLIAndJSON(t *testing.T) {
	typ := parse(t, everyType)
	fromStrings := map[string]any{
		"name": "abc", "count": "7", "ratio": "1.5", "on": "yes", "kind": "b",
		"tags": "x, y", "meta": `{"k":1}`, "when": "2026-09-10T12:00:00Z",
	}
	fromJSON := map[string]any{
		"name": "abc", "count": float64(7), "ratio": 1.5, "on": true, "kind": "b",
		"tags": []any{"x", "y"}, "meta": map[string]any{"k": float64(1)}, "when": "2026-09-10T12:00:00Z",
	}
	for label, in := range map[string]map[string]any{"strings": fromStrings, "json": fromJSON} {
		out, err := typ.Normalize(in)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if out["count"] != int64(7) || out["ratio"] != 1.5 || out["on"] != true || out["kind"] != "b" {
			t.Errorf("%s: scalars wrong: %#v", label, out)
		}
		if tags, _ := out["tags"].([]any); len(tags) != 2 || tags[1] != "y" {
			t.Errorf("%s: tags wrong: %#v", label, out["tags"])
		}
		if meta, _ := out["meta"].(map[string]any); meta["k"] != float64(1) {
			t.Errorf("%s: meta wrong: %#v", label, out["meta"])
		}
		if out["body"] != "" || out["notes"] != "" {
			t.Errorf("%s: absent text fields should be empty strings", label)
		}
	}
	defaults, err := typ.Normalize(map[string]any{"name": "x"})
	if err != nil || defaults["count"] != int64(2) || defaults["kind"] != "a" || defaults["on"] != false {
		t.Errorf("defaults: %v %#v", err, defaults)
	}
}

func TestNormalizeReportsEveryProblemAtOnce(t *testing.T) {
	typ := parse(t, everyType)
	_, err := typ.Normalize(map[string]any{
		"name": "too long", "count": "x", "ratio": "y", "on": "maybe", "kind": "z",
		"tags": 5, "meta": "{bad", "when": "someday", "extra": 1,
	})
	var ve *schema.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	for _, f := range []string{"name", "count", "ratio", "on", "kind", "tags", "meta", "when", "extra"} {
		if ve.Problems[f] == "" {
			t.Errorf("no problem reported for %s: %v", f, ve.Problems)
		}
	}
	if !strings.Contains(ve.Problems["kind"], "a, b") || !strings.Contains(ve.Problems["name"], "5 characters") {
		t.Errorf("problems should say how to fix: %v", ve.Problems)
	}
}

func TestJSONSchemaMirrorsFields(t *testing.T) {
	s := parse(t, everyType).JSONSchema()
	props := s["properties"].(map[string]any)
	if props["count"].(map[string]any)["type"] != "integer" || props["on"].(map[string]any)["type"] != "boolean" {
		t.Errorf("scalar types wrong: %v", props)
	}
	if enum, _ := props["kind"].(map[string]any)["enum"].([]string); len(enum) != 2 {
		t.Errorf("enum missing: %v", props["kind"])
	}
	if props["when"].(map[string]any)["format"] != "date-time" {
		t.Errorf("datetime should carry format: %v", props["when"])
	}
	if req, _ := s["required"].([]string); len(req) != 1 || req[0] != "name" {
		t.Errorf("required: %v", s["required"])
	}
	if s["additionalProperties"] != false {
		t.Errorf("schema must forbid unknown fields")
	}
}

func TestLoadDirectory(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("b.yaml", "name: b\nfields:\n  x: {type: string}")
	write("a.yaml", "name: a\nfields:\n  x: {type: string}")
	write("ignored.txt", "not yaml")
	set, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(set.Names(), ",") != "a,b" {
		t.Errorf("types should be sorted by name: %v", set.Names())
	}
	write("dup.yaml", "name: a\nfields:\n  x: {type: string}")
	if _, err := schema.Load(dir); err == nil || !strings.Contains(err.Error(), "defined twice") {
		t.Errorf("duplicate type names must be rejected: %v", err)
	}
	empty, err := schema.Load(dir + "/missing")
	if err != nil || len(empty.Types) != 0 {
		t.Errorf("missing dir should give an empty set: %v", err)
	}
}

// A workspace keeps the copy of an internal type it was created with. When
// a later release adds a field to that type, Complete gives the workspace's
// copy that field too, without touching what the workspace wrote itself.
func TestCompleteAddsBuiltinFieldsToInternalTypes(t *testing.T) {
	dir := t.TempDir()
	old := "name: block\ninternal: true\nfields:\n  component: {type: string, required: true, description: mine}\n  position: {type: int}\n"
	mine := "name: note\nfields:\n  title: {type: string}\n"
	os.WriteFile(filepath.Join(dir, "block.yaml"), []byte(old), 0o644)
	os.WriteFile(filepath.Join(dir, "note.yaml"), []byte(mine), 0o644)
	ws, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := schema.LoadFS(fstest.MapFS{
		"schema/block.yaml":  {Data: []byte("name: block\ninternal: true\nfields:\n  component: {type: string, required: true, description: theirs}\n  region: {type: enum, values: [main, left, right], default: main}\n")},
		"schema/canvas.yaml": {Data: []byte("name: canvas\ninternal: true\nfields:\n  name: {type: string, required: true}\n")},
		"schema/note.yaml":   {Data: []byte("name: note\nfields:\n  title: {type: string}\n  body: {type: text}\n")},
	}, "schema")
	if err != nil {
		t.Fatal(err)
	}
	ws.Complete(builtin)

	blk, _ := ws.Get("block")
	var names []string
	for _, f := range blk.Fields {
		names = append(names, f.Name)
	}
	if strings.Join(names, ",") != "component,position,region" {
		t.Errorf("missing fields should be appended after the workspace's own: %v", names)
	}
	if region, ok := blk.Field("region"); !ok || region.Default != "main" || len(region.Values) != 3 {
		t.Errorf("region should arrive with its definition, got %+v", region)
	}
	if f, _ := blk.Field("component"); f.Description != "mine" {
		t.Errorf("a field the workspace defines must stay as written, got %q", f.Description)
	}
	if note, _ := ws.Get("note"); len(note.Fields) != 1 {
		t.Errorf("a type the person owns must not be completed, got %d fields", len(note.Fields))
	}
	// A whole internal type the workspace predates arrives too, so the tabs
	// exist in a workspace made before there were tabs.
	if canvas, ok := ws.Get("canvas"); !ok || !canvas.Internal || len(canvas.Fields) != 1 {
		t.Errorf("a missing internal type should be added from the built-in set, got %+v", canvas)
	}
	if names := strings.Join(ws.Names(), ","); names != "block,canvas,note" {
		t.Errorf("types stay sorted after completion, got %s", names)
	}
}

func TestLoadFSMissingDirIsEmpty(t *testing.T) {
	set, err := schema.LoadFS(fstest.MapFS{}, "schema")
	if err != nil || len(set.Types) != 0 {
		t.Fatalf("missing dir should be an empty set, got %v %v", set.Types, err)
	}
}

// A ref field names the type it points at, and a workspace whose refs
// point at a type it does not have is told so before it starts.
func TestARefNamesWhatItPointsAt(t *testing.T) {
	if _, err := schema.Parse([]byte("name: task\nfields:\n  title: {type: string}\n  project: {type: ref}\n")); err == nil || !strings.Contains(err.Error(), "needs to") {
		t.Errorf("a ref without to is refused: %v", err)
	}
	typ := parse(t, "name: task\nfields:\n  title: {type: string}\n  project: {type: ref, to: project}\n")
	props := typ.JSONSchema()["properties"].(map[string]any)
	if desc, _ := props["project"].(map[string]any)["description"].(string); !strings.Contains(desc, "id of a project") {
		t.Errorf("the schema says what a ref holds: %v", props["project"])
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "task.yaml"), []byte("name: task\nfields:\n  title: {type: string}\n  project: {type: ref, to: project}\n"), 0o644)
	set, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := set.CheckRefs(); err == nil || !strings.Contains(err.Error(), `points at "project"`) {
		t.Errorf("a ref to a missing type is named: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "project.yaml"), []byte("name: project\nfields:\n  title: {type: string}\n"), 0o644)
	set, _ = schema.Load(dir)
	if err := set.CheckRefs(); err != nil {
		t.Errorf("with the type there, refs are fine: %v", err)
	}
}
