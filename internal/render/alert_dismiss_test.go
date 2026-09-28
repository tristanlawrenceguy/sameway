package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAlertDismissPropDeclared checks the alert manifest declares an optional
// "dismiss" boolean prop — type is boolean, not required. Acceptance item 2.
func TestAlertDismissPropDeclared(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("no alert component registered")
	}

	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required,omitempty"`
	}
	if err := json.Unmarshal(c.Manifest.Props, &schema); err != nil {
		t.Fatal(err)
	}

	dismissDef, hasDismiss := schema.Properties["dismiss"]
	if !hasDismiss {
		t.Fatalf("alert manifest missing 'dismiss' property")
	}
	if typ, _ := dismissDef["type"].(string); typ != "boolean" {
		t.Errorf("dismiss type is %q; want \"boolean\"", typ)
	}

	reqMap := map[string]bool{}
	for _, r := range schema.Required {
		reqMap[r] = true
	}
	if reqMap["dismiss"] {
		t.Error("'dismiss' should not be in required")
	}
}

// TestAlertDismissPropRendersCloseButton checks that when the alert is rendered
// with dismiss=true, a close button element appears in the output. Acceptance item 2.
func TestAlertDismissPropRendersCloseButton(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "success",
		"message": "Changes saved",
		"dismiss": true,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	if !strings.Contains(out, `data-dismiss`) {
		t.Errorf("alert with dismiss=true should contain data-dismiss attribute;\ngot:\n%s", out)
	}
	if !strings.Contains(out, "sw-alert__close") {
		t.Errorf("alert with dismiss=true should have class sw-alert__close on the button;\ngot:\n%s", out)
	}
	if !strings.Contains(out, `aria-label="Close message"`) {
		t.Errorf("alert close button should have aria-label=\"Close message\";\ngot:\n%s", out)
	}
}

// TestAlertDismissPropFalseRendersNoCloseButton checks that when dismiss is not
// set or false, no close button appears. Acceptance item 2.
func TestAlertDismissPropFalseRendersNoCloseButton(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "success",
		"message": "Changes saved",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	if strings.Contains(out, `data-dismiss`) {
		t.Errorf("alert with dismiss not set must NOT contain data-dismiss;\ngot:\n%s", out)
	}
	if strings.Contains(out, "sw-alert__close") {
		t.Errorf("alert without dismiss must NOT have sw-alert__close button;\ngot:\n%s", out)
	}
}
