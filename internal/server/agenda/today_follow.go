package agenda

import (
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A call ends with "I will get back to you on Thursday", and the
// interaction keeps the day as its follow-up; a client's project has the
// day it must be finished. Both were on the calendar and nowhere a person
// starts the day, so Thursday came and went. Today now gathers them too:
// a follow-up due today under Follow up, a project due today under
// Deadlines, and either one past its day with what is overdue. A
// follow-up is dealt with in one press, Followed up, which clears its day
// and is taken back by Undo; a project leaves Today when it is done.

// keeper says which of Today's lists a record with a day goes in, or nil.
type keeper func(r *store.Record, at time.Time, isToday bool) *[]todayItem

// todayMore adds projects due and follow-ups to the day's lists.
func (s *Service) todayMore(out *todayLists, now time.Time, add func(typ, field string, keep keeper)) {
	dueOrLate := func(today *[]todayItem) keeper {
		return func(r *store.Record, at time.Time, isToday bool) *[]todayItem {
			switch {
			case isToday:
				return today
			case at.Before(now):
				return &out.Late
			}
			return nil
		}
	}
	projects := dueOrLate(&out.Deadlines)
	add("project", "due", func(r *store.Record, at time.Time, isToday bool) *[]todayItem {
		if st, _ := r.Fields["status"].(string); st == "done" {
			return nil
		}
		return projects(r, at, isToday)
	})
	add("interaction", "follow_up", dueOrLate(&out.FollowUps))
}

// followPress is a follow-up's one press on Today.
func (s *Service) followPress(it todayItem) string {
	if it.Type != "interaction" {
		return ""
	}
	return `<div class="sw-cluster">` + string(s.Form(ui.Form{Action: "/today/followed-up", Hidden: ui.Hidden("id", it.ID),
		Button: &ui.Button{Label: "Followed up", Context: it.Title, Variant: ui.Secondary}})) + `</div>`
}

// todayFollowed clears a follow-up's day: the person got back to them.
func (s *Service) todayFollowed(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.PostForm.Get("id")
	rec, err := s.app.Store.Get("interaction", id)
	if err != nil {
		s.Failed(w, r, "Not changed", err, "/today")
		return
	}
	title, _ := rec.Fields["summary"].(string)
	_, act, err := records.WriteAs(s.app.Store, s.Who(r), "updated", "interaction", id, map[string]any{"follow_up": nil})
	if err != nil {
		s.Failed(w, r, "Not changed", err, "/today")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: "Followed up", Text: title + " has no follow-up left.", Undo: act, Of: title}, "/today")
}
