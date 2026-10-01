package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The changes suggested to a record's words wait on its page, above its
// fields, for whoever may change it: each in place, with Accept and
// Decline (chat/suggest.go). Someone who may only look sees none of them.

// suggestionsOn is the section of what waits on a record.
func (s *Server) suggestionsOn(r *http.Request, t *schema.Type, rec *store.Record) string {
	waiting := chat.Suggestions(s.app.Store, t.Name, rec.ID)
	if len(waiting) == 0 || !changes(r) {
		return ""
	}
	from := "/t/" + t.Name + "/" + rec.ID
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="suggested"><h2 id="suggested">Suggested changes</h2>`)
	b.WriteString(`<p class="sw-muted">Nothing changes until you accept one. Each accepted change can be undone.</p>`)
	for _, sg := range waiting {
		text, _ := rec.Fields[str(sg.Fields["field"], "")].(string)
		passage := str(sg.Fields["passage"], "")
		props := map[string]any{"id": "suggestion-" + sg.ID, "why": str(sg.Fields["why"], ""), "passage": passage,
			"replacement": str(sg.Fields["replacement"], ""), "from": from, "actor": s.actorOf(sg),
			"accept": "/suggestions/" + sg.ID + "/accept", "decline": "/suggestions/" + sg.ID + "/decline"}
		if at := strings.Index(text, passage); at < 0 || strings.Count(text, passage) != 1 {
			props["outdated"] = true
		} else {
			props["before"], props["after"] = around(text[:at], true), around(text[at+len(passage):], false)
		}
		b.WriteString(string(s.component("suggestion", props)))
	}
	b.WriteString(`</section>`)
	return b.String()
}

// actorOf is who suggested it, for its colour: an agent, or the assistant.
func (s *Server) actorOf(sg *store.Record) string {
	if w := s.app.Chat.Writers().Of(chat.SuggestionType, sg); strings.Contains(w.Words, "agent") {
		return "agent"
	}
	return "assistant"
}

// around is up to a few words of what comes before or after a passage,
// within its paragraph, so it reads in its sentence.
func around(text string, before bool) string {
	const most = 48
	if before {
		if i := strings.LastIndex(text, "\n"); i >= 0 {
			text = text[i+1:]
		}
		if r := []rune(text); len(r) > most {
			text = string(r[len(r)-most:])
			if i := strings.Index(text, " "); i >= 0 {
				text = text[i:]
			}
		}
		return text
	}
	if i := strings.Index(text, "\n"); i >= 0 {
		text = text[:i]
	}
	if r := []rune(text); len(r) > most {
		text = string(r[:most])
		if i := strings.LastIndex(text, " "); i >= 0 {
			text = text[:i+1]
		}
	}
	return text
}

// suggestionAnswer accepts or declines one suggestion.
func (s *Server) suggestionAnswer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sg, err := s.app.Store.Get(chat.SuggestionType, id)
	if err != nil {
		s.tell(w, r, outcome{Failed: true, Title: "No such suggestion", Text: "It may have been answered already."}, "/")
		return
	}
	about := str(sg.Fields["about"], "")
	typ, _, _ := strings.Cut(about, "/")
	back := "/t/" + about
	if t, ok := s.app.Types.Get(typ); ok && t.Owners && !chat.VisitorOf(r.Context()).Owner() {
		s.tell(w, r, outcome{Failed: true, Title: "Not yours to change"}, "/")
		return
	}
	if r.PathValue("answer") == "decline" {
		if err := chat.DeclineSuggestion(s.app.Store, id); err != nil {
			s.tell(w, r, outcome{Failed: true, Title: "Not declined", Text: err.Error()}, back)
			return
		}
		s.tell(w, r, outcome{Title: "Declined", Text: "The suggestion is set aside; nothing changed."}, back)
		return
	}
	rec, entry, err := chat.AcceptSuggestion(s.app.Store, s.who(r), id)
	switch {
	case errors.Is(err, chat.ErrOutdated):
		s.tell(w, r, outcome{Failed: true, Title: "Not changed", Text: "The words it would change have changed since, so it no longer fits; it is set aside."}, back)
	case err != nil:
		s.tell(w, r, outcome{Failed: true, Title: "Not changed", Text: err.Error()}, back)
	default:
		t, _ := s.app.Types.Get(typ)
		s.tell(w, r, outcome{Title: "Changed " + s.title(t, rec), Text: "The suggestion is in.", Undo: entry, Of: s.title(t, rec)}, back)
	}
}
