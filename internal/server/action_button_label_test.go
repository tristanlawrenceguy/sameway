package server_test

import (
	"strings"
	"testing"
)

// TestActionButtonLabelTrimsLongActionName verifies that when an action has a
// long multi-word title, only three words appear after "Run" in the button's
// visible label. The full untrimmed name is preserved for screen readers in
// the visually-hidden context span. (Acceptance 1.)
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
	// After the change: "Run Agent Test Show" (not "Run Agent Test Show Action").
	if !strings.Contains(page, `<button`) {
		t.Fatal("action detail page should contain a button")
	}

	// The full untrimmed name must NOT appear as visible text after "Run".
	// It should only appear in the visually-hidden span.
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

	// The full untrimmed name must appear in the visually-hidden span.
	if !strings.Contains(page, `<span class="sw-visually-hidden"> Agent Test Show Action</span>`) {
		t.Errorf("full action title should be preserved for screen readers; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelShortNameUnchanged verifies that when the action name is
// three words or fewer, the full name appears after "Run" with no trimming.
// (Acceptance 2.)
func TestActionButtonLabelShortNameUnchanged(t *testing.T) {
	a, h := newApp(t)

	// Create an action whose title is exactly three words — at the limit.
	rec, err := a.Store.Create("action", map[string]any{
		"title": "Turn on alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	wantLabel := ">Run Turn on alarm<"
	if !strings.Contains(page, wantLabel) {
		t.Errorf("button should show full short name 'Run Turn on alarm'; page body:\n%s", truncate(page))
	}

	// The visually-hidden context should also contain the same full title.
	if !strings.Contains(page, `<span class="sw-visually-hidden"> Turn on alarm</span>`) {
		t.Errorf("full action title should be preserved for screen readers; page body:\n%s", truncate(page))
	}
}

// TestActionButtonLabelTwoWords verifies that a two-word action name is not
// trimmed — the full name appears after "Run". (Acceptance 2.)
func TestActionButtonLabelTwoWords(t *testing.T) {
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

	// Full untrimmed title should also be in the visually-hidden context.
	if !strings.Contains(page, `<span class="sw-visually-hidden"> test-action-xyz modified</span>`) {
		t.Errorf("full title should appear in visually-hidden span; page body:\n%s", truncate(page))
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

	// Full title should be in visually-hidden context.
	if !strings.Contains(page, `<span class="sw-visually-hidden"> This is an action with seven words in its name today</span>`) {
		t.Errorf("full title should appear for screen readers; page body:\n%s", truncate(page))
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

	// Full title with original spacing in context for screen readers.
	if !strings.Contains(page, `<span class="sw-visually-hidden"> Run    the    big    red   button</span>`) {
		t.Errorf("full original title should be preserved in visually-hidden span; page body:\n%s", truncate(page))
	}
}
