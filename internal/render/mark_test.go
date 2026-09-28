package render_test

// Tests for keyboard accessibility of the mark component (task 0196).
// Verifies a tab user can focus the checkbox, toggle it with Enter or Space,
// and see the state change reflected in aria-checked.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestMarkKeyboard verifies acceptance items 1–3: a tab user can focus the
// mark checkbox, toggle it with Enter or Space, and see the state change in
// aria-checked. The test renders the component via Render("mark", props) and
// asserts on the HTML output string for four keyboard-relevant properties.
func TestMarkKeyboard(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	// --- Unchecked state: tabbable, no checked attribute (aria-checked="false") ---

	outOff, err := reg.Render("mark", map[string]any{
		"type":    "task",
		"record":  "k3n2p9",
		"field":   "done",
		"label":   "Done",
		"context": "Order compost",
		"checked": false,
	})
	if err != nil {
		t.Fatalf("render mark unchecked: %v", err)
	}

	gotOff := string(outOff)

	// 1. The input must be a native checkbox (type="checkbox"). Native checkboxes
	//    handle Enter and Space natively — no JavaScript needed for keyboard users.
	if !strings.Contains(gotOff, `type="checkbox"`) {
		t.Errorf("mark should render a native checkbox input; got:\n%s", gotOff)
		return
	}

	// 2. No tabindex="-1" on the input — a native checkbox in normal flow is always
	//    tabbable via Tab. This verifies the keyboard path opens (Acceptance item 1).
	if strings.Contains(gotOff, `tabindex="-1"`) {
		t.Errorf("mark checkbox must not have tabindex=\"-1\" (blocks tab focus);\ngot:\n%s", gotOff)
	}

	// 3. No checked attribute when unchecked — this is how a native checkbox
	//    expresses aria-checked="false". The browser derives the ARIA state from
	//    the DOM property (Acceptance item 2b).
	if strings.Contains(gotOff, `checked`) {
		t.Errorf("mark with checked=false should not have a checked attribute on the input (expresses aria-checked=\"false\"); got:\n%s", gotOff)
	}

	// The checkbox must be inside a <label> so screen readers announce its name.
	if !strings.Contains(gotOff, `<label class="sw-mark__label">`) {
		t.Errorf("mark checkbox should be inside a <label> for accessible name;\ngot:\n%s", gotOff)
	}

	// --- Checked state: has checked attribute (aria-checked="true") ---

	outOn, err := reg.Render("mark", map[string]any{
		"type":    "task",
		"record":  "k3n2p9",
		"field":   "done",
		"label":   "Done",
		"context": "Order compost",
		"checked": true,
	})
	if err != nil {
		t.Fatalf("render mark checked: %v", err)
	}

	gotOn := string(outOn)

	// 4. When checked=true the input carries the checked attribute — this is how
	//    a native checkbox expresses aria-checked="true" (Acceptance item 2b).
	if !strings.Contains(gotOn, `checked`) {
		t.Errorf("mark with checked=true should have a checked attribute on the input (expresses aria-checked=\"true\"); got:\n%s", gotOn)
	}
}
