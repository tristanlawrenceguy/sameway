package server

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Things know what they are about. A reminder carries the page of the
// thing it is for and rings with it; every record's page offers the next
// thing to do with it (a reminder, a word with the assistant, its day on
// the calendar); a habit nudges through the clock when its time comes
// and it is not yet met; an entry is titled by its habit. The joins live
// here so each component stays itself.

// aboutOf reads a page path, /t/{type}/{id}, back to the record it names.
func (s *Server) aboutOf(path string) (*schema.Type, *store.Record, bool) {
	return blocks.AboutOf(s.app.Store, path)
}

// title is a record's title, with what the type alone cannot say: an
// entry is its habit and how much.
func (s *Server) title(t *schema.Type, rec *store.Record) string {
	return records.Name(s.app.Store, t, rec)
}

// aboutItem is a reminder's about on its page: the thing, as the way there.
func (s *Server) aboutItem(f schema.Field, val string) map[string]any {
	if t, rec, ok := s.aboutOf(val); ok {
		return map[string]any{"label": f.Display(), "value": s.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID, "prop": f.Name, "source": val}
	}
	return map[string]any{"label": f.Display(), "value": val, "prop": f.Name}
}

// nextThings is what a record's page offers to do with it, in one row
// under the title: a reminder about it, a word with the assistant about
// it, and its day on the calendar when it has one and there is a
// calendar to show it.
//
// None of them are there unless somebody asked for them. A labelled
// field and a button under every record, for the one time in twenty
// anyone wants a reminder, is a form the other nineteen read past — and
// a link to that form is a smaller version of the same thing, still
// there every time, still about the page rather than what is on it. Each
// is a part (remind, ask, day) that the address opens and the workspace
// keeps: see parts.go. Meanwhile the assistant can set a reminder, and
// is already the thing "ask the assistant" would have led to.
func (s *Server) nextThings(t *schema.Type, rec *store.Record, on []string) string {
	if t.Internal || t.Name == ReminderType || len(on) == 0 {
		return ""
	}
	path := "/t/" + t.Name + "/" + rec.ID
	title := s.title(t, rec)
	var b strings.Builder
	if _, ok := s.app.Types.Get(ReminderType); ok && has(on, RemindPart) {
		b.WriteString(string(s.form(ui.Form{Action: "/clock/set", Class: "sw-next__remind", Hidden: ui.Hidden("about", path, "title", title),
			Body:   s.part(ui.TextField{Label: "Remind me at", Name: "at", ID: "remind-at", Hint: "7pm, tomorrow 9am", Autocomplete: "off"}),
			Button: &ui.Button{Label: "Remind me", Context: "about " + title, Variant: ui.Secondary}})))
	}
	if has(on, AskPart) {
		b.WriteString(string(s.part(ui.Link{Href: "/chat?about=" + path, Label: "Ask the assistant", Look: ui.LookButton})))
	}
	if day := s.dayOf(t, rec); day != "" && has(on, DayPart) {
		if cal := s.firstBlock("calendar"); cal != nil {
			b.WriteString(string(s.part(ui.Link{Href: "/canvas/" + cal.ID + "?day=" + day, Label: "See that day", Look: ui.LookButton})))
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return `<div class="sw-next" role="group" aria-label="Do with this">` + b.String() + `</div>`
}

// dayOf is the day a record falls on, from its first date field.
func (s *Server) dayOf(t *schema.Type, rec *store.Record) string {
	field := t.DayField()
	if field == "" {
		return ""
	}
	v, _ := rec.Fields[field].(string)
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return ts.UTC().Format("2006-01-02")
	}
	return ts.Local().Format("2006-01-02")
}

// firstBlock is the first block on the canvas of a component, or nil.
func (s *Server) firstBlock(component string) *store.Record {
	for _, blk := range s.canvasBlocks() {
		if blk.Fields["component"] == component {
			return blk
		}
	}
	return nil
}

// nudges is the clock keeping a habit: one with a remind time that is
// past for today and not yet met rings once, as a reminder about it,
// that rings and reads like any other.
func (s *Server) nudges(now time.Time) []*store.Record {
	rt, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return nil
	}
	habits, _ := s.app.Store.List(HabitType, store.ListOptions{})
	var rang []*store.Record
	for _, hrec := range habits {
		remind, _ := hrec.Fields["remind"].(string)
		if archived, _ := hrec.Fields["archived"].(bool); archived || strings.TrimSpace(remind) == "" {
			continue
		}
		at, dayOnly, ok := when.Parse(remind, now)
		if !ok || dayOnly || at.After(now) || at.Before(track.PeriodStart(now, "day")) {
			continue
		}
		h := track.Normal(blocks.HabitOf(hrec))
		if h.Aim != track.Reach {
			// A reminder is for something still to do; a limit or a
			// record has nothing to do by a time.
			continue
		}
		sum := track.Summarise(h, blocks.EntriesOf(s.app.Store, h.ID), now, 1)
		about := "/t/" + HabitType + "/" + h.ID
		if sum.Met || s.remindedToday(rt, about, now) {
			continue
		}
		fields := map[string]any{"title": h.Name, "at": when.Store(now, false), "kind": "alarm", "state": "rang", "about": about}
		made, err := records.ApplyOps(s.app.Store, records.Op{Type: ReminderType, After: fields})
		if err != nil {
			continue
		}
		rec, err := s.app.Store.Get(ReminderType, made[0].ID)
		if err != nil {
			continue
		}
		records.Record(s.app.Store, "system", records.Change{Action: "rang", Component: ReminderType, ID: rec.ID, Detail: h.Name + ": " + track.Progress(h, sum) + " so far", Href: about})
		rang = append(rang, rec)
	}
	return rang
}

// remindedToday says whether a reminder about something already rang today.
func (s *Server) remindedToday(rt *schema.Type, about string, now time.Time) bool {
	recs, err := query.Filter(s.app.Store, rt, []string{"about=" + about}, "at", 0, now)
	if err != nil {
		return false
	}
	today := now.Local().Format("2006-01-02")
	for _, rec := range recs {
		at, _ := rec.Fields["at"].(string)
		if ts, err := time.Parse(time.RFC3339, at); err == nil && ts.Local().Format("2006-01-02") == today {
			return true
		}
	}
	return false
}
