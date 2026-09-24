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

// TestTextFieldErrorHasNoPrefix checks that a text-field error message does not
// get prefixed with "Error:" by the CSS. The .sw-field__error class must not use
// a ::before pseudo-element to prepend "Error: ". Acceptance item 1.
func TestTextFieldErrorHasNoPrefix(t *testing.T) {
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

	// Read the component CSS to check no ::before adds a prefix.
	c, ok := reg.Get("text-field")
	if !ok {
		t.Fatal("text-field component not registered")
	}
	data, err := c.ReadFile("style.css")
	if err != nil {
		t.Fatalf("read text-field style.css: %v", err)
	}
	css := string(data)
	if strings.Contains(css, `content: "Error: "`) {
		t.Errorf(".sw-field__error must not prepend \"Error:\" via ::before;\ncss:\n%s\nwant: no content prefix on error messages", css)
	}

	_ = reg
}
