package bench

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// request is one thing a person asks, what is there before, and how to
// tell it was done: check says what is wrong, or nothing.
type request struct {
	name  string
	seed  func(t *testing.T, a *app.App)
	say   string
	check func(a *app.App, reply string) string
}

func requests() []request {
	return []request{
		{name: "checklist", say: "Put a checklist of milk, eggs and bread on the page",
			check: func(a *app.App, _ string) string {
				return all(titled(a, "milk"), titled(a, "eggs"), titled(a, "bread"), block(a, "", ""))
			}},
		{name: "note", say: "Make a note called Garden plans: I want tomatoes and basil this year",
			check: func(a *app.App, _ string) string {
				n := find(a, "note", "title", "garden")
				if n == nil {
					return "no note titled garden"
				}
				return want(has(n, "body", "tomato"), "the note does not say tomatoes")
			}},
		{name: "remind", say: "Remind me to call the dentist next Friday",
			check: func(a *app.App, _ string) string {
				day := next(time.Friday)
				for _, typ := range []string{"task", "reminder"} {
					if r := find(a, typ, "title", "dentist"); r != nil {
						return want(onDay(r, day), fmt.Sprintf("the %s is not on %s: %v", typ, day, r.Fields))
					}
				}
				return "no task or reminder about the dentist"
			}},
		{name: "meeting", say: "I have a meeting with Ana Silva and Joe Brown next Tuesday at 2pm about pricing. I want to record its audio so it gets transcribed and written up afterwards.",
			check: func(a *app.App, _ string) string {
				ev := find(a, "event", "title", "pric")
				if ev == nil {
					return "no pricing event"
				}
				day := next(time.Tuesday)
				return all(want(onDay(ev, day) && hourIs(ev, "starts", 14), fmt.Sprintf("starts %v, not %s 14:00", ev.Fields["starts"], day)),
					want(count(ev.Fields["people"]) == 2, fmt.Sprintf("people %v", ev.Fields["people"])),
					want(len(list(a, "reminder")) > 0, "no reminder to record it"))
			}},
		{name: "chart", seed: tasks, say: "Show a chart of my tasks by status",
			check: func(a *app.App, _ string) string { return block(a, "chart", `"task"`) }},
		{name: "todo-list", seed: tasks, say: "Show my tasks that are not done, soonest due first",
			check: func(a *app.App, _ string) string { return block(a, "", "due") }},
		{name: "calendar", say: "Show a calendar of my events",
			check: func(a *app.App, _ string) string { return block(a, "calendar", "") }},
		{name: "countdown", say: "Count down the days to my holiday in Lisbon on 20 December",
			check: func(a *app.App, _ string) string {
				ev := find(a, "event", "title", "lisbon")
				if ev == nil {
					ev = find(a, "event", "where", "lisbon")
				}
				if ev == nil {
					return "no Lisbon event"
				}
				return want(onDay(ev, fmt.Sprintf("%d-12-20", time.Now().Year())), fmt.Sprintf("starts %v", ev.Fields["starts"]))
			}},
		{name: "habit", say: "Keep a habit: 8 glasses of water a day",
			check: func(a *app.App, _ string) string {
				h := find(a, "habit", "name", "water")
				if h == nil {
					return "no water habit"
				}
				return want(fmt.Sprint(h.Fields["target"]) == "8", fmt.Sprintf("target %v", h.Fields["target"]))
			}},
		{name: "new-type", say: "I want to keep track of the books I read, with the author and my rating out of 5",
			check: func(a *app.App, _ string) string {
				for _, n := range a.Store.Types().Names() {
					t, _ := a.Store.Types().Get(n)
					_, au := t.Field("author")
					_, ra := t.Field("rating")
					if au && ra {
						return ""
					}
				}
				return "no type with author and rating"
			}},
		{name: "add-field", say: "Give tasks a priority: low, medium or high",
			check: func(a *app.App, _ string) string {
				t, _ := a.Store.Types().Get("task")
				f, ok := t.Field("priority")
				if !ok {
					return "task has no priority"
				}
				return want(strings.Contains(strings.Join(f.Values, ","), "medium"), fmt.Sprintf("priority is %s %v", f.Type, f.Values))
			}},
		{name: "tab", seed: tasks, say: "Make a tab called Work with my tasks on it",
			check: func(a *app.App, _ string) string {
				c := find(a, chat.CanvasType, "name", "work")
				if c == nil {
					return "no Work tab"
				}
				for _, b := range list(a, chat.BlockType) {
					if b.Fields["canvas"] == c.ID && strings.Contains(jsonOf(b.Fields["props"]), "task") {
						return ""
					}
				}
				return "nothing with tasks on the Work tab"
			}},
		{name: "tick", seed: tasks, say: "I bought the paint, tick it off",
			check: func(a *app.App, _ string) string {
				p := find(a, "task", "title", "paint")
				return want(p != nil && (p.Fields["status"] == "done" || p.Fields["done"] == true), "Buy paint is not done")
			}},
		{name: "question", seed: boiler, say: "When is the boiler engineer coming?",
			check: func(_ *app.App, reply string) string {
				return want(strings.Contains(strings.ToLower(reply), "thursday"), "the reply does not say Thursday")
			}},
		{name: "organise", seed: novel, say: "Organise my novel The Long Field: Chapter 1 and Chapter 2 are its parts, in that order, and Submission guidelines is material for it",
			check: func(a *app.App, _ string) string {
				novel := find(a, "note", "title", "long field")
				c1, c2, g := find(a, "note", "title", "chapter 1"), find(a, "note", "title", "chapter 2"), find(a, "note", "title", "guidelines")
				return all(want(fmt.Sprint(c1.Fields[chat.PartOf]) == novel.ID && fmt.Sprint(c2.Fields[chat.PartOf]) == novel.ID, "the chapters are not its parts"),
					want(strings.Contains(fmt.Sprint(g.Fields[chat.MaterialFor]), novel.ID), "the guidelines are not its material"))
			}},
		{name: "suggest", seed: letter, say: "Fix the spelling in my note Letter to Sam",
			check: func(a *app.App, _ string) string {
				n := find(a, "note", "title", "letter")
				return want(len(list(a, "suggestion")) > 0 || !has(n, "body", "wonderfull"), "neither fixed nor suggested")
			}},
		{name: "automate", say: "Whenever a task is marked done, send its title to https://example.com/hook",
			check: func(a *app.App, _ string) string {
				if act := find(a, "action", "url", "example.com"); act != nil {
					return want(act.Fields["when"] == "changed" && act.Fields["what"] == "task", fmt.Sprintf("the action runs on %v %v", act.Fields["when"], act.Fields["what"]))
				}
				return want(len(list(a, "proposal")) > 0, "no action and nothing asked")
			}},
	}
}

func tasks(t *testing.T, a *app.App) {
	for _, f := range []map[string]any{
		{"title": "Buy paint", "status": "todo", "due": "2026-10-10"},
		{"title": "Send invoice", "status": "doing", "due": "2026-10-05"},
		{"title": "Book flights", "status": "done"},
	} {
		mustCreate(t, a, "task", f)
	}
}

func boiler(t *testing.T, a *app.App) {
	mustCreate(t, a, "note", map[string]any{"title": "Boiler", "body": "The engineer comes on Thursday at 9. The code for the boiler is 4471."})
	mustCreate(t, a, "note", map[string]any{"title": "Garden", "body": "Plant garlic in October."})
}

func novel(t *testing.T, a *app.App) {
	for _, title := range []string{"The Long Field", "Chapter 1", "Chapter 2", "Submission guidelines"} {
		mustCreate(t, a, "note", map[string]any{"title": title, "body": "Words of " + title + "."})
	}
}

func letter(t *testing.T, a *app.App) {
	mustCreate(t, a, "note", map[string]any{"title": "Letter to Sam", "body": "Thank you for the wonderfull dinner. It was realy lovely to see you agian."})
}

func mustCreate(t *testing.T, a *app.App, typ string, f map[string]any) {
	if _, err := a.Store.Create(typ, f); err != nil {
		t.Fatal(err)
	}
}

func list(a *app.App, typ string) []*store.Record {
	r, _ := a.Store.List(typ, store.ListOptions{})
	return r
}

// find is the first record of a type whose field holds the words.
func find(a *app.App, typ, field, words string) *store.Record {
	for _, r := range list(a, typ) {
		if has(r, field, words) {
			return r
		}
	}
	return nil
}

func has(r *store.Record, field, words string) bool {
	return r != nil && strings.Contains(strings.ToLower(fmt.Sprint(r.Fields[field])), words)
}

// titled says whether any content record is titled or named with the words.
func titled(a *app.App, words string) string {
	for _, typ := range a.Store.Types().Names() {
		if find(a, typ, "title", words) != nil || find(a, typ, "name", words) != nil {
			return ""
		}
	}
	return "nothing titled " + words
}

// block says whether a block of the component (any, when "") whose props
// hold the words is on a canvas.
func block(a *app.App, component, words string) string {
	for _, b := range list(a, chat.BlockType) {
		if b.Fields["component"] == "chat" {
			continue
		}
		if (component == "" || b.Fields["component"] == component) && strings.Contains(jsonOf(b.Fields["props"]), words) {
			return ""
		}
	}
	return fmt.Sprintf("no %s block with %s", component, words)
}

func jsonOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func count(v any) int {
	switch l := v.(type) {
	case []any:
		return len(l)
	case []string:
		return len(l)
	}
	return 0
}

// next is the coming day with that weekday, as a person means it.
func next(wd time.Weekday) string {
	now := time.Now()
	d := (int(wd) - int(now.Weekday()) + 7) % 7
	if d == 0 {
		d = 7
	}
	return now.AddDate(0, 0, d).Format("2006-01-02")
}

func onDay(r *store.Record, day string) bool {
	for _, f := range []string{"due", "at", "starts"} {
		if v, ok := r.Fields[f].(string); ok && v != "" {
			if t, _, ok := when.Parse(v, time.Now()); ok && t.In(time.Local).Format("2006-01-02") == day {
				return true
			}
		}
	}
	return false
}

func hourIs(r *store.Record, field string, hour int) bool {
	v, _ := r.Fields[field].(string)
	t, _, ok := when.Parse(v, time.Now())
	return ok && t.In(time.Local).Hour() == hour
}

func want(ok bool, why string) string {
	if ok {
		return ""
	}
	return why
}

func all(whys ...string) string {
	var out []string
	for _, w := range whys {
		if w != "" {
			out = append(out, w)
		}
	}
	return strings.Join(out, "; ")
}
