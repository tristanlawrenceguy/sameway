package server

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// A conversation's status line and its first words: what it says when a
// turn ends, and what it offers to ask before anything has been said.

// status summarises the last turn for the live region.
func (s *Server) status(msgs []*store.Record) template.HTML {
	props := map[string]any{"id": "chat-status", "message": "Ready.", "state": "idle"}
	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		switch last.Fields["role"] {
		case "error":
			// Where the failure is said does not depend on where the person
			// looks: the reason is read out with it, as a reply's words are.
			props["state"], props["message"] = "error", "The last request failed."
			if words, _ := last.Fields["content"].(string); strings.TrimSpace(words) != "" {
				props["said"] = trim.Flat(chat.SanitizeError(words), 200)
			}
		case "assistant":
			n := 0
			if changes, ok := last.Fields["changes"].([]any); ok {
				n = len(changes)
			}
			props["state"] = "done"
			switch n {
			case 0:
				props["message"] = "Assistant replied."
			case 1:
				props["message"] = "Assistant replied and made 1 change."
			default:
				props["message"] = fmt.Sprintf("Assistant replied and made %d changes.", n)
			}
			// The reply's first words, read out but not drawn, so a person who
			// cannot see it arrive hears what it says; the chip stays short.
			// Its links are said as their names, as the reply shows them,
			// not spelled out as brackets and addresses.
			if words, _ := last.Fields["content"].(string); strings.TrimSpace(words) != "" {
				props["said"] = trim.Flat(render.LinkWords(words, s.linkTitle), 200)
			}
		}
	}
	return s.component("status", props)
}

// chatStarts are a few things to ask, for a chat with nothing in it: a
// blank box asks a person to know already what the assistant can do. Each
// is a link that puts the words in the box, where they are read and sent,
// or changed first; it works without a script.
func chatStarts(from string) string {
	starts := []string{"Add a task for tomorrow", "Keep a habit: 8 glasses of water a day", "Make a list of what I need to buy", "What can you do here?"}
	var b strings.Builder
	b.WriteString(`<ul class="sw-plain sw-chat__starts" aria-label="Things to ask">`)
	for _, w := range starts {
		fmt.Fprintf(&b, `<li><a class="sw-link sw-link--button sw-chat__start" href="%s?prompt=%s">%s</a></li>`, template.HTMLEscapeString(from), url.QueryEscape(w), template.HTMLEscapeString(w))
	}
	b.WriteString(`</ul>`)
	return b.String()
}
