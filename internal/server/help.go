package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Help is one page, in the same place on every page (the footer), that
// says how Sameway works in plain words and lets a person make it easier
// to use without asking the assistant: when the assistant is down, or is
// the thing that is hard to use, help must still be there.

// comfort are the settings a person may change here, each with what its
// values are called on the page.
var comfort = []struct {
	key, heading string
	values       [][2]string
}{
	{"ui.text", "Size of the words", [][2]string{{"normal", "Normal"}, {"large", "Large"}, {"larger", "Larger"}}},
	{"ui.spacing", "Room between lines and words", [][2]string{{"normal", "Normal"}, {"wide", "Wide"}}},
	{"ui.pace", "How changes arrive on the page", [][2]string{{"calm", "Calmly"}, {"quick", "Quickly"}, {"still", "All at once"}}},
	{"ui.controls", "Buttons on each item", [][2]string{{"auto", "Hover"}, {"visible", "Always show"}}},
}

func (s *Server) helpPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-ask"><h2 id="help-ask">Asking the assistant</h2>
<p>Type what you want in the message box and press Enter; Shift and Enter starts a new line. The assistant makes your notes, tasks and pages, and answers questions about them. When it wants to do something that cannot be undone, such as sending something to a website, it asks you first.</p>
<p>Tell it what you need, in your own words: "I use a screen reader", "keep things simple", "larger text please". It remembers.</p></section>`)
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-hand"><h2 id="help-hand">Doing things yourself</h2>
<p>Every list has a button to add one, such as Add a note. On anything you made, Edit changes it where it is, and Save keeps the change. Escape or Cancel leaves it as it was.</p>
<p>After you do something, a message at the top of the page says what happened.</p></section>`)
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-undo"><h2 id="help-undo">Taking things back</h2>
<p>Almost everything can be undone: the message after a change has an Undo button, and <a class="sw-link" href="/activity">Activity</a> lists every change with its own Undo. A deleted workspace goes to the trash, and <a class="sw-link" href="/workspaces">Workspaces</a> can bring it back.</p></section>`)
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-keys"><h2 id="help-keys">Keyboard</h2><ul>
<li>Tab and Shift+Tab move between things; the first Tab on a page offers a way straight to the main part.</li>
<li>Enter sends a message; Shift+Enter starts a new line.</li>
<li>In the formatting buttons above a text, the arrow keys move between them.</li>
<li>Escape leaves an edit without saving.</li></ul></section>`)
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-comfort"><h2 id="help-comfort">Making Sameway easier to use</h2>`)
	for _, c := range comfort {
		now := s.app.Workspace.Get(c.key)
		b.WriteString(`<div class="sw-stack--tight"><h3>` + template.HTMLEscapeString(c.heading) + `</h3><ul class="sw-plain sw-cluster">`)
		for _, v := range c.values {
			current := now == v[0] || (now == "" && v[0] == c.values[0][0])
			b.WriteString(`<li><form method="post" action="/help/set"><input type="hidden" name="key" value="` + c.key + `"><input type="hidden" name="value" value="` + v[0] + `">`)
			// Each says which setting it is for, so two called Normal are
			// told apart by a screen reader (WCAG 2.4.6).
			props := map[string]any{"label": v[1], "type": "submit", "variant": "secondary", "context": ", " + strings.ToLower(c.heading)}
			if current {
				props["variant"], props["context"] = "primary", ", "+strings.ToLower(c.heading)+", chosen"
			}
			b.WriteString(string(s.component("button", props)) + `</form></li>`)
		}
		b.WriteString(`</ul></div>`)
	}
	b.WriteString(`</section>`)
	s.page(w, r, "Help", template.HTML(b.String()), pageOptions{})
}

// helpSet changes one comfort setting, logged so it can be undone.
func (s *Server) helpSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key, value := r.PostForm.Get("key"), r.PostForm.Get("value")
	name := ""
	for _, c := range comfort {
		if c.key == key {
			for _, v := range c.values {
				if v[0] == value {
					name = strings.ToLower(v[1])
				}
			}
		}
	}
	if name == "" {
		s.failed(w, r, "Not changed", errors.New("that is not something this page changes"), "/help")
		return
	}
	was := s.app.Workspace.Get(key)
	if err := s.app.Workspace.Set(key, value); err != nil {
		s.failed(w, r, "Not changed", err, "/help")
		return
	}
	undo := s.record(r, chat.Change{Action: "set", Component: key, Detail: value, Before: map[string]any{"value": was}})
	s.tellAt(w, r, outcome{Title: "Changed", Text: "Now: " + name + ".", Undo: undo}, "/help")
}
