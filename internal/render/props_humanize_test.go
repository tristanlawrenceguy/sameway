package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestValidationErrorsDoNotShowRawJSONSchemaKeywords checks that canvas validation
// error messages do not leak internal JSON Schema terminology such as "missing property",
// "additional properties ... not allowed", or "is the wrong type". This covers the
// acceptance items for human-readable error descriptions.
func TestValidationErrorsDoNotShowRawJSONSchemaKeywords(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		component string
		props     map[string]any
		noWant    []string // none of these should appear in the error
	}{
		{
			name:      "missing required label does not say missing property",
			component: "button",
			props:     map[string]any{},
			noWant:    []string{"missing property"},
		},
		{
			name:      "unknown prop does not say additional properties",
			component: "button",
			props:     map[string]any{"label": "x", "bogus": 1},
			noWant:    []string{"additional properties"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := reg.Render(tc.component, tc.props)
			if err == nil {
				t.Fatalf("expected validation error for %s with props %+v", tc.component, tc.props)
			}
			errStr := err.Error()
			for _, no := range tc.noWant {
				if strings.Contains(errStr, no) {
					t.Errorf("error should not contain %q: %q", no, errStr)
				}
			}
		})
	}
}

// TestValidationErrorsShowHumanDescriptions checks that validation errors use
// plain-language descriptions instead of schema keywords. This covers the
// acceptance item for human-readable error messages on canvas alert blocks.
func TestValidationErrorsShowHumanDescriptions(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	_, err := reg.Render("button", map[string]any{})
	if err == nil {
		t.Fatal("expected validation error for button without label")
	}
	errStr := err.Error()

	// Should NOT contain machine-language phrases
	for _, raw := range []string{"missing property", "additional properties", "is the wrong type"} {
		if strings.Contains(errStr, raw) {
			t.Errorf("error should not show %q: %q", raw, errStr)
		}
	}

	// Should still convey what is wrong — field name and that something is needed
	if !strings.Contains(errStr, "Label") && !strings.Contains(errStr, "label") {
		t.Errorf("error should mention the label field: %q", errStr)
	}
}
