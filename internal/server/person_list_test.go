package server_test

// On /t/person (people list page), no standalone <time> element appears below any
// person's summary text — each entry shows only the interaction count line with its
// inline date. Acceptance item 1 of task 0341.

import (
	"net/http"
	"strings"
	"testing"
)

// TestPersonListRowHasNoBareTimeElement asserts that person list rows on /t/person
// do not contain a standalone <time> element for UpdatedAt, since the count fact
// already carries date info ("last Wed 14 Oct"). Acceptance item 1.
func TestPersonListRowHasNoBareTimeElement(t *testing.T) {
	t.Parallel()

	a, h := newApp(t)

	var person struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/person", map[string]any{
		"name": "Mum",
	}), &person)

	// Seed an interaction so the count fact carries date info ("last ...").
	a.Store.Create("interaction", map[string]any{"summary": "Coffee", "person": person.ID})

	body := get(t, h, "/t/person").Body.String()

	rowStart := strings.Index(body, `href="/t/person/`+person.ID)
	if rowStart < 0 {
		t.Fatalf("could not find person row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of person row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, "<time") {
		t.Errorf("person list row must not contain a standalone <time> element; the count fact already carries date info. Row:\n%s", truncate(row))
	}
}
