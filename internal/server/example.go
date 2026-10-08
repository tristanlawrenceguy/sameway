package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A newcomer could only find out what Sameway is like by filling it, and
// an empty page shows nothing of what a full one does. Try it with an
// example makes a workspace of its own, beside theirs, with a believable
// week in it dated around today: tasks late, due and done, a project,
// the day's dentist and the week's lunch, a habit kept for five days, a
// reminder for tonight, notes and a person; and a Home tab that shows them.
// It is a workspace like any other: changed freely, and deleted from
// Workspaces into the trash when they are done with it.

// exampleName is the example workspace's name.
const exampleName = "Example"

func (s *Server) tryExample(w http.ResponseWriter, r *http.Request) {
	dir, err := s.blank(exampleName)
	if err != nil {
		dir, err = s.blank(exampleName + " " + time.Now().Format("2 Jan 15.04"))
	}
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	if err := fillExample(dir, time.Now()); err != nil {
		s.showWorkspaces(w, r, "The example could not be filled: "+err.Error())
		return
	}
	url, notStarted := s.start(dir)
	if notStarted != nil {
		s.showWorkspaceCreated(w, r, exampleName, dir, "new", notStarted)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// fillExample puts a week of things in the workspace at dir, around now.
func fillExample(dir string, now time.Time) error {
	a, err := app.Load(dir, false)
	if err != nil {
		return err
	}
	defer a.Close()
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") }
	at := func(d, h, m int) string { return fmt.Sprintf("%s %02d:%02d", day(d), h, m) }
	made := map[string]string{}
	put := func(key, typ string, f map[string]any) error {
		rec, _, err := records.Write(a.Store, "created", typ, "", f)
		if err == nil && key != "" {
			made[key] = rec.ID
		}
		return err
	}
	steps := []struct {
		key, typ string
		f        func() map[string]any
	}{
		{"ana", "person", func() map[string]any {
			return map[string]any{"name": "Ana Silva", "email": "ana@example.com", "organisation": "Silva Design"}
		}},
		{"garden", "project", func() map[string]any { return map[string]any{"title": "Garden", "status": "active"} }},
		{"", "task", func() map[string]any { return map[string]any{"title": "Renew the passport", "due": day(-2)} }},
		{"", "task", func() map[string]any { return map[string]any{"title": "Call the bank about the card", "due": day(0)} }},
		{"", "task", func() map[string]any {
			return map[string]any{"title": "Send the invoice", "due": day(2), "for": made["ana"]}
		}},
		{"", "task", func() map[string]any {
			return map[string]any{"title": "Order tomato seeds", "due": day(3), "project": made["garden"]}
		}},
		{"", "task", func() map[string]any {
			return map[string]any{"title": "Fix the fence", "due": day(6), "project": made["garden"]}
		}},
		{"", "task", func() map[string]any {
			return map[string]any{"title": "Book the car service", "due": day(-1), "done": true}
		}},
		{"", "event", func() map[string]any { return map[string]any{"title": "Dentist", "starts": at(0, 15, 30)} }},
		{"", "event", func() map[string]any {
			return map[string]any{"title": "Lunch with Ana", "starts": at(1, 12, 30), "where": "Café Lisboa"}
		}},
		{"", "event", func() map[string]any {
			return map[string]any{"title": "Team meeting", "starts": at(4, 9, 0), "repeat": "every week"}
		}},
		{"", "reminder", func() map[string]any { return map[string]any{"title": "Take the bins out", "at": at(0, 20, 0)} }},
		{"water", "habit", func() map[string]any {
			return map[string]any{"name": "Water", "cadence": "day", "target": 8, "aim": "reach", "combine": "sum", "unit": "glasses"}
		}},
		{"", "note", func() map[string]any {
			return map[string]any{"title": "Holiday ideas", "body": "- Lisbon in May\n- A week walking in the Dolomites\n- Somewhere with a beach and no plans", "tags": []any{"travel"}}
		}},
		{"", "note", func() map[string]any {
			return map[string]any{"title": "Pancakes", "body": "200 g flour, 2 eggs, 300 ml milk, a pinch of salt.\n\nWhisk, rest ten minutes, fry thin.", "tags": []any{"recipes"}}
		}},
	}
	for _, st := range steps {
		if err := put(st.key, st.typ, st.f()); err != nil {
			return fmt.Errorf("%s: %w", st.typ, err)
		}
	}
	for d, glasses := range []int{6, 8, 9, 7, 8} {
		if err := put("", "entry", map[string]any{"habit": made["water"], "at": day(d - 5), "amount": glasses}); err != nil {
			return fmt.Errorf("entry: %w", err)
		}
	}
	for _, b := range []map[string]any{
		{"component": "collection", "span": 6, "props": map[string]any{"type": "task", "where": []any{"done=false", "due<=+7d"}, "order": "due", "label": "Due this week", "show": []any{"due", "project"}}},
		{"component": "calendar", "span": 6, "props": map[string]any{"types": []any{"task", "event", "reminder"}, "caption": "This month"}},
		{"component": "tracker", "span": 6, "props": map[string]any{"habits": []any{"Water"}}},
	} {
		if _, err := records.ApplyOps(a.Store, records.Op{Type: records.BlockType, After: a.Chat.BlockFields(b)}); err != nil {
			return fmt.Errorf("block: %w", err)
		}
	}
	return nil
}
