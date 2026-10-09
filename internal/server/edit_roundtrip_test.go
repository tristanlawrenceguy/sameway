package server_test

import (
	"html"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Edit then Save with nothing changed changes nothing: every field the
// edit form fills in, as the person reads it, is read back as the value it
// was, on the 12-hour and the 24-hour clock, for a day alone, midday,
// midnight and a minute either side of it. The day field's words changed
// (Sat 19 Sep 2026 at 2pm, midday), and they must still read back.
func TestTheEditFormReadsBackWhatItShows(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	project, _ := a.Store.Create("project", map[string]any{"title": "Garden"})
	day := time.Date(2026, 9, 19, 0, 0, 0, 0, time.Local)
	values := []string{
		when.Store(day, true),
		when.Store(day.Add(14*time.Hour), false),
		when.Store(day.Add(12*time.Hour), false),
		when.Store(day, false),                   // midnight
		when.Store(day.Add(-time.Minute), false), // a minute before it
		when.Store(day.Add(time.Minute), false),  // a minute after it
		when.Store(day.Add(9*time.Hour+5*time.Minute), false),
	}
	defer a.Workspace.Set("ui.clock", "")
	for _, clock := range []string{"12", "24"} {
		if err := a.Workspace.Set("ui.clock", clock); err != nil {
			t.Fatal(err)
		}
		for _, due := range values {
			task, _ := a.Store.Create("task", map[string]any{"title": "Repot", "due": due, "status": "doing", "done": false,
				"project": project.ID, "tags": []any{"home", "pots"}, "notes": "The big pot.", "repeat": "every week"})
			before, _ := a.Store.Get("task", task.ID)
			page := get(t, h, "/t/task/"+task.ID).Body.String()
			form := url.Values{}
			// What the form is built from (editfields.go): a field's words, or
			// the value a choice, a box or a ref stores.
			for _, m := range regexp.MustCompile(`data-prop="([a-z]+)"[^>]*?data-source="([^"]*)"[^>]*>([^<]*)<`).FindAllStringSubmatch(between(page, "<template data-edit-fields>", "</template>"), -1) {
				v := html.UnescapeString(m[3])
				if v == "" {
					v = html.UnescapeString(m[2])
				}
				if m[1] == "project" || m[1] == "status" || m[1] == "done" || m[1] == "repeat" {
					v = html.UnescapeString(m[2])
				}
				form.Set("prop-"+m[1], v)
			}
			if form.Get("prop-due") == "" {
				t.Fatalf("the edit form holds no due date for %s", due)
			}
			wantStatus(t, postForm(t, h, "/t/task/"+task.ID+"/props", form), http.StatusSeeOther)
			got, _ := a.Store.Get("task", task.ID)
			for _, f := range []string{"due", "status", "done", "project", "repeat"} {
				if got.Fields[f] != before.Fields[f] {
					t.Errorf("%s-hour clock, due %s: %s %v came back as %v (the form said %q)", clock, due, f, before.Fields[f], got.Fields[f], form.Get("prop-"+f))
				}
			}
		}
	}
}

// between is the part of s after start and before the end after it.
func between(s, start, end string) string {
	i := regexp.MustCompile(regexp.QuoteMeta(start)).FindStringIndex(s)
	if i == nil {
		return ""
	}
	rest := s[i[1]:]
	j := regexp.MustCompile(regexp.QuoteMeta(end)).FindStringIndex(rest)
	if j == nil {
		return rest
	}
	return rest[:j[0]]
}
