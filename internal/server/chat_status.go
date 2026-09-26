package server

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
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
			props["state"], props["message"], props["live"] = "error", "The last request failed.", "assertive"
			if words, _ := last.Fields["content"].(string); strings.TrimSpace(words) != "" {
				props["said"] = clipWords(words, 200)
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
				props["message"] = "Assistant replied and made 1 change to the canvas."
			default:
				props["message"] = fmt.Sprintf("Assistant replied and made %d changes to the canvas.", n)
			}
			// The reply's first words, read out but not drawn, so a person who
			// cannot see it arrive hears what it says; the chip stays short.
			if words, _ := last.Fields["content"].(string); strings.TrimSpace(words) != "" {
				props["said"] = clipWords(words, 200)
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

// clipWords is text on one line, cut at about n characters.
func clipWords(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// messageTime is when a message was sent, as its chat shows it: the time
// alone today, the day with it before, so an old chat reads true.
func messageTime(at time.Time) string {
	at, now := at.Local(), time.Now()
	if at.YearDay() == now.YearDay() && at.Year() == now.Year() {
		return at.Format("15:04")
	}
	if at.Year() == now.Year() {
		return at.Format("2 Jan 15:04")
	}
	return at.Format("2 Jan 2006 15:04")
}
