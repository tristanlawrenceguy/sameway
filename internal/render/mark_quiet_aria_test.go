package render_test

// Tests for quiet + ariaLabel on the mark component (task 0178).
// When both quiet=true and ariaLabel are set, the mark template must not
// render any adjacent text — visible or visually-hidden — because the
// aria-label already provides the full accessible name. Rendering anything
// after the input would cause a screen reader to announce the label twice.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAMarkQuietWithAriaLabelRendersNoAdjacentText checks that the mark
// component, when given both quiet=true and ariaLabel, renders nothing after
// the <input> — no visible label and no visually-hidden span. The aria-label
// on the input is the sole accessible name (Acceptance 1).
func TestAMarkQuietWithAriaLabelRendersNoAdjacentText(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Render("mark", map[string]any{
		"type":      "task",
		"record":    "abc123",
		"field":     "done",
		"label":     "Done",
		"context":   "Order compost",
		"checked":   false,
		"quiet":     true,
		"ariaLabel": "Mark done Order compost",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, `aria-label="Mark done Order compost"`) {
		t.Errorf("mark with ariaLabel should render it on the input; got:\n%s", got)
		return
	}

	// After the closing > of the <input>, nothing should follow — no visible
	// text and no span. The aria-label is the sole source of the accessible name.
	if strings.Contains(got, `<span class="sw-visually-hidden">`) {
		t.Errorf("mark with quiet+ariaLabel must not render a visually-hidden span after the input (duplicate label); got:\n%s", got)
	}

	// There should be no visible text between the closing > of the input and
	// the closing </label> that is outside any classed span. The aria-label
	// alone names the control.
	if !strings.Contains(got, `></label><input type="hidden"`) {
		t.Errorf("mark with quiet+ariaLabel should have no text between input and hidden field; got:\n%s", got)
	}
}

// TestAMarkQuietWithoutAriaLabelStillRendersHiddenSpan checks that the mark
// component, when given quiet=true but NO ariaLabel, still renders the
// visually-hidden span as before. This is the existing behaviour for the
// manifest example "quiet" and must not change (Acceptance 3).
func TestAMarkQuietWithoutAriaLabelStillRendersHiddenSpan(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Render("mark", map[string]any{
		"type":    "task",
		"record":  "abc123",
		"field":   "done",
		"label":   "Done",
		"context": "Order compost",
		"checked": false,
		"quiet":   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, `<span class="sw-visually-hidden">Done Order compost</span>`) {
		t.Errorf("mark with quiet but no ariaLabel should still render the hidden span; got:\n%s", got)
	}
}

// TestAMarkNonQuietWithAriaLabelRendersVisibleLabel checks that when quiet is
// false and ariaLabel is set, the mark renders both the visible label and the
// visually-hidden context — just like the canvas case where a record block
// shows its checkbox (Acceptance 3).
func TestAMarkNonQuietWithAriaLabelRendersVisibleLabel(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Render("mark", map[string]any{
		"type":      "task",
		"record":    "abc123",
		"field":     "done",
		"label":     "Done",
		"context":   "Order compost",
		"checked":   false,
		"ariaLabel": "Mark done Order compost",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, "> Done") {
		t.Errorf("mark without quiet should render the visible label; got:\n%s", got)
	}
	if strings.Contains(got, `<span class="sw-visually-hidden"> Order compost</span>`) {
		// The context is hidden — fine.
	} else {
		t.Errorf("mark with context should render a visually-hidden span for the context; got:\n%s", got)
	}
}
