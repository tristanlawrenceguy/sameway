package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// What an action suggests (chat/suggest_records.go) waits on Today: a
// task from an email, an event from a booking. Make it, change it, or say
// no, each a press; for an email or a shared message waiting to be sorted,
// the answer sorts it too. What the person does is what the action follows
// next time.

// suggestedFrom is the suggestion waiting from a record, if any.
func (s *Server) suggestedFrom(rec *store.Record) *store.Record {
	for _, p := range s.app.Chat.Suggestions() {
		if p.Fields["from"] == rec.Type+"/"+rec.ID {
			return p
		}
	}
	return nil
}

// suggestionPresses are a suggestion's three answers.
func (s *Server) suggestionPresses(p *store.Record, context string) string {
	var b strings.Builder
	b.WriteString(`<div class="sw-cluster">`)
	for _, f := range []struct {
		action, label string
		variant       ui.Variant
	}{{"/suggested/yes", "Make it", ui.Secondary}, {"/suggested/change", "Change", ui.Quiet}, {"/suggested/no", "No", ui.Quiet}} {
		b.WriteString(string(s.form(ui.Form{Action: f.action, Hidden: ui.Hidden("id", p.ID), Button: &ui.Button{Label: f.label, Context: context, Variant: f.variant}})))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// suggestionWords is what a suggestion asks, and why.
func suggestionWords(p *store.Record) string {
	summary, _ := p.Fields["summary"].(string)
	why, _ := p.Fields["detail"].(string)
	if why = strings.TrimSpace(strings.TrimSuffix(why, ".")); why != "" {
		summary += " " + strings.ToUpper(why[:1]) + why[1:] + "."
	}
	return summary
}

// suggestedSection is Suggested for you on Today: what actions suggested
// from records not waiting to be sorted (those show beside their record).
func (s *Server) suggestedSection(sorting map[string]bool) string {
	esc := template.HTMLEscapeString
	var b strings.Builder
	for _, p := range s.app.Chat.Suggestions() {
		if sorting[fmt.Sprint(p.Fields["from"])] {
			continue
		}
		b.WriteString(`<li class="sw-stack"><p>` + esc(suggestionWords(p)) + `</p>` + s.suggestionPresses(p, fmt.Sprint(p.Fields["summary"])) + `</li>`)
	}
	if b.Len() == 0 {
		return ""
	}
	return `<h2>Suggested for you</h2><ul class="sw-plain sw-rows">` + b.String() + `</ul>`
}

// answered is the suggestion a press is about, still waiting.
func (s *Server) answered(r *http.Request) (*store.Record, error) {
	r.ParseForm()
	for _, p := range s.app.Chat.Suggestions() {
		if p.ID == r.PostForm.Get("id") {
			return p, nil
		}
	}
	return nil, errors.New("that suggestion has been answered already")
}

// sortSource takes the to sort tag off what a suggestion came from.
func (s *Server) sortSource(r *http.Request, p *store.Record) {
	typ, id, _ := strings.Cut(fmt.Sprint(p.Fields["from"]), "/")
	if rec, err := s.app.Store.Get(typ, id); err == nil && taggedWith(rec, toSort) {
		s.sortedNote(r, rec.ID) // mail_sort.go
	}
}

func (s *Server) suggestedYes(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Accept(p.ID)
	}
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	s.sortSource(r, p)
	s.tagsAgreed(p) // classify_today.go
	s.tellAt(w, r, outcome{Title: "Made", Text: "Made, as suggested."}, "/today")
}

func (s *Server) suggestedChange(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Accept(p.ID)
	}
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	s.sortSource(r, p)
	s.tagsAgreed(p)
	done, _ := s.app.Store.Get(records.ProposalType, p.ID)
	to := "/today"
	if made, _ := done.Fields["made"].(string); made != "" {
		to = "/t/" + made + "#edit"
	}
	s.tellAt(w, r, outcome{Title: "Made", Text: "Made; change what you want and save. The action learns from what you change."}, to)
}

func (s *Server) suggestedNo(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Dismiss(p.ID)
	}
	if err != nil {
		s.failed(w, r, "Not changed", err, "/today")
		return
	}
	s.sortSource(r, p)
	s.tellAt(w, r, outcome{Title: "Turned down", Text: "Nothing was made. The action learns from that too."}, "/today")
}

// writeShare makes the note a share becomes. A message or a photo, not a
// page to read, may ask something, so it waits to be sorted.
func (s *Server) writeShare(r *http.Request, title, body string, toBeSorted bool) (*store.Record, string, error) {
	fields := map[string]any{"title": title, "body": body}
	if toBeSorted {
		fields["tags"] = []any{"shared", toSort}
	}
	return records.WriteAs(s.app.Store, s.who(r), "created", shareType, "", fields)
}
