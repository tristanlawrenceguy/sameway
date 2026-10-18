package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// The ways agents went wrong this week are answered with the way that
// works: a page's action asked under /api names itself without /api, a
// field with its operator sent bare is pointed at where, and a tab there
// is not says Home is canvas=.
func TestAnAgentsWrongTurnIsAnsweredWithTheWay(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	for _, c := range []struct{ method, path, want string }{
		{http.MethodPost, "/api/today/let-go", "POST /today/let-go is: a page's, without /api"},
		{http.MethodGet, "/api/review", "GET /review is: a page's, without /api"},
		{http.MethodGet, "/api/task?updated_at>=-7d", "?where=updated_at>=-7d"},
		{http.MethodGet, "/api/block?canvas=home", "Home is canvas= with nothing after it"},
	} {
		res := do(t, h, c.method, c.path, nil, "")
		if res.Code < 400 || !strings.Contains(res.Body.String(), c.want) {
			t.Errorf("%s %s: %d, want %q in %s", c.method, c.path, res.Code, c.want, res.Body.String())
		}
	}
	if res := get(t, h, "/api/nothing-like-it"); !strings.Contains(res.Body.String(), "there is no content type nothing-like-it") {
		t.Errorf("a route that is nowhere says so as before: %s", res.Body.String())
	}
}

// A habit read over the API says where it stands, as the assistant is
// told: the agent need not add its entries up by hand.
func TestAHabitReadSaysWhereItStands(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	run, _ := a.Store.Create("habit", map[string]any{"name": "Running", "cadence": "week", "target": 20, "unit": "miles"})
	a.Store.Create("entry", map[string]any{"habit": run.ID, "amount": 5, "at": a.Now().Format("2006-01-02")})
	if body := get(t, h, "/api/habit/"+run.ID).Body.String(); !strings.Contains(body, `"standing": "5 of 20 miles`) {
		t.Errorf("the habit says where it stands: %s", body)
	}
}
