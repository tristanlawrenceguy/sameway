package inbox

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Your turn to reply, on Today: conversations whose latest email asked you
// something (records/turn.go), and those waiting on the other side folded
// away as a count, with how long each has waited. Each opens on its whole
// conversation, since a reply needs what came before. One that needs no
// answer is taken off with a press, as HEY's Reply Later is marked done;
// the next email in it decides again.

func (s *Service) turnSection(list []chat.Waiting) string {
	esc := template.HTMLEscapeString
	var yours, theirs strings.Builder
	line := func(e *store.Record) string {
		return `<li class="sw-stack--tight"><a class="sw-link" href="/t/email/` + e.ID + `?show=` + ConversationPart + `#conversation">` + esc(s.nameOf(e)) + `</a>`
	}
	for _, w := range chat.WaitingOf(list, chat.WaitReply) {
		e := w.Record
		body, _ := e.Fields["body"].(string)
		yours.WriteString(line(e) + ` <span class="sw-muted sw-small">` + esc(s.whoWrote(e)) + `: ` + esc(clip(strings.Join(strings.Fields(records.OwnWords(body)), " "), 120)) + `</span>`)
		for _, p := range w.Presses {
			yours.WriteString(string(s.Form(ui.Form{Action: p.Post, Hidden: ui.Hidden("id", p.ID), Button: &ui.Button{Label: p.Label, Context: w.Words, Variant: ui.Quiet}})))
		}
		yours.WriteString(`</li>`)
	}
	// Waiting on them is not waiting for the person, so it is found here,
	// folded away as a count.
	waiting := 0
	if _, ok := s.app.Types.Get(records.EmailType); ok {
		emails, _ := s.app.Store.List(records.EmailType, store.ListOptions{OrderBy: "created_at"})
		for _, e := range emails {
			if e.Fields["turn"] == "theirs" {
				waiting++
				theirs.WriteString(line(e) + ` <span class="sw-muted sw-small">waiting ` + esc(waitedFor(s.Now().Sub(sentTime(e)))) + `</span></li>`)
			}
		}
	}
	var b strings.Builder
	if yours.Len() > 0 {
		b.WriteString(`<h2>Your turn to reply</h2><ul class="sw-plain sw-rows">` + yours.String() + `</ul>`)
	}
	if waiting > 0 {
		if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "Waiting on them", "count": waiting, "of": schema.Noun(waiting, "conversation")}, template.HTML(`<ul class="sw-plain sw-rows">`+theirs.String()+`</ul>`)); err == nil {
			b.WriteString(string(body))
		}
	}
	return b.String()
}

// sentTime is when an email was sent, or when it came in when it does not
// say: an email read from Sent a week late waited from when it was sent.
func sentTime(e *store.Record) time.Time {
	if v, _ := e.Fields["received"].(string); v != "" {
		if at, err := time.Parse(time.RFC3339, v); err == nil {
			return at
		}
	}
	return e.CreatedAt
}

// mailAnswered takes a conversation off Your turn to reply: the person
// answered another way, or it needs no answer. It is update_record of the
// email's turn, logged and undone like any change.
func (s *Service) mailAnswered(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	e, err := s.app.Store.Get(records.EmailType, r.PostForm.Get("id"))
	if err != nil || e.Fields["turn"] != "yours" {
		s.Failed(w, r, "Not changed", errors.New("that is not waiting on you any more"), "/today")
		return
	}
	_, act, err := records.WriteAs(s.app.Store, s.Who(r), "updated", records.EmailType, e.ID, map[string]any{"turn": ""})
	if err != nil {
		s.Failed(w, r, "Not changed", err, "/today")
		return
	}
	title := s.nameOf(e)
	s.TellAt(w, r, web.Outcome{Title: "No reply needed", Text: title + " is off your turn to reply. The next email in it decides again.", Undo: act, Of: title}, "/today")
}

// waitedFor is how long, in a word or two.
func waitedFor(d time.Duration) string {
	switch days := int(d.Hours() / 24); {
	case days < 1:
		return "since today"
	case days == 1:
		return "a day"
	default:
		return strconv.Itoa(days) + " days"
	}
}
