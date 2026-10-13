package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestTaskListRowLinkNoAddedPrefix asserts that the heading link for each task
// row on /t/task does not contain "(added ..." as part of its text content. The
// disambiguation text should be outside the <a> tag, in a separate element.
// (Acceptance 1.)
func TestTaskListRowLinkNoAddedPrefix(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	now := time.Now()
	taskTitle := "Buy groceries"
	_, err := a.Store.Create("task", map[string]any{
		"title": taskTitle,
		"due":   now.AddDate(0, 0, -1).Format(time.RFC3339), // yesterday
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	// Find the row by its heading — this comes after the <a> tag opens so
	// that searching within the captured row still sees the full opening.
	rowStart := strings.Index(body, `<h3 class="sw-row__title"`)
	if rowStart < 0 {
		t.Fatal("could not find task row in /t/task page")
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of task row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// Extract the <a> tag from within this row.
	aTagStart := strings.Index(row, `<a class="sw-row__link"`)
	if aTagStart < 0 {
		t.Fatal("could not find sw-row__link in task row")
	}
	// Find the closing </a>.
	aTagBody := row[aTagStart:]
	closeA := strings.Index(aTagBody, `</a>`)
	if closeA < 0 {
		t.Fatal("could not find closing </a> in task row")
	}
	linkContent := aTagBody[:closeA]

	// The link must NOT contain "(due" as part of the visible text either.
	if strings.Contains(linkContent, "(due") {
		t.Errorf("task list row link must not contain '(due'; found in:\n%s", truncate(row))
	}

	// The link must NOT contain "(added" anywhere — that is the raw field label.
	if strings.Contains(linkContent, "(added") {
		t.Errorf("task list row link must not contain '(added'; found in:\n%s", truncate(row))
	}
}

// TestTaskListRowLinkNoDuePrefix asserts that task rows with a due date show
// no "(due ..." text inside the heading link. The disambiguation should be
// outside the <a> tag. (Acceptance 1.)
func TestTaskListRowLinkNoDuePrefix(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	now := time.Now()
	taskTitle := "Return library books"
	_, err := a.Store.Create("task", map[string]any{
		"title": taskTitle,
		"due":   now.AddDate(0, 0, -3).Format(time.RFC3339), // three days ago
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	rowStart := strings.Index(body, `<h3 class="sw-row__title"`)
	if rowStart < 0 {
		t.Fatal("could not find task row in /t/task page")
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of task row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	aTagStart := strings.Index(row, `<a class="sw-row__link"`)
	if aTagStart < 0 {
		t.Fatal("could not find sw-row__link in task row")
	}
	closeA := strings.Index(row[aTagStart:], `</a>`)
	if closeA < 0 {
		t.Fatal("could not find closing </a> in task row")
	}
	linkContent := row[aTagStart : aTagStart+closeA]

	if strings.Contains(linkContent, "(due") {
		t.Errorf("task list row link must not contain '(due'; found in:\n%s", truncate(row))
	}
}

// TestTaskListRowDateOutsideLink asserts that date information (added date, due
// date) still appears on each task row but outside the heading link, in a
// secondary element within the row. (Acceptance 2.)
func TestTaskListRowDateOutsideLink(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	now := time.Now()
	taskTitle := "Weekly errands"
	// Two tasks with the same title so Apart generates disambiguation text.
	_, err := a.Store.Create("task", map[string]any{
		"title": taskTitle,
		"due":   now.AddDate(0, 0, -1).Format(time.RFC3339), // yesterday
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Store.Create("task", map[string]any{
		"title": taskTitle,
		"due":   now.AddDate(0, 0, -2).Format(time.RFC3339), // two days ago
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	rowStart := strings.Index(body, `<h3 class="sw-row__title"`)
	if rowStart < 0 {
		t.Fatal("could not find task row in /t/task page")
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of task row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// The row must still contain the disambiguation/apart text somewhere.
	// It should be in a visually-hidden span with class sw-visually-hidden,
	// but NOT inside the <a> tag.
	if !strings.Contains(row, `class="sw-visually-hidden"`) {
		t.Errorf("task row must contain a visually-hidden span for apart text; found in:\n%s", truncate(row))
	}

	// The apart text should appear outside the </a> closing tag. Find where
	// the link ends and check that sw-visually-hidden appears after it.
	aTagStart := strings.Index(row, `<a class="sw-row__link"`)
	if aTagStart < 0 {
		t.Fatal("could not find sw-row__link in task row")
	}
	closeA := strings.Index(row[aTagStart:], `</a>`)
	if closeA < 0 {
		t.Fatal("could not find closing </a> in task row")
	}
	linkEndPos := aTagStart + closeA

	// After the link, there should still be content with sw-visually-hidden.
	afterLink := row[linkEndPos:]
	if !strings.Contains(afterLink, `class="sw-visually-hidden"`) {
		t.Errorf("visually-hidden apart text must appear after the closing </a>; found in:\n%s", truncate(row))
	}

	// The link's own text should be just the title (trimmed to 6 words).
	linkText := row[aTagStart : aTagStart+closeA]
	visibleWords := strings.Fields(strings.ReplaceAll(linkText, "<", " "))
	if len(visibleWords) > 6 {
		t.Errorf("link text should be trimmed to at most 6 visible words, got %d: %q", len(visibleWords), linkText)
	}

	// The title itself must still be present.
	if !strings.Contains(linkText, taskTitle) {
		t.Errorf("task row link must contain the task title %q; found in:\n%s", taskTitle, truncate(row))
	}
}

// TestTaskListRowHasVisuallyHiddenApartAfterLink asserts that the visually-hidden
// apart span (containing date disambiguation) appears after the </a> tag rather
// than inside it. This is the structural fix: moving apartHTML from inside to
// outside the heading link. (Acceptance 2.)
func TestTaskListRowHasVisuallyHiddenApartAfterLink(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	now := time.Now()
	// Two tasks with the same title so Apart generates disambiguation text.
	_, err := a.Store.Create("task", map[string]any{
		"title": "Quick Test Task",
		"due":   now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Store.Create("task", map[string]any{
		"title": "Quick Test Task",
		"due":   now.AddDate(0, 0, -1).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task")
	wantStatus(t, page, http.StatusOK)
	body := page.Body.String()

	rowStart := strings.Index(body, `<h3 class="sw-row__title"`)
	if rowStart < 0 {
		t.Fatal("could not find task row in /t/task page")
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of task row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// The pattern </a></hN><span class="sw-visually-hidden"> should exist.
	// That is: the apart span comes AFTER the heading element closes, not before.
	if !strings.Contains(row, `</a></h`) {
		t.Errorf("task row link must close with </a> before the heading tag ends; found in:\n%s", truncate(row))
	}

	// The sw-visually-hidden span should appear after the </hN> closing.
	hClosePos := strings.Index(row, `</a></h`)
	if hClosePos >= 0 {
		afterH := row[hClosePos:]
		nextVisSpan := strings.Index(afterH, `class="sw-visually-hidden"`)
		if nextVisSpan < 0 {
			t.Errorf("visually-hidden apart span must appear after </a></hN>; found in:\n%s", truncate(row))
		} else if nextVisSpan < 5 {
			// The span should be right after the heading (within a few chars for
			// whitespace), but definitely not before </a>.
			t.Errorf("visually-hidden apart span must appear AFTER the heading closes, not inside it; found in:\n%s", truncate(row))
		}
	}
}
