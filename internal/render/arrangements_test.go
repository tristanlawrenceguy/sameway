package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An arrangement is a page of thought written once: every built-in one
// loads, says when it serves a person, and uses only real components with
// valid props; a workspace can add its own; a wrong one is refused when
// it is loaded, not when it is asked for.
func TestArrangementsAreCheckedWhenLoaded(t *testing.T) {
	reg := builtins(t)
	arrangements := reg.Arrangements()
	if len(arrangements) < 4 {
		t.Fatalf("expected the built-in arrangements, got %d", len(arrangements))
	}
	for _, a := range arrangements {
		if a.Use == nil || len(a.Use.When) < 20 || len(a.Use.Not) < 10 || len(a.Blocks) < 2 {
			t.Errorf("%s: an arrangement needs use.when, use.not and blocks, got %+v", a.Name, a)
		}
		for _, b := range a.Blocks {
			if _, ok := reg.Get(b.Component); !ok {
				t.Errorf("%s: block %s uses unknown component %s", a.Name, b.Key, b.Component)
			}
		}
	}
	if _, ok := reg.Arrangement("week"); !ok {
		t.Error("the week arrangement should be there")
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mine.json"), []byte(`{"name":"mine","description":"Mine.","use":{"when":"the person wants what I want, every time.","not":"never, really."},"blocks":[{"key":"a","component":"heading","props":{"text":"Hello"}}]}`), 0o644)
	if err := reg.LoadArrangementsDir(dir, "workspace"); err != nil {
		t.Fatal(err)
	}
	if a, ok := reg.Arrangement("mine"); !ok || a.Source != "workspace" {
		t.Error("a workspace arrangement should load with its source")
	}

	bad := t.TempDir()
	os.WriteFile(filepath.Join(bad, "broken.json"), []byte(`{"name":"broken","description":"x","use":{"when":"whenever a person needs a broken thing."},"blocks":[{"key":"a","component":"heading","props":{}}]}`), 0o644)
	if err := reg.LoadArrangementsDir(bad, "workspace"); err == nil || !strings.Contains(err.Error(), "block a") {
		t.Errorf("an arrangement whose block has invalid props should be refused with the block named, got %v", err)
	}
}
