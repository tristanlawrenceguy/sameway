package schema_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		"tags": 5, "meta": "{bad", "when": "yesterday", "extra": 1,
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
