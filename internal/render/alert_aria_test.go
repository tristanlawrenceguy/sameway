package render_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAlertIconSpanNoAriaHidden asserts that the icon span does NOT have
// aria-hidden="true", so screen readers can announce unicode icon characters.
// This is the core accessibility fix: WCAG 1.4.1 violation where icons were
// hidden from assistive tech. Acceptance item 1 and 2.
func TestAlertIconSpanNoAriaHidden(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "danger",
		"icon":    "\u26a0", // ⚠
		"title":   "No model connected",
		"message": "Edit the llm section of workspace.yaml and restart.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The icon span must exist but without aria-hidden="true".
	if !contains(out, `<span class="sw-alert__icon"`) {
		t.Fatalf("output missing <span class=\"sw-alert__icon\">;\ngot:\n%s", out)
	}

	// It must NOT contain aria-hidden on the icon span.
	if contains(out, `aria-hidden="true"`) {
		t.Errorf("icon span has aria-hidden=\"true\" which hides unicode icons from screen readers (WCAG 1.4.1);\ngot:\n%s", out)
	}

	// The icon text must be present as visible content in the DOM.
	if !contains(out, "\u26a0") {
		t.Errorf("icon span missing unicode character ⚠;\ngot:\n%s", out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
