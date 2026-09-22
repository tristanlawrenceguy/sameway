package server

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
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
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(path), "/"), "/")
	if len(parts) != 3 || parts[0] != "t" {
		return nil, nil, false
	}
	t, ok := s.app.Types.Get(parts[1])
	if !ok {
		return nil, nil, false
	}
	rec, err := s.app.Store.Get(t.Name, parts[2])
	if err != nil {
		return nil, nil, false
	}
	return t, rec, true
}

// title is a record's title, with what the type alone cannot say: an
// entry is its habit and how much.
func (s *Server) title(t *schema.Type, rec *store.Record) string {
	if t.Name == EntryType {
		if ht, h, ok := s.aboutOf("/t/" + HabitType + "/" + str(rec.Fields["habit"], "")); ok {
			amount := number(rec.Fields["amount"])
			if amount == 0 && rec.Fields["amount"] == nil {
				amount = 1
			}
			unit, _ := h.Fields["unit"].(string)
			how := track.Amount(amount, unit)
			if unit == "" && amount == 1 {
				how = "done"
			}
			return s.title(ht, h) + ": " + how
		}
	}
	return titleOf(t, rec)
}

// aboutCell is a reminder's about on its page: the thing, as the way there.
func (s *Server) aboutCell(f schema.Field, val string) string {
	if t, rec, ok := s.aboutOf(val); ok {
		return fmt.Sprintf(`<dd data-prop="%s" data-source="%s"><a class="sw-link" href="/t/%s/%s">%s</a></dd>`, f.Name, template.HTMLEscapeString(val), t.Name, rec.ID, template.HTMLEscapeString(s.title(t, rec)))
	}
	return fmt.Sprintf(`<dd data-prop="%s">%s</dd>`, f.Name, template.HTMLEscapeString(val))
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
		fmt.Fprintf(&b, `<form method="post" action="/clock/set" class="sw-next__remind"><input type="hidden" name="about" value="%s"><input type="hidden" name="title" value="%s">%s%s</form>`,
			template.HTMLEscapeString(path), template.HTMLEscapeString(title),
			s.component("text-field", map[string]any{"label": "Remind me at", "name": "at", "id": "remind-at", "hint": "7pm, tomorrow 9am", "autocomplete": "off"}),
			s.component("button", map[string]any{"label": "Remind me", "context": "about " + title, "type": "submit", "variant": "secondary"}))
	}
	if has(on, AskPart) {
		b.WriteString(string(s.component("link", map[string]any{"href": "/chat?about=" + path, "label": "Ask the assistant", "look": "button"})))
	}
	if day := s.dayOf(t, rec); day != "" && has(on, DayPart) {
		if cal := s.firstBlock("calendar"); cal != nil {
			b.WriteString(string(s.component("link", map[string]any{"href": "/canvas/" + cal.ID + "?day=" + day, "label": "See that day", "look": "button"})))
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return `<div class="sw-next" role="group" aria-label="Do with this">` + b.String() + `</div>`
}

// dayOf is the day a record falls on, from its first date field.
func (s *Server) dayOf(t *schema.Type, rec *store.Record) string {
	field := dateField(t, nil)
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

// ringWords is what a ring says beyond the page and where it leads: the
// thing it is about, and for a habit where it stands; the reminder
// itself when it is about nothing.
func (s *Server) ringWords(rec *store.Record) (text, url string) {
	url = "/t/" + ReminderType + "/" + rec.ID
	text = "It is time."
	about, _ := rec.Fields["about"].(string)
	t, target, ok := s.aboutOf(about)
	if !ok {
		return text, url
	}
	url = about
	text = s.title(t, target)
	if t.Name == HabitType {
		h := habitOf(target)
		sum := track.Summarise(h, s.entriesOf(h.ID), time.Now(), 1)
		text = h.Name + ": " + track.Progress(sum, h.Unit) + " so far"
	}
	return text, url
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
		h := habitOf(hrec)
		sum := track.Summarise(h, s.entriesOf(h.ID), now, 1)
		about := "/t/" + HabitType + "/" + h.ID
		if sum.Met || s.remindedToday(rt, about, now) {
			continue
		}
		fields := map[string]any{"title": h.Name, "at": when.Store(now, false), "kind": "alarm", "state": "rang", "about": about}
		rec, err := s.app.Store.Create(ReminderType, fields)
		if err != nil {
			continue
		}
		chat.Record(s.app.Store, "system", chat.Change{Action: "rang", Component: ReminderType, ID: rec.ID, Detail: h.Name + ": " + track.Progress(sum, h.Unit) + " so far", Href: about})
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

// everyEvent is every listed type's records on their days, for a
// calendar that shows all of it: each event says what kind it is.
func (s *Server) everyEvent(now time.Time) []any {
	events := []any{}
	for _, t := range s.app.Types.Types {
		if t.Internal || !s.listed(t) {
			continue
		}
		field := dateField(t, nil)
		if field == "" {
			continue
		}
		recs, err := query.Filter(s.app.Store, t, nil, field, 0, now)
		if err != nil {
			continue
		}
		for _, rec := range recs {
			if ev := s.eventOf(t, rec, field); ev != nil {
				ev["meta"] = t.Name
				events = append(events, ev)
			}
		}
	}
	return events
}
