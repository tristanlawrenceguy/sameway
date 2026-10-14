package server_test

// Heading links on record list pages must not embed raw field name prefixes
// like "(added …)" or "(due …)" inside the <a> element's text content. These
// are the three backlog items (0721, 0722, 0723) sharing one root cause:
// apartHTML from records told apart ends up inside link accessible names.

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestTaskListHeadingLinkNoRawDueLabel creates two tasks with the same title
// and different due dates so they are told apart by their "due" field, fetches
// /t/task, and asserts that no <a class="sw-row__link"> element contains
// "(due". Covers acceptance item 1 (backlog 0723).
func TestTaskListHeadingLinkNoRawDueLabel(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	y, m, d := tonight.Date()
	due1 := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, 3).Format("2006-01-02")
	due2 := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, -2).Format("2006-01-02")

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
		"title": "Buy coffee",
		"due":   due1,
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
		"title": "Buy coffee",
		"due":   due2,
	}), http.StatusCreated)

	body := get(t, h, "/t/task").Body.String()

	assertNoRawLabelInRowLink(t, body, "(due")
}

// TestActionListHeadingLinkNoRawAddedLabel creates three actions with the same
// title so they are told apart by when they were added (the "added" field),
// fetches /t/action, and asserts that no <a class="sw-row__link"> element
// contains "(added". Covers acceptance item 2 (backlog 0721).
func TestActionListHeadingLinkNoRawAddedLabel(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Test action",
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Test action",
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Test action",
	}), http.StatusCreated)

	body := get(t, h, "/t/action").Body.String()

	t.Logf("ROW LINKS:\n%v", extractRowLinks(body))

	assertNoRawLabelInRowLink(t, body, "(added")
}

// TestProjectListHeadingLinkNoRawAddedLabel creates three projects with the same
// title so they are told apart by when they were added (the "added" field),
// fetches /t/project, and asserts that no <a class="sw-row__link"> element
// contains "(added". Covers acceptance item 3 (backlog 0722).
func TestProjectListHeadingLinkNoRawAddedLabel(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title": "Website Redesign",
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title": "Website Redesign",
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title": "Website Redesign",
	}), http.StatusCreated)

	body := get(t, h, "/t/project").Body.String()

	assertNoRawLabelInRowLink(t, body, "(added")
}

// extractRowLinks returns all <a class="sw-row__link"> elements from the page.
func extractRowLinks(body string) []string {
	var links []string
	linkStart := 0
	for {
		idx := strings.Index(body[linkStart:], `<a class="sw-row__link"`)
		if idx < 0 {
			break
		}
		linkStart += idx
		closeA := strings.Index(body[linkStart:], "</a>")
		if closeA < 0 {
			break
		}
		links = append(links, body[linkStart:linkStart+closeA])
		linkStart += closeA
	}
	return links
}

// assertNoRawLabelInRowLink extracts each <a class="sw-row__link"> from the page
// and asserts that its content does not contain the given prefix. This catches
// cases where apart text like "(added …)" or "(due …)" leaks inside link text.
func assertNoRawLabelInRowLink(t *testing.T, body, prefix string) {
	t.Helper()
	linkStart := 0
	for {
		idx := strings.Index(body[linkStart:], `<a class="sw-row__link"`)
		if idx < 0 {
			break
		}
		linkStart += idx
		closeA := strings.Index(body[linkStart:], "</a>")
		if closeA < 0 {
			break
		}
		linkContent := body[linkStart : linkStart+closeA]
		if strings.Contains(linkContent, prefix) {
			t.Errorf("list row heading link must not contain %q; found in:\n%s", prefix, truncate(linkContent))
		}
		linkStart += closeA
	}
}
