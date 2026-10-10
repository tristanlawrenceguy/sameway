package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An email's conversation, on its page when asked (?show=conversation).
// Your turn to reply on Today opens it, because answering an email needs
// what came before it; at rest an email's page is that email alone.
//
// It reads the way Gmail and Apple Mail read a conversation: oldest first,
// each email under a heading of who wrote it and when, so a screen reader
// moves from one to the next by heading. Each says only its own words:
// what a reply quotes is already above it, and the email this page is
// about is said once, at the top, and only named here.

// ConversationPart is an email's whole conversation, in order.
const ConversationPart = "conversation"

func (s *Server) conversationOn(r *http.Request, t *schema.Type, rec *store.Record) string {
	if t.Name != records.EmailType || !s.showing(r, ConversationPart) {
		return ""
	}
	emails := records.Conversation(s.app.Store, rec)
	if len(emails) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<section class="sw-stack" id="conversation" aria-labelledby="conversation-title"><h2 id="conversation-title">The conversation, %s</h2><ol class="sw-plain sw-stack">`, schema.Count(len(emails), "email"))
	for _, e := range emails {
		b.WriteString(`<li><article class="sw-stack--tight" aria-labelledby="said-` + e.ID + `"><h3 id="said-` + e.ID + `">`)
		head := template.HTMLEscapeString(s.whoWrote(e) + ", " + s.sentAt(t, e))
		if e.ID == rec.ID {
			b.WriteString(head + `, this email</h3><p class="sw-muted">Its words are at the top of this page.</p></article></li>`)
			continue
		}
		body, _ := e.Fields["body"].(string)
		b.WriteString(`<a class="sw-link" href="/t/email/` + e.ID + `">` + head + `</a></h3><div class="sw-prose">` + string(prose.Render(records.OwnWords(body), 4)) + `</div></article></li>`)
	}
	_, here := s.shown(r)
	b.WriteString(`</ol>` + s.fewer("/t/email/"+rec.ID, ConversationPart, "the conversation", here) + `</section>`)
	return b.String()
}

// whoWrote is who sent an email, by name: You for the person's own.
func (s *Server) whoWrote(e *store.Record) string {
	if mine, _ := e.Fields["from_me"].(bool); mine {
		return "You"
	}
	from, _ := e.Fields["from"].(string)
	if name, _, ok := strings.Cut(from, " <"); ok && strings.TrimSpace(name) != "" {
		return strings.Trim(strings.TrimSpace(name), `"`)
	}
	if from == "" {
		return "Someone"
	}
	return from
}

// sentAt is when an email was sent, in words.
func (s *Server) sentAt(t *schema.Type, e *store.Record) string {
	if f, ok := t.Field("received"); ok && e.Fields["received"] != nil {
		return s.display(*f, e.Fields["received"])
	}
	return e.CreatedAt.Local().Format("Mon 2 Jan 15:04")
}
