package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestValidationErrorsUseReadableFieldNames checks that canvas validation
// error messages display human-readable field names with proper capitalization
// instead of raw lowercase prop keys or JSON pointer syntax. This covers the
// acceptance items for readable multi-field and single-field error messages.
func TestValidationErrorsUseReadableFieldNames(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		component      string
		props          map[string]any
		wantSubstrings []string // all must appear in the error
	}{
		{
			name:           "missing required label on button",
			component:      "button",
			props:          map[string]any{},
			wantSubstrings: []string{"Label"},
		},
		{
			name:           "unknown prop key shows readable name",
			component:      "button",
			props:          map[string]any{"label": "x", "bogus": 1},
			wantSubstrings: []string{"invalid props"},
		},
		{
			name:           "variant enum error shows capitalized prop name",
			component:      "button",
			props:          map[string]any{"label": "x", "variant": "huge"},
			wantSubstrings: []string{"Variant"},
		},
		{
			name:           "heading level out of range shows capitalized name",
			component:      "heading",
			props:          map[string]any{"text": "x", "level": 9},
			wantSubstrings: []string{"Level"},
		},
		{
			name:           "missing required message on alert",
			component:      "alert",
			props:          map[string]any{},
			wantSubstrings: []string{"Message"},
		},
		{
			name:           "missing items array on list",
			component:      "list",
			props:          map[string]any{},
			wantSubstrings: []string{"Items"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := reg.Render(tc.component, tc.props)
			if err == nil {
				t.Fatalf("expected validation error for %s with props %+v", tc.component, tc.props)
			}
			errStr := err.Error()
			for _, want := range tc.wantSubstrings {
				if !strings.Contains(errStr, want) {
					t.Errorf("error %q should contain readable field name %q", errStr, want)
				}
			}
		})
	}
}

// TestValidationErrorsDoNotShowRawJSONPointer checks that error messages do
// not expose raw JSON pointer paths like /label or /variant to the user.
func TestValidationErrorsDoNotShowRawJSONPointer(t *testing.T) {
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
			name:      "button missing label does not show /label path",
			component: "button",
			props:     map[string]any{},
			noWant:    []string{"/label"},
		},
		{
			name:      "button bad variant does not show /variant path",
			component: "button",
			props:     map[string]any{"label": "x", "variant": "huge"},
			noWant:    []string{"/variant"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := reg.Render(tc.component, tc.props)
			if err == nil {
				t.Fatalf("expected validation error for %s", tc.component)
			}
			errStr := err.Error()
			for _, no := range tc.noWant {
				if strings.Contains(errStr, no) {
					t.Errorf("error should not show raw JSON pointer path %q: %q", no, errStr)
				}
			}
		})
	}
}

// TestValidationErrorsMultiField shows all readable names when multiple props fail.
func TestValidationErrorsMultiField(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	// Button missing required label AND has unknown prop: both errors should
	// use readable names. The root-level "/" error for the missing 'label'
	// should resolve to a human-readable field name rather than showing "/".
	_, err := reg.Render("button", map[string]any{
		"extra_words": "sr text",
	})
	if err == nil {
		t.Fatal("expected validation error for button with missing label and extra prop")
	}
	errStr := err.Error()

	// Should contain "Label" (capitalized, not "/")
	if !strings.Contains(errStr, "Label") {
		t.Errorf("multi-field error should show 'Label' readable name: %q", errStr)
	}

	// Should NOT have a bare "/" as the field identifier for label
	// The root path "/" appears in the JSON pointer but should not be used as field name
	if strings.Contains(errStr, "/:") {
		t.Errorf("multi-field error should not show raw '/' as field name: %q", errStr)
	}

	// Should contain "invalid props" prefix
	if !strings.Contains(errStr, "invalid props") {
		t.Errorf("error should start with 'invalid props': %q", errStr)
	}
}

// TestValidationErrorsSingleFieldNoRegression ensures single-field errors
// like "Title is required" continue to work correctly (no regression from
// the readable-field-name fix).
func TestValidationErrorsSingleFieldNoRegression(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	_, err := reg.Render("button", map[string]any{})
	if err == nil {
		t.Fatal("expected error for button without label")
	}
	errStr := err.Error()

	// Should contain the human-readable field name "Label" with capital L
	if !strings.Contains(errStr, "Label") {
		t.Errorf("single-field error should show 'Label': %q", errStr)
	}

	// Should NOT contain raw path like "/label" or just "/" as field name
	if strings.Contains(errStr, "/:") {
		t.Errorf("error should not use raw '/' prefix: %q", errStr)
	}
}
