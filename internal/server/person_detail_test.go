package server_test

// On a person detail page (e.g., /t/person/<id>), at most one date reference
// appears after the heading, and it has a clear label. No duplicate dates are
// announced by a screen reader. Acceptance items 2–3 of task 0341.

import (
	"net/http"
	"strings"
	"testing"
)

// TestPersonDetailHasNoWhenMadeSpan asserts that a person detail page does not
// include a sw-detail__when span with the creation date, since it duplicates
// the "last contacted" date already in the count fact. Acceptance items 2–3.
func TestPersonDetailHasNoWhenMadeSpan(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var person struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/person", map[string]any{
		"name": "Priya",
	}), &person)

	page := get(t, h, "/t/person/"+person.ID).Body.String()

	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("person detail page lede must not have a sw-detail__when span — the count fact already shows last-contacted date")
	}
}
