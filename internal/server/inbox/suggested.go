package inbox

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// What an action suggests (automate/suggest.go) waits on Today: a
// task from an email, an event from a booking. Make it, change it, or say
// no, each a press; for an email or a shared message waiting to be sorted,
// the answer sorts it too. What the person does is what the action follows
// next time.

// suggestedFrom is the suggestion waiting from a record, if any.
func suggestedFrom(list []chat.Waiting, rec *store.Record) *chat.Waiting {
	for _, w := range chat.WaitingOf(list, chat.WaitSuggestion) {
		if w.About == rec.Type+"/"+rec.ID {
			return &w
		}
	}
	return nil
}

// presses are the answers to what waits, side by side, the first the one
// most likely.
func (s *Service) presses(ps []chat.Press, context string) string {
	var b strings.Builder
	b.WriteString(`<div class="sw-cluster">`)
	for i, p := range ps {
		variant := ui.Quiet
		if i == 0 {
			variant = ui.Secondary
		}
		b.WriteString(string(s.Form(ui.Form{Action: p.Post, Hidden: ui.Hidden("id", p.ID), Button: &ui.Button{Label: p.Label, Context: context, Variant: variant}})))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// suggestedSection is Suggested for you on Today: what actions suggested
// from records not waiting to be sorted (those show beside their record).
func (s *Service) suggestedSection(list []chat.Waiting, sorting map[string]bool) string {
	esc := template.HTMLEscapeString
	var b strings.Builder
	for _, w := range chat.WaitingOf(list, chat.WaitSuggestion) {
		if sorting[w.About] {
			continue
		}
		b.WriteString(`<li class="sw-stack"><p>` + esc(w.Words) + `</p>` + s.presses(w.Presses, fmt.Sprint(w.Record.Fields["summary"])) + `</li>`)
	}
	if b.Len() == 0 {
		return ""
	}
	return `<h2>Suggested for you</h2><ul class="sw-plain sw-rows">` + b.String() + `</ul>`
}

// answered is the suggestion a press is about, still waiting.
func (s *Service) answered(r *http.Request) (*store.Record, error) {
	r.ParseForm()
	for _, p := range s.app.Chat.Suggestions() {
		if p.ID == r.PostForm.Get("id") {
			return p, nil
		}
	}
	return nil, errors.New("that suggestion has been answered already")
}

// sortSource takes the to sort tag off what a suggestion came from.
func (s *Service) sortSource(r *http.Request, p *store.Record) {
	typ, id, _ := strings.Cut(fmt.Sprint(p.Fields["from"]), "/")
	if rec, err := s.app.Store.Get(typ, id); err == nil && chat.Tagged(rec, chat.ToSort) {
		s.sortedNote(r, rec.ID) // mail_sort.go
	}
}

func (s *Service) suggestedYes(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Accept(p.ID)
	}
	if err != nil {
		s.Failed(w, r, "Not made", err, "/today")
		return
	}
	s.sortSource(r, p)
	s.tagsAgreed(p) // classify_today.go
	s.TellAt(w, r, web.Outcome{Title: "Made", Text: "Made, as suggested."}, "/today")
}

func (s *Service) suggestedChange(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Accept(p.ID)
	}
	if err != nil {
		s.Failed(w, r, "Not made", err, "/today")
		return
	}
	s.sortSource(r, p)
	s.tagsAgreed(p)
	done, _ := s.app.Store.Get(records.ProposalType, p.ID)
	to := "/today"
	if made, _ := done.Fields["made"].(string); made != "" {
		to = "/t/" + made + "#edit"
	}
	s.TellAt(w, r, web.Outcome{Title: "Made", Text: "Made; change what you want and save. The action learns from what you change."}, to)
}

func (s *Service) suggestedNo(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Dismiss(p.ID)
	}
	if err != nil {
		s.Failed(w, r, "Not changed", err, "/today")
		return
	}
	s.sortSource(r, p)
	s.TellAt(w, r, web.Outcome{Title: "Turned down", Text: "Nothing was made. The action learns from that too."}, "/today")
}
