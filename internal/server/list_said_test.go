package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A list page narrowed or ordered says so in a person's words: how many of
// what, which, and in what order by what kind of thing the field is; no
// field names and no stray comma.
func TestAListPageSaysItsOrderInWords(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	habit, _ := a.Store.Create("habit", map[string]any{"name": "Running", "cadence": "day", "unit": "miles"})
	for i := 1; i <= 3; i++ {
		postJSON(t, h, http.MethodPost, "/api/entry", map[string]any{"habit": habit.ID, "amount": i, "at": when.Store(time.Now().AddDate(0, 0, -i), false)})
		postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Task", "due": when.Store(time.Now().AddDate(0, 0, i), true)})
	}
	for path, want := range map[string]string{
		"/t/entry?order=-at":                   "3 entries, by logged, latest first.",
		"/t/entry?order=amount":                "3 entries, by quantity, smallest first.",
		"/t/task?where=done%3Dfalse&order=due": "3 tasks, not done, by due, earliest first.",
		"/t/task?order=-updated_at":            "3 tasks, by when last changed, latest first.",
	} {
		page := get(t, h, path).Body.String()
		if !strings.Contains(page, want) {
			i := strings.Index(page, `<p class="sw-muted">`)
			t.Errorf("%s says %q, got %.120s", path, want, page[max(i, 0):])
		}
		if strings.Contains(page, " , ") || strings.Contains(page, "largest or newest") {
			t.Errorf("%s has a stray comma or both readings", path)
		}
	}
}
