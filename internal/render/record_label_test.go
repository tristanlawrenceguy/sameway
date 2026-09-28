package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestRecordComponentPageLabelIsShort renders the record component with
// detail="page" and asserts its visible label text is ≤3 words. This pins
// Acceptance 1 (≤3 words) through the render surface, not just the server page.
func TestRecordComponentPageLabelIsShort(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Render("record", map[string]any{
		"type":     "note",
		"record":   "k3n2p9",
		"detail":   "page",
		"title":    "Call the dentist",
		"text":     "Ask about Thursday.",
		"textProp": "body",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := string(out)
	if strings.Contains(got, "Open this note on its page") {
		t.Error("the record component with detail=page should have a ≤3-word visible label; 'Open this note on its page' is 6 words — use a short label with context for screen readers (Acceptance 1)")
	}
	if !strings.Contains(got, "sw-link--button") {
		t.Error("the record component with detail=page should carry sw-link--button class (Acceptance 2)")
	}
}
