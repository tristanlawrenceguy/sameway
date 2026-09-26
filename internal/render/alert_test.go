package render_test

// The alert: its kind said four ways, its title a heading, its close button
// a whole target, and a live role only where one is announced.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

func renderAlert(t *testing.T, props map[string]any) string {
	t.Helper()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	out, err := reg.Render("alert", props)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return string(out)
}

// TestAnAlertSaysItsKindInWords: a screen reader hears the kind as a word
// first; the mark, a shape of its own per kind, is silent; nothing is added
// to what is seen.
func TestAnAlertSaysItsKindInWords(t *testing.T) {
	for kind, want := range map[string][2]string{
		"info": {"Information: ", "ℹ"}, "success": {"Success: ", "✓"}, "warning": {"Warning: ", "⚠"}, "danger": {"Error: ", "!"},
	} {
		out := renderAlert(t, map[string]any{"kind": kind, "message": "Something happened."})
		if !strings.Contains(out, `<span class="sw-visually-hidden">`+want[0]+`</span>`) {
			t.Errorf("%s: a screen reader should hear %q first:\n%s", kind, want[0], out)
		}
		if !strings.Contains(out, `<span class="sw-alert__icon" aria-hidden="true">`+want[1]+`</span>`) {
			t.Errorf("%s: the mark should be %q and silent:\n%s", kind, want[1], out)
		}
		if strings.Contains(out, "sw-alert__kind") {
			t.Errorf("%s: no visible kind label repeats the title", kind)
		}
	}
}

// TestAnAlertTitleIsAHeading: a person moving by headings finds it.
func TestAnAlertTitleIsAHeading(t *testing.T) {
	if out := renderAlert(t, map[string]any{"kind": "danger", "title": "Not saved", "message": "The title is needed."}); !strings.Contains(out, `<h2 class="sw-alert__title">`) {
		t.Errorf("a title should be an h2:\n%s", out)
	}
	if out := renderAlert(t, map[string]any{"title": "Not saved", "message": "x", "level": 3}); !strings.Contains(out, `<h3 class="sw-alert__title">`) {
		t.Errorf("level 3 should make an h3:\n%s", out)
	}
}

// TestOnlyALiveAlertTakesARole: a message present on load is not announced
// whatever its role, so only one put on the page later, or an outcome,
// takes one: alert for a warning or error, status otherwise.
func TestOnlyALiveAlertTakesARole(t *testing.T) {
	if out := renderAlert(t, map[string]any{"kind": "danger", "message": "x"}); strings.Contains(out, "role=") || strings.Contains(out, "aria-live") {
		t.Errorf("an alert on the page at load should take no live role:\n%s", out)
	}
	for kind, role := range map[string]string{"danger": "alert", "warning": "alert", "success": "status", "info": "status"} {
		if out := renderAlert(t, map[string]any{"kind": kind, "message": "x", "live": true}); !strings.Contains(out, `role="`+role+`"`) {
			t.Errorf("a live %s alert should be role=%s:\n%s", kind, role, out)
		}
	}
}

// TestAnAlertCloseButtonIsNamedAndReachable: a native button, named for what
// it closes, never taken out of the Tab order.
func TestAnAlertCloseButtonIsNamedAndReachable(t *testing.T) {
	out := renderAlert(t, map[string]any{"kind": "success", "message": "Changes made.", "dismiss": true})
	for _, want := range []string{`<button type="button" class="sw-alert__close" data-dismiss aria-label="Close message">`, "sw-alert--dismissible"} {
		if !strings.Contains(out, want) {
			t.Errorf("a dismissible alert should carry %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, `tabindex="-1"`) {
		t.Errorf("the close button must stay in the Tab order")
	}
	if out := renderAlert(t, map[string]any{"message": "x"}); strings.Contains(out, "sw-alert__close") {
		t.Errorf("no close button unless asked for")
	}
}
