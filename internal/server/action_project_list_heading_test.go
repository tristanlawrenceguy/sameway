package server_test

// Action and project list page heading links must not contain raw field name
// prefixes like "(added ..." inside their <a> elements. Distinguishing text
// goes outside the link in a visually-hidden span (backlog 0721, 0722).

import (
	"net/http"
	"strings"
	"testing"
)

// TestActionListHeadingLinksContainNoAdded verifies that action list page
// heading links contain only record titles — no "(added ..." inside any <a>.
// It creates two actions with the same title, fetches /t/action, and checks
// that apart text is outside </a>, not inside the link (acceptance 1).
func TestActionListHeadingLinksContainNoAdded(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	for range 2 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
			"title": "Call plumber",
		}), http.StatusCreated)
	}

	body := get(t, h, "/t/action").Body.String()

	// Acceptance 1: no heading link contains "(added ..." as visible text.
	if strings.Contains(body, `>Call plumber<span class="sw-visually-hidden">`) {
		t.Errorf("apart text is still inside a heading <a> tag (acceptance 1)\n%s", truncate(body))
	}

	// Acceptance: apart text should be OUTSIDE the anchor tag.
	if !strings.Contains(body, `</a><span class="sw-visually-hidden">`) {
		t.Errorf("apart text should be outside </a> in a visually hidden span\n%s", truncate(body))
	}

	// Both twin actions should have apart text outside their links.
	count := strings.Count(body, `</a><span class="sw-visually-hidden">`)
	if count < 2 {
		t.Errorf("expected at least 2 action links with apart text outside </a>, got %d\n%s", count, truncate(body))
	}

	// Date metadata (Changed ...) still appears as visible text in sw-row__meta.
	if !strings.Contains(body, "sw-row__meta") {
		t.Errorf("action list rows should have a <p class=\"sw-row__meta\"> with date info\n%s", truncate(body))
	}

	if !strings.Contains(body, "Changed ") {
		t.Errorf("action list row should show visible \"Changed ...\" metadata\n%s", truncate(body))
	}
}

// TestProjectListHeadingLinksContainNoAdded verifies that project list page
// heading links contain only record titles — no "(added ..." inside any <a>.
// It creates two projects with the same title, fetches /t/project, and checks
// that apart text is outside </a>, not inside the link (acceptance 2).
func TestProjectListHeadingLinksContainNoAdded(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	for range 2 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
			"title": "Garden",
		}), http.StatusCreated)
	}

	body := get(t, h, "/t/project").Body.String()

	// Acceptance 2: no heading link contains "(added ..." as visible text.
	if strings.Contains(body, `>Garden<span class="sw-visually-hidden">`) {
		t.Errorf("apart text is still inside a heading <a> tag (acceptance 2)\n%s", truncate(body))
	}

	// Acceptance: apart text should be OUTSIDE the anchor tag.
	if !strings.Contains(body, `</a><span class="sw-visually-hidden">`) {
		t.Errorf("apart text should be outside </a> in a visually hidden span\n%s", truncate(body))
	}

	// Both twin projects should have apart text outside their links.
	count := strings.Count(body, `</a><span class="sw-visually-hidden">`)
	if count < 2 {
		t.Errorf("expected at least 2 project links with apart text outside </a>, got %d\n%s", count, truncate(body))
	}

	// Date metadata: project has a DayField (due) but may not have it set.
	// When unset, facts() produces no date text for it; the row still has
	// apart info outside </a>, which is what we test above.
}

// TestActionUniqueTitleHasNoApartTextInHeadingLink verifies that an action
// with a unique title has no apart text in its heading link. (Acceptance 1.)
func TestActionUniqueTitleHasNoApartTextInHeadingLink(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Single action title",
	}), http.StatusCreated)

	body := get(t, h, "/t/action").Body.String()

	if !strings.Contains(body, `<a class="sw-row__link" href="/t/action/`) {
		t.Fatalf("action list page should have an action link\n%s", truncate(body))
	}

	for _, kw := range []string{"(added ", "(id "} {
		if strings.Contains(body, `>Single action title`+kw) {
			t.Errorf("unique action should not have apart text %q inside the link\n%s", kw, truncate(body))
		}
	}
}

// TestProjectUniqueTitleHasNoApartTextInHeadingLink verifies that a project
// with a unique title has no apart text in its heading link. (Acceptance 2.)
func TestProjectUniqueTitleHasNoApartTextInHeadingLink(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title": "Single project title",
	}), http.StatusCreated)

	body := get(t, h, "/t/project").Body.String()

	if !strings.Contains(body, `<a class="sw-row__link" href="/t/project/`) {
		t.Fatalf("project list page should have a project link\n%s", truncate(body))
	}

	for _, kw := range []string{"(added ", "(id "} {
		if strings.Contains(body, `>Single project title`+kw) {
			t.Errorf("unique project should not have apart text %q inside the link\n%s", kw, truncate(body))
		}
	}
}
