package server_test

// Project list page heading links must not embed raw field labels such as
// "(added …)" in their visible or adjacent text (acceptance item 3 of task 0681 / goal 0098).

import (
	"strings"
	"testing"
)

func TestProjectListPageNoRawAddedLabelInHeading(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{"title": "My Project"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/project").Body.String()

	rowStart := strings.Index(page, `href="/t/project/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find project row in page\n%s", truncate(page))
	}
	rowEnd := strings.Index(page[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of project row\n%s", truncate(page))
	}
	row := page[rowStart : rowStart+rowEnd]

	// The row must NOT contain "(added …)" as context.
	if strings.Contains(row, "sw-visually-hidden") && strings.Contains(strings.ToLower(row), "(added ") {
		t.Errorf("project list row visually hidden context must not start with '(added '; found in:\n%s", truncate(row))
	}
}
