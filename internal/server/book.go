package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The booking link: anyone, a person or their AI agent, asks for a time
// with the owner at /book, from the hours their events leave free, shown
// as times and never as what they are. It is on when the owner turns
// publish.book on, which is put to them first, and then the internet
// reaches it through Tailscale Funnel like anything published.
//
// The one safeguard is that a request does nothing: it waits on Today as
// a request for an event, each on its own, until the owner presses Make
// it, Change or No, and its address tells the one asking what became of
// it (chat/booking.go).

const (
	bookPath = "/book"
	bookGet  = "GET /book"
	bookPost = "POST /book"
	// bookDays is how far ahead times are offered, and the hours of the day
	// they start at, in the owner's zone.
	bookDays  = 14
	bookFirst = 9
	bookLast  = 17
)

// bookable says whether the booking link is on, for the one asking: the
// owner and the people they let in may look at it whenever it is on.
func (s *Server) bookable() bool { return s.Published().Book }

// openHours are the hours the owner's events leave free, each starting on
// the hour, from the next one, for the next two weeks. The hour is the
// owner's: Truncate counts from UTC, which is half past in India.
func (s *Server) openHours() []time.Time {
	now := s.now()
	start := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, bookDays)
	busy := s.app.Records.Busy(start, end)
	var out []time.Time
	for t := start; t.Before(end); t = t.Add(time.Hour) {
		if h := t.Hour(); h < bookFirst || h >= bookLast {
			continue
		}
		taken := false
		for _, b := range busy {
			if t.Before(b.To) && t.Add(time.Hour).After(b.From) {
				taken = true
				break
			}
		}
		if !taken {
			out = append(out, t)
		}
	}
	return out
}

func (s *Server) bookPage(w http.ResponseWriter, r *http.Request) {
	if !s.bookable() {
		http.NotFound(w, r)
		return
	}
	hours := s.openHours()
	if wantsJSON(r) {
		times := make([]string, len(hours))
		for i, t := range hours {
			times[i] = t.UTC().Format(time.RFC3339)
		}
		writeJSON(w, http.StatusOK, map[string]any{"hours": times, "each": "one hour",
			"ask":  map[string]any{"post": bookPath, "fields": map[string]string{"at": "one of hours", "name": "your name", "email": "how to reach you", "note": "optional, a few words"}},
			"then": "the request waits for " + s.app.Workspace.Config.Name + "'s yes; nothing is booked until then, and what you are sent back has the address that says what became of it"})
		return
	}
	s.page(w, r, "Find a time", s.bookForm(hours, "", nil), pageOptions{})
}

// bookForm is the page's form: the free hours to pick from, and who is
// asking. Each hour is said in full, with its time zone where it is
// chosen, since whoever asks may be anywhere and a screen reader reads
// Thu as a word. After a problem it keeps what they typed.
func (s *Server) bookForm(hours []time.Time, problem string, typed url.Values) template.HTML {
	if len(hours) == 0 {
		return s.part(ui.Empty{Message: "There are no free hours in the next two weeks."})
	}
	var options []any
	for _, t := range hours {
		at := t.In(s.now().Location())
		options = append(options, map[string]any{"value": t.UTC().Format(time.RFC3339), "label": at.Format("Monday 2 January") + ", " + when.Clock(at, s.h24())})
	}
	intro := template.HTML(`<p>Pick an hour that is free, and say who you are. Your request waits for an answer: nothing is booked until then.</p>`)
	if problem != "" {
		intro = s.part(ui.Alert{Kind: ui.Problem, Message: problem}) + intro
	}
	return intro + s.form(ui.Form{Action: bookPath, Class: "sw-stack",
		Body: s.component("select", map[string]any{"label": "When", "name": "at", "as": "dropdown", "options": options, "value": typed.Get("at"), "required": true,
			"hint": "In " + when.Zone(hours[0], hours[len(hours)-1]) + ". Each is an hour."}) +
			s.part(ui.TextField{Label: "Your name", Name: "name", Value: typed.Get("name"), Required: true, MaxLength: 80, Autocomplete: "name"}) +
			s.part(ui.TextField{Label: "Your email", Name: "email", Value: typed.Get("email"), Type: ui.Email, Required: true, MaxLength: 120, Autocomplete: "email"}) +
			s.component("textarea", map[string]any{"label": "What it is about", "name": "note", "value": typed.Get("note"), "rows": 3, "maxlength": 500, "hint": "Optional."}),
		Button: &ui.Button{Label: "Ask for this time", Variant: ui.Primary}})
}

func (s *Server) bookAsk(w http.ResponseWriter, r *http.Request) {
	if !s.bookable() {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	at, err := time.Parse(time.RFC3339, r.PostForm.Get("at"))
	open := false
	for _, t := range s.openHours() {
		open = open || err == nil && t.Equal(at)
	}
	if !open {
		problem := "That hour is not free any more: pick another."
		if r.PostForm.Get("at") == "" {
			problem = "Pick a time."
		}
		s.page(w, r, "Find a time", s.bookForm(s.openHours(), problem, r.PostForm), pageOptions{Status: http.StatusUnprocessableEntity})
		return
	}
	token, err := s.app.Chat.Booking(r.PostForm.Get("name"), r.PostForm.Get("email"), r.PostForm.Get("note"), at)
	if err != nil {
		s.page(w, r, "Find a time", s.bookForm(s.openHours(), web.PlainError(err), r.PostForm), pageOptions{Status: http.StatusUnprocessableEntity})
		return
	}
	s.Changed()
	asked := []map[string]any{s.requestSeen(r, s.app.Chat.RequestByToken(token))} // request_page.go
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"requests": asked})
		return
	}
	s.page(w, r, "Asked", template.HTML(fmt.Sprintf(`<p>Your request has gone to %s. Nothing is booked until they say yes.</p>`, template.HTMLEscapeString(s.app.Workspace.Config.Name)))+s.askedLinks(asked), pageOptions{})
}
