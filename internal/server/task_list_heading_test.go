package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// Task list heading links must not show raw field name prefixes like
// "added" or "due" as visible text inside the anchor tag. Distinguishing
// dates go in a <time> element outside the link (backlog 0723).

func TestTaskListHeadingLinksContainNoAddedOrDue(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	overdue, later := seedTwins(t, h)
	page := get(t, h, "/t/task").Body.String()

	// Acceptance 1 & 2: no heading link contains "added" or "due" as visible text.
	insideLink := `>Call plumber<span class="sw-visually-hidden"> (` + overdue + `)</span></a>`
	if strings.Contains(page, insideLink) {
		t.Errorf("apart text %q is still visible inside a heading link (acceptance 1/2)\n%s", overdue, truncate(page))
	}

	// Acceptance 3: apart text goes into the aria-label on the anchor, not in a separate span.
	if !strings.Contains(page, `aria-label="Call plumber (`+overdue+")") {
		t.Errorf("apart text %q should be in the link's aria-label (acceptance 3)\n%s", overdue, truncate(page))
	}

	_ = later
}

func TestUndatedTaskTwinsAlsoKeepApartOutsideHeadingLink(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	for range 2 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
			"title": "Call plumber",
		}), http.StatusCreated)
	}
	page := get(t, h, "/t/task").Body.String()

	// Undated twins are told apart by ordinal ("first of 2") or "(added ...)"
	// depending on timestamp resolution. This text must not be inside any
	// heading <a> tag; it goes into the aria-label instead (acceptance 1/2).
	if strings.Contains(page, `>Call plumber<span class="sw-visually-hidden">`) {
		t.Errorf("apart text is still visible inside a heading link (acceptance 1/2)\n%s", truncate(page))
	}

	// Both twins must have apart text in their aria-labels.
	ariaCount := strings.Count(page, `aria-label="Call plumber (`)
	if ariaCount < 2 {
		t.Errorf("expected at least 2 task links with apart text in aria-label, got %d\n%s", ariaCount, truncate(page))
	}
}

func TestUniqueTaskTitleHasNoApartTextInHeadingLink(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
		"title": "Single task title",
	}), http.StatusCreated)
	page := get(t, h, "/t/task").Body.String()

	// A unique title has no apart text at all. The link should be just the title.
	if !strings.Contains(page, `<a class="sw-row__link" href="/t/task/`) {
		t.Fatalf("task list page should have a task link\n%s", truncate(page))
	}

	// No "added" or "due" visible in any heading link text.
	for _, kw := range []string{"(added ", "(due "} {
		if strings.Contains(page, `>Single task title`+kw) {
			t.Errorf("unique task should not have apart text %q inside the link\n%s", kw, truncate(page))
		}
	}
}
