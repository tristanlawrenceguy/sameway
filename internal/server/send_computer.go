package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A link found on the phone is often wanted at the computer's keyboard,
// and so are a few words: an address, a code, a line to paste into a
// form. Send to the computer takes either across: a link opens in the
// computer's browser, words go on its clipboard, ready to paste. It is
// held on the phone's app icon (manifest shortcuts), and needs Sameway
// opened on the computer, where there is a screen to show it.

// sendPage is the field to send from.
func (s *Server) sendPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	if fromThisComputer(r) {
		b.WriteString(`<p class="sw-muted">This is the computer itself: this page is for a phone or another device, to send something here.</p>`)
	}
	b.WriteString(string(s.form(ui.Form{Action: "/computer/send", Class: "sw-stack",
		Body: s.component("textarea", map[string]any{"label": "A link or some words", "name": "what", "rows": 4,
			"hint": "A link opens in the computer's browser. Words go on its clipboard, to paste there."}),
		Button: &ui.Button{Label: "Send to the computer"}})))
	s.page(w, r, "Send to the computer", template.HTML(b.String()), pageOptions{Lede: "Open a link or paste some words on the computer, from your phone."})
}

// sendToComputer opens a link on the computer's screen, or puts words on
// its clipboard.
func (s *Server) sendToComputer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	what := strings.TrimSpace(r.PostForm.Get("what"))
	switch {
	case what == "":
		s.failed(w, r, "Not sent", errors.New("nothing came to send: paste a link or some words"), "/computer/send")
	case s.fleet == nil || s.fleet.Show == nil || s.fleet.Copy == nil:
		s.failed(w, r, "Not sent", errors.New("this Sameway runs without a screen of its own: open it on the computer with sameway open"), "/computer/send")
	case !strings.ContainsAny(what, " \n\t") && strings.Contains(what, "://"):
		u, err := keptLink(what)
		if err != nil {
			s.failed(w, r, "Not sent", err, "/computer/send")
			return
		}
		if err := s.fleet.Show(u.String()); err != nil {
			s.failed(w, r, "Not sent", err, "/computer/send")
			return
		}
		s.tell(w, r, outcome{Title: "Open on the computer", Text: u.Host + " is open in the computer's browser."}, "/computer/send")
	default:
		if err := s.fleet.Copy(what); err != nil {
			s.failed(w, r, "Not sent", err, "/computer/send")
			return
		}
		s.tell(w, r, outcome{Title: "On the computer's clipboard", Text: "Paste it there with Ctrl+V, or Cmd+V on a Mac."}, "/computer/send")
	}
}
