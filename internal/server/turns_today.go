package server

import (
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Your turn to reply, on Today: conversations whose latest email asked you
// something (records/turn.go), and those waiting on the other side folded
// away as a count, with how long each has waited.

func (s *Server) turnSection() string {
	if _, ok := s.app.Types.Get("email"); !ok {
		return ""
	}
	emails, err := s.app.Store.List("email", store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return ""
	}
	esc := template.HTMLEscapeString
	var yours, theirs strings.Builder
	waiting := 0
	for _, e := range emails {
		line := `<li><a class="sw-link" href="/t/email/` + e.ID + `">` + esc(s.nameOf(e)) + `</a>`
		switch e.Fields["turn"] {
		case "yours":
			from, _ := e.Fields["from"].(string)
			body, _ := e.Fields["body"].(string)
			yours.WriteString(line + ` <span class="sw-muted sw-small">` + esc(from) + `: ` + esc(clipRunes(strings.Join(strings.Fields(body), " "), 120)) + `</span></li>`)
		case "theirs":
			waiting++
			theirs.WriteString(line + ` <span class="sw-muted sw-small">waiting ` + esc(waitedFor(time.Since(e.CreatedAt))) + `</span></li>`)
		}
	}
	var b strings.Builder
	if yours.Len() > 0 {
		b.WriteString(`<h2>Your turn to reply</h2><ul class="sw-plain sw-rows">` + yours.String() + `</ul>`)
	}
	if waiting > 0 {
		if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "Waiting on them", "count": waiting, "of": "conversation"}, template.HTML(`<ul class="sw-plain sw-rows">`+theirs.String()+`</ul>`)); err == nil {
			b.WriteString(string(body))
		}
	}
	return b.String()
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
