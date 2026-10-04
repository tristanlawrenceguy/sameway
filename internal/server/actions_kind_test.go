package server_test

// The actions list page must not show internal type names or schema descriptors
// such as "Call a web address kind" under action entries. Acceptance items 1–2 of
// task 0216 (backlog 0517).

import (
	"net/http"
	"strings"
	"testing"
)

// TestActionsListDoesNotShowKindBadge verifies that the actions listing page
// does not render a badge showing the action's kind enum value label such as
// "Call a web address". The entire page is one type of thing (action), so
// repeating the kind descriptor is noise. (Acceptance 1.)
func TestActionsListDoesNotShowKindBadge(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
	}), &act)

	body := get(t, h, "/t/action").Body.String()

	// The row for this action should not contain a badge with its kind label.
	// Look at the li element for this record and check it has no badge with kind text.
	rowStart := strings.Index(body, `href="/t/action/`+act.ID)
	if rowStart < 0 {
		t.Fatalf("could not find action row in page\n%s", truncate(body))
	}
	// Grab the li that contains this record (next </li>).
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of action row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, "Call a web address") {
		t.Errorf("action row must not show kind descriptor 'Call a web address'; found in:\n%s", truncate(row))
	}
	if strings.Contains(row, `sw-visually-hidden">kind`) ||
		strings.Contains(row, `sw-visually-hidden" >kind`) {
		t.Errorf("action row must not expose the field name 'kind' even visually hidden; found in:\n%s", truncate(row))
	}
}

// TestActionDetailLedeDoesNotShowKindBadge verifies that on an action's own
// detail page the lede line does not include a badge showing its kind. (Acceptance 1.)
func TestActionDetailLedeDoesNotShowKindBadge(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
	}), &act)

	rec := get(t, h, "/t/action/"+act.ID)
	body := rec.Body.String()

	// Extract just the lede paragraph.
	ledeStart := strings.Index(body, `<p class="sw-lede">`)
	if ledeStart < 0 {
		t.Fatalf("action detail has no lede\n%s", truncate(body))
	}
	ledeEnd := strings.Index(body[ledeStart:], "</p>")
	if ledeEnd < 0 {
		t.Fatalf("could not find end of lede\n%s", truncate(body))
	}
	lede := body[ledeStart : ledeStart+ledeEnd]

	if strings.Contains(lede, "Call a web address") {
		t.Errorf("action lede must not show kind descriptor 'Call a web address';\n%s", lede)
	}
}

// TestActionsListStillShowsUpdatedTime verifies that removing the kind badge does
// not remove other useful facts like the "Updated …" timestamp. (Acceptance 1.)
func TestActionsListStillShowsUpdatedTime(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
	}), &act)

	body := get(t, h, "/t/action").Body.String()

	if !strings.Contains(body, "sw-muted") {
		t.Errorf("actions list should still show when each action was updated; found in:\n%s", truncate(body))
	}
}
