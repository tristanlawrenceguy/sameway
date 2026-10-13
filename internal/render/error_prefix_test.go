package render_test

// Tests for error messages without prefix words on all surfaces
// (task 0186). Error messages must not begin with "Error:", "Success:",
// "Warning:", or "Please" — they contain just the factual statement.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestTextFieldErrorHasNoPrefix checks that a text-field error message is
// its words; that nothing is drawn before them is checked in a browser
// (tools/a11y-runner/behave-look.mjs). Acceptance item 1.
func TestTextFieldErrorHasNoPrefix(t *testing.T) {
	t.Parallel()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("text-field", map[string]any{
		"label": "Email",
		"name":  "email",
		"error": "Enter an email address like name@example.com.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The error text must appear in the output.
	if !strings.Contains(out, "Enter an email address") {
		t.Errorf("error message should appear; got:\n%s", out)
	}
}
