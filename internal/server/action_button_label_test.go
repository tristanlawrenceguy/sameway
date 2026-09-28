package server_test

import (
	"strings"
	"testing"
)

// TestActionButtonLabelTrimsLongActionName verifies that when an action has a
// long multi-word title, only three words appear after "Run" in the button's
// visible label. No duplicate context span is rendered. (Acceptance 1.)
func TestActionButtonLabelTrimsLongActionName(t *testing.T) {
	a, h := newApp(t)

	// Create an action with a title longer than three words.
	rec, err := a.Store.Create("action", map[string]any{
		"title": "Agent Test Show Action",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// The visible button label should be "Run" + at most 3 words.
	if !strings.Contains(page, `<button`) {
		t.Fatal("action detail page should contain a button")
	}

	// The full untrimmed name must NOT appear as visible text after "Run".
	for _, line := range strings.Split(page, "\n") {
		if strings.Contains(line, `>Run Agent Test Show Action<`) ||
			strings.Contains(line, ">Run Agent Test Show Action ") {
			t.Errorf("visible button label must not show more than 3 words after Run: found %q", line)
		}
	}

	// The trimmed visible label "Run Agent Test Show" should be present.
	if !strings.Contains(page, `>Run Agent Test Show`) {
		t.Errorf("button label should contain 'Run Agent Test Show'; page body:\n%s", truncate(page))
	}

	// No visually-hidden context span should follow the Run button.
	runBtn := `<button`
	idx := strings.Index(page, runBtn)
	if idx < 0 {
		t.Fatal("could not find Run button")
	}
	rest := page[idx:]
	btnEnd := strings.Index(rest, "</button>")
	if btnEnd >= 0 && strings.Contains(rest[:btnEnd], `class="sw-visually-hidden"`) {
		t.Errorf("Run button must not have a visually-hidden context span; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelShortNameUnchanged verifies that when the action name is
// three words or fewer, the full name appears after "Run" with no trimming.
// (Acceptance 2.)
func TestActionButtonLabelShortNameUnchanged(t *testing.T) {
	a, h := newApp(t)

	// Create an action whose title is exactly three words — at the limit.
	rec, err := a.Store.Create("action", map[string]any{
		"title": "Daily Stretch Break Reminder",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	wantLabel := ">Run Daily Stretch Break<"
	if !strings.Contains(page, wantLabel) {
		t.Errorf("button should show trimmed name 'Run Daily Stretch Break'; page body:\n%s", truncate(page))
	}

	// No visually-hidden context span after the Run button.
	runBtn := `<button`
	idx := strings.Index(page, runBtn)
	if idx < 0 {
		t.Fatal("could not find Run button")
	}
	rest := page[idx:]
	btnEnd := strings.Index(rest, "</button>")
	if btnEnd >= 0 && strings.Contains(rest[:btnEnd], `class="sw-visually-hidden"`) {
		t.Errorf("Run button must not have a visually-hidden context span; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelTwoWordName verifies that an action name with two words
// is shown in full (no trimming needed under the 3-word limit).
func TestActionButtonLabelTwoWordName(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "test-action-xyz modified",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// Two words is within the 3-word limit, so full name appears.
	wantLabel := ">Run test-action-xyz modified<"
	if !strings.Contains(page, wantLabel) {
		t.Errorf("button should show full two-word name 'Run test-action-xyz modified'; page body:\n%s", truncate(page))
	}

	// No visually-hidden context span after the Run button.
	runBtn := `<button`
	idx := strings.Index(page, runBtn)
	if idx < 0 {
		t.Fatal("could not find Run button")
	}
	rest := page[idx:]
	btnEnd := strings.Index(rest, "</button>")
	if btnEnd >= 0 && strings.Contains(rest[:btnEnd], `class="sw-visually-hidden"`) {
		t.Errorf("Run button must not have a visually-hidden context span; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelVeryLongName verifies that an action name with many words
// is trimmed to exactly three, not six or any other count. (Acceptance 1.)
func TestActionButtonLabelVeryLongName(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "This is an action with seven words in its name today",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// The visible label should be "Run This is an" (3 words after Run).
	if !strings.Contains(page, `>Run This is an`) {
		t.Errorf("button should show 'Run This is an'; page body:\n%s", truncate(page))
	}

	// Must not contain the 4th word as visible text.
	if strings.Contains(page, ">Run This is an action") {
		t.Error("visible label must be trimmed to exactly 3 words after Run")
	}

	// No visually-hidden context span after the Run button.
	runBtn := `<button`
	idx := strings.Index(page, runBtn)
	if idx < 0 {
		t.Fatal("could not find Run button")
	}
	rest := page[idx:]
	btnEnd := strings.Index(rest, "</button>")
	if btnEnd >= 0 && strings.Contains(rest[:btnEnd], `class="sw-visually-hidden"`) {
		t.Errorf("Run button must not have a visually-hidden context span; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelWithExtraSpaces verifies that extra whitespace between
// words does not cause issues — strings.Fields normalizes spaces. (Acceptance 1.)
func TestActionButtonLabelWithExtraSpaces(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Run    the    big    red   button",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// The label should use trimmed (space-normalized) words.
	if !strings.Contains(page, `>Run Run the big`) {
		t.Errorf("button should show 'Run Run the big'; page body:\n%s", truncate(page))
	}

	// No visually-hidden context span after the Run button.
	runBtn := `<button`
	idx := strings.Index(page, runBtn)
	if idx < 0 {
		t.Fatal("could not find Run button")
	}
	rest := page[idx:]
	btnEnd := strings.Index(rest, "</button>")
	if btnEnd >= 0 && strings.Contains(rest[:btnEnd], `class="sw-visually-hidden"`) {
		t.Errorf("Run button must not have a visually-hidden context span; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelOneWord verifies that a single-word action name is not
// trimmed. (Acceptance 2.)
func TestActionButtonLabelOneWord(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Deploy",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	wantLabel := ">Run Deploy<"
	if !strings.Contains(page, wantLabel) {
		t.Errorf("button should show full name 'Run Deploy'; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelExactlyThreeWords verifies that a title with exactly
// three words is shown in full — no trimming at the boundary. (Acceptance 2.)
func TestActionButtonLabelExactlyThreeWords(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Agent Test Show",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	wantLabel := ">Run Agent Test Show<"
	if !strings.Contains(page, wantLabel) {
		t.Errorf("button should show all three words 'Run Agent Test Show'; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelSendEmail verifies that a two-word action name appears
// in full. (Acceptance 2.)
func TestActionButtonLabelSendEmail(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Send email",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	wantLabel := ">Run Send email<"
	if !strings.Contains(page, wantLabel) {
		t.Errorf("button should show full name 'Run Send email'; page body:\n%s", truncate(page))
	}
}
