package agenda

import (
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// What a person keeps up each day (a relative's tablets twice a day, a
// run, half an hour's reading) was on a tracker block or the habit's own
// page, and never where the day is planned, so a missed dose was found in
// the evening, if at all. Today now lists each daily habit not yet met
// today, with where it stands and the same Log press a tracker has (Undo
// takes it back); once met, it leaves. A weekly or monthly one is not
// there: it is not today's, and would sit on every day of the week. One
// that only records, or keeps within a limit, is not something to do.

// todayHabit is a daily habit not met yet.
type todayHabit struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Href     string `json:"page"`
	Progress string `json:"progress"`
}

// habitsToDo are the daily habits not yet met today, in name order.
func (s *Service) habitsToDo(now time.Time) []todayHabit {
	if _, ok := s.app.Types.Get(records.HabitType); !ok {
		return nil
	}
	recs, _ := s.app.Store.List(records.HabitType, store.ListOptions{OrderBy: "name"})
	out := []todayHabit{}
	for _, r := range recs {
		h := track.Normal(blocks.HabitOf(r))
		if archived, _ := r.Fields["archived"].(bool); archived || h.Cadence != "day" || h.Aim != track.Reach {
			continue
		}
		sum := track.Summarise(h, blocks.EntriesOf(s.app.Store, h.ID), now, 7)
		if sum.Met {
			continue
		}
		out = append(out, todayHabit{ID: h.ID, Title: h.Name, Href: relate.Page(records.HabitType, h.ID), Progress: track.Progress(h, sum) + missedSaid(sum)}) // today_habits_missed.go
	}
	return out
}

// habitsSection is Today's Habits: each with where it stands and Log.
func (s *Service) habitsSection(habits []todayHabit) string {
	if len(habits) == 0 {
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<h2>Habits</h2>` + s.habitTwins(habits) + `<ul class="sw-plain sw-rows">`) // today_habits_missed.go
	for _, h := range habits {
		b.WriteString(`<li class="sw-row"><p class="sw-row__title"><a class="sw-link" href="` + esc(h.Href) + `">` + esc(h.Title) + `</a> <span class="sw-muted sw-small">` + esc(h.Progress) + `</span></p>`)
		b.WriteString(`<div class="sw-cluster">` + string(s.Form(ui.Form{Action: "/habit/" + h.ID + "/log", From: "/today",
			Button: &ui.Button{Label: "Log", Context: h.Title, Variant: ui.Secondary}})) + `</div></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}
