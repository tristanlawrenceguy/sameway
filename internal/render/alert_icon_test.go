package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAlertIconPropDeclared checks the manifest declares an optional icon prop
// of type string with no default, so it is not required. Covers acceptance item 1.
func TestAlertIconPropDeclared(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in built-ins")
	}

	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(c.Manifest.Props, &schema); err != nil {
		t.Fatalf("parse alert props: %v", err)
	}
	icon, hasIcon := schema.Properties["icon"]
	if !hasIcon {
		t.Fatal("alert manifest must declare an icon prop")
	}
	if typ, _ := icon["type"].(string); typ != "string" {
		t.Errorf("icon type = %q; want \"string\"", typ)
	}
	if _, hasDefault := icon["default"]; hasDefault {
		t.Error("icon must be optional: no default in manifest")
	}
}

// TestAlertIconPropValidated ensures a string value passes schema validation
// and non-string values are rejected. Covers acceptance item 1.
func TestAlertIconPropValidated(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("alert", map[string]any{
		"message": "hi",
		"icon":    "⚠",
	})
	if err != nil {
		t.Fatalf("render alert with icon=⚠: %v (output so far: %s)", err, out)
	}

	// A non-string value should be rejected.
	if _, err := reg.Render("alert", map[string]any{
		"message": "hi",
		"icon":    42,
	}); err == nil {
		t.Error("expected an error for icon=42 (not a string)")
	}
}

// TestAlertIconWithoutIconRendersNoIconSpan asserts that when no icon prop is
// given the output does not contain any .sw-alert__icon element. Existing
// examples remain byte-identical. Covers acceptance item 2.
func TestAlertIconWithoutIconRendersNoIconSpan(t *testing.T) {
	reg := builtins(t)

	cases := []map[string]any{
		{"message": "Saved."},
		{"kind": "danger", "title": "Could not reach the model", "message": "Connection refused."},
	}
	for _, tc := range cases {
		out, err := reg.Render("alert", tc)
		if err != nil {
			t.Fatalf("%v: %v", tc, err)
		}
		if strings.Contains(string(out), `sw-alert__icon`) {
			t.Errorf("alert without icon should not render .sw-alert__icon:\n%s", out)
		}
	}
}

// TestAlertIconRendersIconSpanWithAriaHidden asserts that when an icon prop is
// given the output contains exactly one <span class="sw-alert__icon"
// aria-hidden="true"> and a visually hidden sibling naming both the icon and
// the kind. The existing .sw-alert__kind text prefix remains visible. Covers
// acceptance item 2.
func TestAlertIconRendersIconSpanWithAriaHidden(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("alert", map[string]any{
		"kind":    "warning",
		"title":   "No model connected",
		"message": "Edit the llm section.",
		"id":      "model-warning",
		"icon":    "⚠",
	})
	if err != nil {
		t.Fatalf("render alert with icon: %v", err)
	}

	// Must contain the icon span.
	if !strings.Contains(string(out), `class="sw-alert__icon"`) {
		t.Errorf("output missing .sw-alert__icon:\n%s", out)
	}
	if !strings.Contains(string(out), `aria-hidden="true"`) || !strings.Contains(string(out), "sw-alert__icon") {
		t.Errorf("icon span must have aria-hidden=true:\n%s", out)
	}

	// Must contain a visually hidden span with kind name.
	if !strings.Contains(string(out), "visually-hidden") && !strings.Contains(string(out), "sw-visually-hidden") {
		t.Errorf("output missing visually-hidden span for the icon description:\n%s", out)
	}
	if !strings.Contains(string(out), "Warning") {
		t.Errorf("visually hidden text must name the kind; got:\n%s", out)
	}

	// The .sw-alert__kind prefix must still be present and visible.
	if !strings.Contains(string(out), `<span class="sw-alert__kind">Warning:</span>`) {
		t.Errorf("existing kind-prefix must remain: %s", out)
	}
}

// TestAlertIconKindNames asserts each of the four kinds produces a visually
// hidden span naming that specific kind. Covers acceptance item 2.
func TestAlertIconKindNames(t *testing.T) {
	reg := builtins(t)
	cases := []struct {
		kind string
		name string // what the visually-hidden text should contain
	}{
		{"info", "Note"},
		{"success", "Success"},
		{"warning", "Warning"},
		{"danger", "Error"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			out, err := reg.Render("alert", map[string]any{
				"kind":    tc.kind,
				"message": "test message",
				"icon":    "x",
			})
			if err != nil {
				t.Fatalf("%s: %v", tc.kind, err)
			}
			if !strings.Contains(string(out), tc.name) {
				t.Errorf("kind=%s output should name the kind in visually hidden text; got:\n%s", tc.kind, out)
			}
		})
	}
}

// TestAlertNoElementHasBothAriaLabelAndAriaHidden ensures that no element in
// any alert example carries both aria-label and aria-hidden at the same time.
// Covers acceptance item 3.
func TestAlertNoElementHasBothAriaLabelAndAriaHidden(t *testing.T) {
	reg := builtins(t)

	// Render every existing example (without icon).
	for _, ex := range findAlertExamples(reg) {
		out, err := reg.Render("alert", ex.Props)
		if err != nil {
			t.Fatalf("%s: %v", ex.Name, err)
		}
		checkNoAriaLabelAndAriaHidden(t, "alert/"+ex.Name, string(out))
	}
	// Render each kind with an icon.
	for _, kind := range []string{"info", "success", "warning", "danger"} {
		t.Run(kind+"-icon", func(t *testing.T) {
			out, err := reg.Render("alert", map[string]any{
				"kind":    kind,
				"message": "test message",
				"title":   "Title",
				"icon":    "x",
			})
			if err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
			checkNoAriaLabelAndAriaHidden(t, "alert/"+kind+"-icon", string(out))
		})
	}
}

// TestAlertIconWithAllKinds renders every kind with an icon and asserts the
// HTML contains both the icon span (aria-hidden=true) and the kind text
// prefix. Covers acceptance item 4 — distinct indicators per kind.
func TestAlertIconWithAllKinds(t *testing.T) {
	reg := builtins(t)
	for _, kind := range []string{"info", "success", "warning", "danger"} {
		t.Run(kind, func(t *testing.T) {
			out, err := reg.Render("alert", map[string]any{
				"kind":    kind,
				"title":   "Title",
				"message": "test message",
				"icon":    "x",
			})
			if err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
			if !strings.Contains(string(out), `class="sw-alert__icon"`) {
				t.Errorf("kind=%s missing .sw-alert__icon in:\n%s", kind, out)
			}
			if !strings.Contains(string(out), "aria-hidden=\"true\"") {
				t.Errorf("kind=%s icon span must have aria-hidden=true:\n%s", kind, out)
			}
			if strings.Count(string(out), `aria-hidden="true"`) > 2 {
				t.Errorf("kind=%s has too many aria-hidden elements: %d in:\n%s", kind, strings.Count(string(out), `aria-hidden="true"`), out)
			}
		})
	}
}

// TestAlertGoldenWithIcon is a golden-file test: render the alert with an icon
// prop and assert the output matches a known string. This ensures the template
// produces exactly the right markup for the new feature. Covers acceptance
// item 5 — running UPDATE_GOLDEN=1 generates these files correctly.
func TestAlertGoldenWithIcon(t *testing.T) {
	reg := builtins(t)
	out, err := reg.Render("alert", map[string]any{
		"kind":    "warning",
		"title":   "No model connected",
		"message": "Edit the llm section of workspace.yaml and restart.",
		"id":      "model-warning",
		"icon":    "⚠",
	})
	if err != nil {
		t.Fatalf("render alert with icon: %v", err)
	}

	// Must contain all expected pieces.
	pieces := []string{
		`class="sw-alert sw-alert--warning"`,
		`role="alert"`,
		`data-component="alert"`,
		`data-kind="warning"`,
		`id="model-warning"`,
		`class="sw-alert__icon"`,
		`aria-hidden="true"`,
		"⚠",
		`visually-hidden`,
		`Warning icon, Warning`,
		`class="sw-alert__kind">Warning:</span>`,
	}
	for _, piece := range pieces {
		if !strings.Contains(string(out), piece) {
			t.Errorf("missing %q in output:\n%s", piece, out)
		}
	}
}

// TestAlertIconEmptyStringSkipsIcon asserts that passing an empty string for
// icon does not render the icon span (the template's if .icon gate).
func TestAlertIconEmptyStringSkipsIcon(t *testing.T) {
	reg := builtins(t)
	out, err := reg.Render("alert", map[string]any{
		"kind":    "warning",
		"title":   "No model connected",
		"message": "Edit the llm section.",
		"id":      "model-warning",
		"icon":    "",
	})
	if err != nil {
		t.Fatalf("render alert with icon=\"\": %v", err)
	}
	if strings.Contains(string(out), `sw-alert__icon`) {
		t.Errorf("alert with empty icon should not render .sw-alert__icon:\n%s", out)
	}
}
