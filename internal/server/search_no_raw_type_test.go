package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// Search result headings show only the record title — no raw schema type
// names like "Event", "Task", or "Block" appear as standalone text after
// the title. The visible .sw-card__meta paragraph carries no type name, and
// the visually-hidden span carries no capitalized type word either (acceptance 1).

func TestSearchHeadingHasNoRawTypeNameVisible(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Boiler repair"}), http.StatusCreated)

	results := get(t, h, "/search?q=boiler").Body.String()

	// The visible meta paragraph must not contain a capitalized type word
	// like "Note" or "Event". It should be empty (the %s placeholder in
	// hitItem becomes "").
	if strings.Contains(results, `<p class="sw-card__meta">Note</p>`) {
		t.Errorf("visible meta contains raw type name: %s", truncate(results))
	}
	if strings.Contains(results, `<p class="sw-card__meta">Event</p>`) {
		t.Errorf("visible meta contains raw type name Event: %s", truncate(results))
	}
	if strings.Contains(results, `<p class="sw-card__meta">Task</p>`) {
		t.Errorf("visible meta contains raw type name Task: %s", truncate(results))
	}

	// The page should be a search results page.
	if !strings.Contains(results, `<title>Search:`) {
		t.Errorf("did not get a search results page: %s", truncate(results))
	}
}

// The ", close in meaning" match indicator text is removed from search result
// headings (acceptance 4). A near-match heading has no "close in meaning"
// substring inside the sw-visually-hidden span.

func TestSearchHeadingHasNoCloseInMeaningText(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	boiler, _ := a.Store.Create("note", map[string]any{"title": "Boiler repair, call Marek"})
	a.Store.Create("note", map[string]any{"title": "Ask the plumber about the tap"})
	search.UseMeaning(a.Store, func(string) []string { return []string{"note/" + boiler.ID} })

	results := get(t, h, "/search?q=plumber").Body.String()

	// The heading accessible name must not contain ", close in meaning" inside
	// the sw-visually-hidden span. We check every visually-hidden span because
	// only one result may be a near-match; the others are exact and pass.
	for {
		start := strings.Index(results, `class="sw-visually-hidden"`)
		if start == -1 {
			break
		}
		end := strings.Index(results[start:], "</span>")
		if end == -1 {
			t.Fatalf("no closing </span> for visually-hidden: %s", truncate(results))
		}
		hidden := results[start : start+end]

		if strings.Contains(hidden, "close in meaning") {
			t.Errorf("visually-hidden heading span contains 'close in meaning': %s", hidden)
		}
		results = results[start+end+len("</span>"):]
	}
}

// Where mixed-type search results need disambiguation, the kind is shown in
// lowercase and parentheses (e.g., "(event)") rather than as a standalone
// capitalized word like "Event" (acceptance 2).

func TestMixedTypeDisambiguationUsesLowercaseParentheses(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
		"title": "Call plumber",
		"due":   tonight.AddDate(0, 0, -3).Format("2006-01-02"),
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Call plumber",
	}), http.StatusCreated)

	results := get(t, h, "/search?q=plumber").Body.String()

	// The heading accessible names (sw-visually-hidden) must not contain
	// capitalized type words like " — Task" or " — Note". Check every result.
	for {
		start := strings.Index(results, `class="sw-visually-hidden"`)
		if start == -1 {
			break
		}
		end := strings.Index(results[start:], "</span>")
		if end == -1 {
			t.Fatalf("no closing </span> for visually-hidden: %s", truncate(results))
		}
		hidden := results[start : start+end]

		for _, kind := range []string{"Event", "Task", "Note", "Action"} {
			if strings.Contains(hidden, " — "+kind) {
				t.Errorf("visually-hidden span contains capitalized raw type %q: %s", kind, hidden)
			}
		}

		// If there is a parenthesized disambiguation it should be lowercase.
		if idx := strings.Index(hidden, "("); idx != -1 {
			closeParen := strings.Index(hidden[idx:], ")")
			if closeParen != -1 && closeParen > 0 {
				parensContent := hidden[idx+1 : idx+closeParen]
				for _, r := range parensContent {
					if r >= 'A' && r <= 'Z' {
						t.Errorf("parenthesized disambiguation is not lowercase %q in: %s", parensContent, hidden)
						break
					}
				}
			}
		}
		results = results[start+end+len("</span>"):]
	}
}

// Record list page rows do not display raw type names or unlabeled enum values
// as standalone text next to timestamps (acceptance 3). This is already true:
// Glance does not produce type-name facts.

func TestRecordListRowsHaveNoRawTypeName(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	for i := range 3 {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
			"title": "Task " + string(rune('A'+i)),
			"due":   tonight.Format("2006-01-02"),
		}), http.StatusCreated)
	}

	listPage := get(t, h, "/t/task").Body.String()

	// The row meta must not contain a raw type name like "Task" as standalone text.
	if strings.Contains(listPage, `<p class="sw-row__meta">Task</p>`) {
		t.Errorf("row meta contains raw type name: %s", truncate(listPage))
	}
}
