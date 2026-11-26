package bench

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// shapes are the changes to the kinds of things and to the settings, each
// asked in plain words: a new property, a new kind, another choice on a
// pick-list, and three settings a person names by what they want.
func shapes() []request {
	setting := func(key string, ok func(string) bool) func(*app.App, string) string {
		return func(a *app.App, _ string) string {
			v := a.Workspace.Get(key)
			return want(ok(v), key+" is "+v)
		}
	}
	return []request{
		{name: "shape-field", say: "Add a due date to my notes",
			check: func(a *app.App, _ string) string {
				t, _ := a.Store.Types().Get("note")
				for _, f := range t.Fields {
					if strings.Contains(f.Name, "due") && f.Type == "datetime" {
						return ""
					}
				}
				return "note has no due date field"
			}},
		{name: "shape-type", say: "I want to keep my recipes here: each has a name, its ingredients and how many minutes it takes",
			check: func(a *app.App, _ string) string {
				t, ok := a.Store.Types().Get("recipe")
				return want(ok && len(t.Fields) >= 3, "no recipe type with its three fields")
			}},
		{name: "shape-choice", say: "Tasks need another status: waiting",
			check: func(a *app.App, _ string) string {
				t, _ := a.Store.Types().Get("task")
				f, _ := t.Field("status")
				return want(f != nil && strings.Contains(strings.ToLower(strings.Join(f.Values, ",")), "waiting"), "status has no waiting choice")
			}},
		{name: "shape-bigger", say: "The writing is too small for me, make it bigger",
			check: setting("ui.text", func(v string) bool { return v == "large" || v == "larger" })},
		{name: "shape-reader", say: "I use a screen reader",
			check: setting("ui.needs", func(v string) bool { return strings.Contains(strings.ToLower(v), "screen reader") })},
		{name: "shape-clock", say: "Show times the 24-hour way, like 14:00",
			check: setting("ui.clock", func(v string) bool { return v == "24" })},
	}
}
