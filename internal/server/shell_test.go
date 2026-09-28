package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestCanvasIsAnApplicationWithOrWithoutPanes: the home page keeps the app
// shell and the whole width whether or not a pane exists, so it does not
// change shape when one appears; its own title leaves the screen once it
// holds anything; and a pane is named by what is in it.
func TestCanvasIsAnApplicationWithOrWithoutPanes(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))
	if shell, _ := htmltest.Attr(doc.Elements("body")[0], "data-shell"); shell != "app" {
		t.Errorf("canvas without panes should still be the app shell, got %q", shell)
	}
	if class, _ := htmltest.Attr(doc.Elements("h1")[0], "class"); !strings.Contains(class, "sw-visually-hidden") {
		t.Errorf("a canvas with blocks keeps its title in the outline only, got class %q", class)
	}
	if n := len(doc.Elements("aside")); n != 0 {
		t.Errorf("no pane without pane blocks, got %d", n)
	}

	rec := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "calendar", "props": map[string]any{"month": "2026-09", "detail": "brief"}, "region": "left",
	})
	wantStatus(t, rec, http.StatusCreated)
	rec = get(t, h, "/")
	doc = parse(t, rec)
	if n := len(doc.WithAttr("aria-label", "Left pane")); n != 1 {
		t.Fatalf("expected one left pane, got %d", n)
	}
	page := rec.Body.String()
	if !strings.Contains(page, "September 2026") || strings.Contains(page, "History") {
		t.Errorf("a pane holding one calendar is named after it, not History")
	}
	if len(doc.WithAttr("aria-label", "Blocks in the left pane")) != 1 {
		t.Errorf("the pane's list says which pane it is")
	}
}
