package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// The Log button says Log, and to a reader which habit: its whole name, or
// the name cut at a word with an ellipsis, never left hanging on "for".
func TestHabitLogButtonSaysWhichHabit(t *testing.T) {
	a, h := newApp(t)
	for name, want := range map[string]string{
		"Daily stretch break":                          "Daily stretch break",
		"Exercise for thirty minutes every single day": "Exercise for thirty minutes every single…",
	} {
		hab, err := a.Store.Create(server.HabitType, map[string]any{"name": name, "cadence": "day", "target": 1})
		if err != nil {
			t.Fatal(err)
		}
		page := get(t, h, "/t/"+server.HabitType+"/"+hab.ID).Body.String()
		button := `sw-pressable">Log<span class="sw-visually-hidden"> ` + want + `</span></button>`
		if !strings.Contains(page, button) {
			t.Errorf("the Log button for %q should read Log %q; page body:\n%s", name, want, truncate(page))
		}
	}
}

// TestAddButtonCorrectArticle verifies that the actions list page button uses
// "Add an" (not "Add a") when the type name starts with a vowel sound, and
// stays at ≤3 visible words total.  This covers Acceptance 2.
func TestAddButtonCorrectArticle(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/action").Body.String()

	if !strings.Contains(rec, "Add an action") {
		t.Errorf("actions list should say 'Add an action'; found button text:\n%s", truncate(rec))
	}

	if strings.Contains(rec, ">Add a action<") || strings.Contains(rec, ">Add a action ") {
		t.Error("actions list must use 'an' before vowel-starting type name; found 'Add a action'")
	}
}

// TestImportButtonLabelIsShort verifies that import links on content type list
// pages show three words or fewer visible text (not the full schema name).
// This covers Acceptance 3.
func TestImportButtonLabelIsShort(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note").Body.String()

	if !strings.Contains(rec, ">Import<") {
		t.Errorf("import link should show just 'Import' as visible text; page body:\n%s", truncate(rec))
	}
}

// TestHelpPageSettingLabelsAreShort verifies that help page setting toggle
// buttons show at most three visible words — no multi-word descriptions as
// button labels.  This covers Acceptance 4.
func TestHelpPageSettingLabelsAreShort(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/help").Body.String()

	// These current values exceed three words and must be trimmed.
	badLabels := []string{
		"Calmly, one at a time",
		"All at once, without motion",
		"Show when pointed at",
	}
	for _, bad := range badLabels {
		if strings.Contains(rec, ">"+bad) {
			t.Errorf("help page setting button must be ≤3 visible words; found label %q (%d words)",
				bad, len(strings.Fields(bad)))
		}
	}
}

// TestRunButtonNoDuplicateContext verifies that the Run button on action detail
// pages does not duplicate the full title in a visually-hidden context span.
// Screen reader output should be "Run {trimmed title}" with no appended full name.
// This covers Acceptance 5.
func TestRunButtonNoDuplicateContext(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Daily Stretch Break Reminder",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// The full untrimmed title must NOT appear in a visually-hidden context span.
	fullContextPattern := `<span class="sw-visually-hidden"> Daily Stretch Break Reminder</span>`
	if strings.Contains(page, fullContextPattern) {
		t.Errorf("Run button must not duplicate the full title as context; "+
			"found %q — screen readers hear 'Run Daily Stretch' + 'Daily Stretch Break Reminder'",
			fullContextPattern)
	}

	// There should be no sw-visually-hidden span after the Run button at all.
	runBtn := `<button type="submit"`
	idx := strings.Index(page, runBtn)
	if idx < 0 {
		t.Fatal("action detail page should contain a submit button")
	}
	rest := page[idx:]
	btnEnd := strings.Index(rest, "</button>")
	if btnEnd >= 0 {
		buttonContent := rest[:btnEnd]
		if viStart := strings.Index(buttonContent, `class="sw-visually-hidden"`); viStart >= 0 {
			t.Errorf("Run button must not have a visually-hidden context span; "+
				"found one after the label: %q",
				strings.TrimSpace(buttonContent[viStart+28:min(viStart+150, len(buttonContent))]))
		}
	}

	// The visible trimmed label should still be present.
	if !strings.Contains(page, ">Run Daily Stretch") {
		t.Errorf("visible button label should contain trimmed title 'Run Daily Stretch'; page body:\n%s", truncate(page))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
