package render_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAlertIconSpanNoAriaHiddenRegression verifies that the icon span in the
// alert component does not carry aria-hidden="true".  This is a regression
// guard for backlog items 0019 and 0010: unicode icons (⚠, ℹ, etc.) were
// previously hidden from assistive technology because the template included
// aria-hidden="true" on the <span class="sw-alert__icon"> element.
//
// Run: go test ./internal/render/ -run TestAlertIconSpanNoAriaHiddenRegression -v -count=1
func TestAlertIconSpanNoAriaHiddenRegression(t *testing.T) {
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

	// The icon span must be present.
	iconOpen := `<span class="sw-alert__icon"`
	if !containsStr(out, iconOpen) {
		t.Fatalf("alert output missing the icon span;\nwant substring %q\n\ngot:\n%s", iconOpen, out)
	}

	// The icon span must NOT have aria-hidden="true".  If it does, unicode
	// characters inside it are invisible to screen readers — a WCAG 1.4.1
	// violation (WCAG 2.2 AAA).
	if containsStr(out, `aria-hidden="true"`) {
		t.Errorf("icon span carries aria-hidden=\"true\" which hides the unicode icon from assistive technology;\nthis was fixed for backlog items 0019 and 0010.\ngot:\n%s", out)
	}

	// The unicode character must appear as text content in the DOM.
	if !containsStr(out, "\u26a0") {
		t.Errorf("the unicode icon ⚠ (\\u26a0) is missing from rendered output;\ngot:\n%s", out)
	}
}

// TestAlertInfoIconNoAriaHiddenRegression verifies the same regression for
// kind=info with icon ℹ, which uses a different golden file.  Acceptance item
// covers both danger and info kinds (backlog items 0019, 0010).
func TestAlertInfoIconNoAriaHiddenRegression(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "info",
		"icon":    "\u2139", // ℹ
		"title":   "Note",
		"message": "Informational message.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	if !containsStr(out, `<span class="sw-alert__icon"`) {
		t.Fatalf("alert output missing the icon span;\ngot:\n%s", out)
	}

	if containsStr(out, `aria-hidden="true"`) {
		t.Errorf("info-kind alert icon span has aria-hidden=\"true\";\ngot:\n%s", out)
	}

	if !containsStr(out, "\u2139") {
		t.Errorf("the unicode icon ℹ (\\u2139) is missing from rendered output;\ngot:\n%s", out)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
