package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAlertREADMEContainsWCAG141IconProp verifies the alert README documents
// that the icon prop serves as a WCAG 1.4.1 non-colour mechanism. Acceptance
// item 1 — README must contain the specific WCAG 1.4.1 paragraph about icons.
func TestAlertREADMEContainsWCAG141IconProp(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	alert, ok := reg.Get("alert")
	if !ok {
		t.Fatal("no alert component registered")
	}

	readme, err := alert.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	content := string(readme)

	// Must mention the icon prop as a WCAG 1.4.1 mechanism.
	if !strings.Contains(content, "WCAG 1.4.1") {
		t.Errorf("README missing 'WCAG 1.4.1' reference;\ngot:\n%s", content)
	}

	// Must mention non-colour distinction.
	if !strings.Contains(content, "non-colour") {
		t.Errorf("README must explain icon prop provides non-colour distinction;\ngot:\n%s", content)
	}

	// Must describe the icon prop as part of the non-colour mechanism.
	if !strings.Contains(content, "icon") && !strings.Contains(content, "Icon") {
		t.Error("README must mention that colour and icon convey meaning independently of CSS styling")
	}
}

// TestAlertREADMEIconPropExamples checks the README mentions concrete icon
// examples (⚠ or ℹ) as shown to users. Acceptance item 1 — usage guidance
// should include example unicode characters.
func TestAlertREADMEIconPropExamples(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	alert, ok := reg.Get("alert")
	if !ok {
		t.Fatal("no alert component registered")
	}

	readme, err := alert.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	content := string(readme)

	// Must show an icon example — either ⚠ or ℹ.
	hasWarningIcon := strings.Contains(content, "\u26a0")
	hasInfoIcon := strings.Contains(content, "\u2139")
	if !hasWarningIcon && !hasInfoIcon {
		t.Errorf("README must include a unicode icon example (⚠ or ℹ);\ngot:\n%s", content)
	}

	// Must mention a kind label example to show how it used to appear.
	if !strings.Contains(content, "⚠") {
		t.Error("README should show '⚠' as an icon example for danger alerts")
	}
}

// TestAlertManifestWCAGNotesMentionIconProp verifies the manifest's a11y.wcag.notes
// mentions that an icon prop provides an additional non-colour mechanism.
// Acceptance item 2 — manifest WCAG notes must document the icon as a fallback.
func TestAlertManifestWCAGNotesMentionIconProp(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	alert, ok := reg.Get("alert")
	if !ok {
		t.Fatal("no alert component registered")
	}

	var a11y struct {
		WCAG struct {
			Target string `json:"target"`
			Notes  string `json:"notes"`
		} `json:"wcag"`
	}
	if err := json.Unmarshal(alert.Manifest.A11y, &a11y); err != nil {
		t.Fatalf("parse a11y: %v", err)
	}

	notes := a11y.WCAG.Notes

	// Must mention colour as the primary kind indicator.
	if !strings.Contains(notes, "colour") && !strings.Contains(notes, "Color") {
		t.Error("WCAG notes must say kind is conveyed by colour and optionally an icon prop")
	}

	// Must reference 1.4.1.
	if !strings.Contains(notes, "1.4.1") {
		t.Error("WCAG notes must reference WCAG 1.4.1")
	}

	// Must mention role=alert for warning/danger (existing note).
	if !strings.Contains(notes, "role=alert") && !strings.Contains(notes, "role = alert") {
		t.Error("WCAG notes must still mention that warning and danger use role=alert")
	}

	// Target must be AAA.
	if a11y.WCAG.Target != "AAA" {
		t.Errorf("a11y.wcag.target is %q; want \"AAA\"", a11y.WCAG.Target)
	}
}
