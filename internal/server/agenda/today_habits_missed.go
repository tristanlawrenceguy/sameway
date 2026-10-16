package agenda

import (
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Two things a daily habit on Today now says when they are so, and only
// then. Back after two weeks, a person caring for her mother's medicines
// saw the morning dose not logged today, and nothing of the days before:
// a habit usually kept says the days of the last week it was not. And
// with two habits called Water, she could not tell which to log: two
// that nothing tells apart are said once, with the press a list has to
// keep the newest (server/keep_one.go).

// missedSaid is ", not kept Tue and Wed" for a habit kept most of the six
// days before today and missed on some; "" otherwise, and before the
// habit was kept at all.
func missedSaid(sum track.Summary) string {
	if len(sum.Last) < 2 {
		return ""
	}
	var kept int
	var missed []string
	for _, p := range sum.Last[:len(sum.Last)-1] { // the days before today
		switch {
		case !sum.First.IsZero() && p.Start.Before(sum.First):
		case p.Met:
			kept++
		default:
			missed = append(missed, p.Start.Format("Mon"))
		}
	}
	if kept < 3 || len(missed) == 0 {
		return ""
	}
	return ", not kept " + schema.AndNames(missed)
}

// habitTwins is the line for habits on Today that nothing tells apart,
// with the press to keep the newest, or "".
func (s *Service) habitTwins(habits []todayHabit) string {
	t, ok := s.app.Types.Get(records.HabitType)
	if !ok {
		return ""
	}
	shown := map[string]bool{}
	for _, h := range habits {
		shown[h.ID] = true
	}
	var b strings.Builder
	for _, g := range records.Twins(s.app.Store, t) {
		if !shown[g[0].ID] && !shown[g[len(g)-1].ID] {
			continue
		}
		name, _ := g[0].Fields["name"].(string)
		b.WriteString(`<div class="sw-cluster sw-quiet"><span>` + template.HTMLEscapeString(schema.Count(len(g), "habit")+" called "+name+", and nothing tells them apart.") + `</span>` +
			string(s.Form(ui.Form{Action: "/t/habit/" + g[0].ID + "/keep-one", From: "/today", Button: &ui.Button{Label: "Keep the newest", Context: name, Variant: ui.Secondary}})) + `</div>`)
	}
	return b.String()
}
