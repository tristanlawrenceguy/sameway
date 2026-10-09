package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A piece of writing offers the kinds of help an editor gives, named as a
// writer would, and the writer picks: a check of spelling and typos, a
// tightening of the sentences, or feedback on how it is put together.
// Each asks the assistant for that and nothing more, and what it suggests
// waits for them (suggestions.go); feedback on structure comes back in
// words, as an editor's would, not as changes. Help comes when asked: it
// is a part of the page (parts.go), off until the assistant sees a reason
// and hands over ?show=writing-help, or the person keeps it on.

// writingHelpWords is how long a piece is before help is offered.
const writingHelpWords = 30

func (s *Server) writingHelp(r *http.Request, t *schema.Type, rec *store.Record) string {
	if !changes(r) || t.Internal || !s.showing(r, WritingHelpPart) {
		return ""
	}
	field := ""
	for _, f := range t.Fields {
		if f.Type == "markdown" && !f.ReadOnly {
			field = f.Name
			break
		}
	}
	text, _ := rec.Fields[field].(string)
	if field == "" || len(strings.Fields(text)) < writingHelpWords {
		return ""
	}
	what := s.title(t, rec) + " (/t/" + t.Name + "/" + rec.ID + ")"
	asks := []struct{ label, ask string }{
		{"Check spelling and typos", "Check the spelling, grammar and typos in " + what + " and suggest fixes. Change nothing else."},
		{"Make it clearer and tighter", "Suggest how to make " + what + " clearer and tighter, keeping my voice: small changes where they matter, each with why."},
		{"Feedback on structure", "Give me feedback on how " + what + " is put together: what to move, cut or add, and why. Tell me; suggest no changes."},
	}
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="writing-help"><h2 id="writing-help">Help with the writing</h2><p class="sw-muted">The assistant suggests; nothing changes until you accept.</p><p class="sw-cluster">`)
	for _, a := range asks {
		b.WriteString(string(s.part(ui.Link{Href: "/chat?prompt=" + url.QueryEscape(a.ask), Label: a.label, Look: ui.LookButton})))
		b.WriteString(" ")
	}
	b.WriteString(`</p>`)
	_, here := s.shown(r)
	b.WriteString(s.fewer("/t/"+t.Name+"/"+rec.ID, WritingHelpPart, "help with the writing", here))
	b.WriteString(`</section>`)
	return b.String()
}
