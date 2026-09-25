package render_test

// Tests for keyboard accessibility of the alert component (goal 0082).
// Verifies that a dismissible alert renders a close button reachable by Tab,
// the root region has role and aria-live attributes for live updates, and
// non-dismissible alerts omit the close button while keeping live regions.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestAlertKeyboard verifies acceptance items 2–3: a tab user can focus the
// close button on a dismissible alert, and the root region carries role and
// aria-live attributes. When no dismiss is set, no close button appears but
// live-region attributes remain so screen readers announce the message.
func TestAlertKeyboard(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	// --- Dismissible alert: success kind with dismiss button ---

	outDismiss, err := reg.Render("alert", map[string]any{
		"kind":    "success",
		"message": "Changes made.",
		"dismiss": true,
	})
	if err != nil {
		t.Fatalf("render dismissible alert: %v", err)
	}

	gotDismiss := string(outDismiss)

	// 1. The close button exists with the sw-alert__close class. A native <button>
	//    element is naturally focusable via Tab — no tabindex manipulation needed
	//    (Acceptance item 2).
	if !strings.Contains(gotDismiss, `class="sw-alert__close"`) {
		t.Errorf("dismissible alert should render a close button with class sw-alert__close;\ngot:\n%s", gotDismiss)
	}

	// 2. The close button is a native <button type="button"> element — this is how
	//    the template expresses "naturally tabbable" (Acceptance item 2).
	if !strings.Contains(gotDismiss, `<button`) {
		t.Errorf("dismissible alert should render a native <button> for close;\ngot:\n%s", gotDismiss)
	}

	// 3. No tabindex="-1" anywhere in the output — keyboard path is open (Acceptance
	//    item 2). A native button in normal flow is always reachable via Tab.
	if strings.Contains(gotDismiss, `tabindex="-1"`) {
		t.Errorf("dismissible alert must not use tabindex=\"-1\" (blocks tab focus);\ngot:\n%s", gotDismiss)
	}

	// 4. The root div carries role="status" because kind is "success" (not warning/
	//    danger). This tells screen readers to announce the content when it changes
	//    rather than interrupting immediately (Acceptance item 2).
	if !strings.Contains(gotDismiss, `role="status"`) {
		t.Errorf("dismissible success alert should have role=\"status\";\ngot:\n%s", gotDismiss)
	}

	// 5. The root div carries aria-live="polite" for a status-role region — polite
	//    announcements wait until the user is idle before speaking (Acceptance item 2).
	if !strings.Contains(gotDismiss, `aria-live="polite"`) {
		t.Errorf("dismissible success alert should have aria-live=\"polite\";\ngot:\n%s", gotDismiss)
	}

	// 6. The close button has an accessible name via aria-label="Close". Without this,
	//    a screen reader would only announce "button" with no purpose (Acceptance item 2).
	if !strings.Contains(gotDismiss, `aria-label="Close"`) {
		t.Errorf("close button should have aria-label=\"Close\" for accessible name;\ngot:\n%s", gotDismiss)
	}

	// --- Non-dismissible alert: info kind, no close button ---

	outNoDismiss, err := reg.Render("alert", map[string]any{
		"kind":    "info",
		"message": "No changes.",
	})
	if err != nil {
		t.Fatalf("render non-dismissible alert: %v", err)
	}

	gotNoDismiss := string(outNoDismiss)

	// 7. No close button class exists — a non-dismissible alert should not render
	//    one (Acceptance item 3).
	if strings.Contains(gotNoDismiss, `sw-alert__close`) {
		t.Errorf("non-dismissible alert must not have sw-alert__close;\ngot:\n%s", gotNoDismiss)
	}

	// 8. No data-dismiss attribute — confirms the template did not render a dismiss
	//    button when dismiss is false/absent (Acceptance item 3).
	if strings.Contains(gotNoDismiss, `data-dismiss`) {
		t.Errorf("non-dismissible alert must not have data-dismiss;\ngot:\n%s", gotNoDismiss)
	}

	// 9. The root div still carries role="status" for info kind (Acceptance item 3).
	if !strings.Contains(gotNoDismiss, `role="status"`) {
		t.Errorf("non-dismissible info alert should have role=\"status\";\ngot:\n%s", gotNoDismiss)
	}

	// 10. The root div still carries aria-live="polite" for info kind (Acceptance item 3).
	if !strings.Contains(gotNoDismiss, `aria-live="polite"`) {
		t.Errorf("non-dismissible info alert should have aria-live=\"polite\";\ngot:\n%s", gotNoDismiss)
	}

	// --- Warning kind: role="alert" + aria-live="assertive" ---

	outWarning, err := reg.Render("alert", map[string]any{
		"kind":    "warning",
		"message": "Something needs attention.",
	})
	if err != nil {
		t.Fatalf("render warning alert: %v", err)
	}

	gotWarning := string(outWarning)

	// 11. Warning kind should get role="alert" (not status) so screen readers
	//     interrupt immediately when the message appears (Acceptance item 4).
	if !strings.Contains(gotWarning, `role="alert"`) {
		t.Errorf("warning alert should have role=\"alert\";\ngot:\n%s", gotWarning)
	}

	// 12. Warning kind should get aria-live="assertive" to interrupt immediately
	//     rather than waiting for idle (Acceptance item 4).
	if !strings.Contains(gotWarning, `aria-live="assertive"`) {
		t.Errorf("warning alert should have aria-live=\"assertive\";\ngot:\n%s", gotWarning)
	}
}
