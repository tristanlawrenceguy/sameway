package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A tracker shows the habits it names, by name or id, in that order, under
// the label it is given; a name that is no habit's is refused when
// written; and a habit's unit is said once by its amount field. In the
// agent evaluation a reading tracker showed every habit, its label was
// nowhere to be seen, and its field read "Amount in pages pages".
func TestATrackerShowsTheHabitsItNames(t *testing.T) {
	a, h := newApp(t)
	water, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Drink water", "cadence": "day", "target": 8, "unit": "glasses"})
	pages, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Pages per day", "cadence": "day", "target": 30, "unit": "pages"})
	a.Store.Create(server.HabitType, map[string]any{"name": "Morning stretch"})

	var made struct{ ID, Shows string }
	res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "tracker", "props": map[string]any{"label": "Reading", "habits": []any{"pages per day", water.ID}}})
	wantStatus(t, res, http.StatusCreated)
	decode(t, res, &made)
	if made.Shows != "2 habits: Pages per day and Drink water" {
		t.Errorf("a tracker written says which habits, got %q", made.Shows)
	}
	page := get(t, h, "/canvas/"+made.ID).Body.String()
	for _, want := range []string{
		`aria-label="Reading"`, `<h2 class="sw-tracker__title">Reading</h2>`,
		`<label class="sw-tracker__unit" for="log-` + pages.ID + `"><span class="sw-visually-hidden">Amount in </span>pages<span class="sw-visually-hidden"> for Pages per day</span></label>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the tracker shows %s\n%s", want, truncate(page))
		}
	}
	if strings.Contains(page, "Morning stretch") {
		t.Error("a habit not named is left out")
	}
	if p, w := strings.Index(page, `data-id="`+pages.ID+`"`), strings.Index(page, `data-id="`+water.ID+`"`); p < 0 || w < 0 || p > w {
		t.Error("the habits are in the order named")
	}
	for _, twice := range []string{"pages pages", "glasses glasses", `aria-hidden="true">pages<`} {
		if strings.Contains(page, twice) {
			t.Errorf("the unit is said once: %s", twice)
		}
	}

	res = postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "tracker", "props": map[string]any{"habits": []any{"Page per day"}}})
	wantStatus(t, res, http.StatusUnprocessableEntity)
	if body := res.Body.String(); !strings.Contains(body, "there is no habit Page per day (did you mean Pages per day?)") {
		t.Errorf("a habit not there is said, with the one likely meant: %s", body)
	}
	// Without a label, none is shown over it; it is still named.
	res = postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "tracker", "props": map[string]any{"tags": []any{}}})
	wantStatus(t, res, http.StatusCreated)
	decode(t, res, &made)
	if page := get(t, h, "/canvas/"+made.ID).Body.String(); !strings.Contains(page, `aria-label="Keeping up"`) || strings.Contains(page, `sw-tracker__title">Keeping up`) {
		t.Error("a tracker with no label shows no heading and is called Keeping up")
	}
}
