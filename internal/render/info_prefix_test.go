package render_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestInfoAlertNoKindPrefix checks that a kind="info" alert does NOT include
// the "Note:" kind label prefix or any <span class="sw-alert__kind"> element.
// The blue styling and icon (when present) communicate meaning. Acceptance item 1.
func TestInfoAlertNoKindPrefix(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "info",
		"title":   "Saved.",
		"message": "Your edits were applied.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The word "Note:" must NOT appear anywhere in the output.
	if contains(out, "Note:") {
		t.Errorf("info alert should not contain \"Note:\" prefix;\ngot:\n%s\nwant: no \"Note:\" text", out)
	}

	// No sw-alert__kind span at all for info kind.
	if contains(out, `<span class="sw-alert__kind">`) {
		t.Errorf("info alert should not render a sw-alert__kind span;\ngot:\n%s\nwant: no kind span", out)
	}

	// The remaining signals must still be present.
	if !contains(out, `data-kind="info"`) {
		t.Error("output missing data-kind=\"info\"")
	}
	if !contains(out, "Saved.") {
		t.Errorf("title text \"Saved.\" should appear; got:\n%s", out)
	}
	if !contains(out, "Your edits were applied.") {
		t.Errorf("message text should appear; got:\n%s", out)
	}

	// role must still be status for info kind.
	if !contains(out, `role="status"`) {
		t.Error("info alert missing role=\"status\"")
	}
}

// TestWarningAlertNoKindPrefix checks that a kind="warning" alert does NOT
// include the "Warning:" kind label prefix or any <span class="sw-alert__kind">
// element. The amber styling and icon (when present) communicate meaning.
// Acceptance item 2.
func TestWarningAlertNoKindPrefix(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "warning",
		"title":   "No model connected",
		"message": "Edit the llm section and restart.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The word "Warning:" must NOT appear anywhere in the output.
	if contains(out, "Warning:") {
		t.Errorf("warning alert should not contain \"Warning:\" prefix;\ngot:\n%s\nwant: no \"Warning:\" text", out)
	}

	// No sw-alert__kind span at all for warning kind.
	if contains(out, `<span class="sw-alert__kind">`) {
		t.Errorf("warning alert should not render a sw-alert__kind span;\ngot:\n%s\nwant: no kind span", out)
	}

	// The remaining signals must still be present.
	if !contains(out, `data-kind="warning"`) {
		t.Error("output missing data-kind=\"warning\"")
	}
	if !contains(out, "No model connected") {
		t.Errorf("title text should appear; got:\n%s", out)
	}
	if !contains(out, "Edit the llm section and restart.") {
		t.Errorf("message text should appear; got:\n%s", out)
	}

	// role must be alert for warning kind.
	if !contains(out, `role="alert"`) {
		t.Error("warning alert missing role=\"alert\"")
	}
}

// TestDangerAlertNoKindPrefix checks that a kind="danger" alert does NOT
// include the "Error:" kind label prefix or any <span class="sw-alert__kind">
// element, while still keeping its ⚠ icon in .sw-alert__icon. Acceptance item 3.
func TestDangerAlertNoKindPrefix(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "danger",
		"title":   "Could not reach the model",
		"message": "Connection refused at http://localhost:11434.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The word "Error:" must NOT appear anywhere in the output.
	if contains(out, "Error:") {
		t.Errorf("danger alert should not contain \"Error:\" prefix;\ngot:\n%s\nwant: no \"Error:\" text", out)
	}

	// No sw-alert__kind span at all for danger kind.
	if contains(out, `<span class="sw-alert__kind">`) {
		t.Errorf("danger alert should not render a sw-alert__kind span;\ngot:\n%s\nwant: no kind span", out)
	}

	// The icon span must still be present with its unicode character.
	if !contains(out, `<span class="sw-alert__icon"`) {
		t.Errorf("danger alert should have .sw-alert__icon span;\ngot:\n%s", out)
	}
	if !contains(out, "\u26a0") {
		t.Errorf("danger alert icon missing unicode character ⚠;\ngot:\n%s", out)
	}

	// The remaining signals must still be present.
	if !contains(out, `data-kind="danger"`) {
		t.Error("output missing data-kind=\"danger\"")
	}
	if !contains(out, "Could not reach the model") {
		t.Errorf("title text should appear; got:\n%s", out)
	}

	// role must be alert for danger kind.
	if !contains(out, `role="alert"`) {
		t.Error("danger alert missing role=\"alert\"")
	}
}

// TestSuccessAlertNoKindPrefixStillHolds checks that a success alert still does
// NOT include the "Success:" kind label prefix, confirming task 0166's fix is
// not regressed. Acceptance item 4.
func TestSuccessAlertNoKindPrefixStillHolds(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "success",
		"title":   "Changes saved",
		"message": "Your edits were applied.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The word "Success:" must NOT appear anywhere in the output.
	if contains(out, "Success:") {
		t.Errorf("success alert should not contain \"Success:\" prefix;\ngot:\n%s\nwant: no \"Success:\" text", out)
	}

	// No sw-alert__kind span at all for success kind.
	if contains(out, `<span class="sw-alert__kind">`) {
		t.Errorf("success alert should not render a sw-alert__kind span;\ngot:\n%s\nwant: no kind span", out)
	}

	// The remaining signals must still be present.
	if !contains(out, `data-kind="success"`) {
		t.Error("output missing data-kind=\"success\"")
	}
	if !contains(out, "Changes saved") {
		t.Errorf("title text should appear; got:\n%s", out)
	}

	// role must still be status.
	if !contains(out, `role="status"`) {
		t.Error("success alert missing role=\"status\"")
	}
}
