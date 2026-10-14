package agenda

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// The box for everything in the person's head, and handing it to the
// assistant to sort (today_focus.go says why).

// dumpBox is the box for everything in the person's head.
func (s *Service) dumpBox() string {
	return `<h2>Too much in your head?</h2>` + string(s.Form(ui.Form{Action: "/today/sort", Class: "sw-stack",
		Body: s.Component("textarea", map[string]any{"label": "Everything on your mind", "name": "words", "rows": 4,
			"hint": "As it comes, in any order: things to do, to remember, to reply to. The assistant makes them tasks with their days and says which three to start with."}),
		Button: &ui.Button{Label: "Sort it out for me", Variant: ui.Secondary}}))
}

// todaySort hands what the person wrote to the assistant, to sort.
func (s *Service) todaySort(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	words := strings.TrimSpace(r.PostForm.Get("words"))
	if words == "" {
		s.Failed(w, r, "Nothing to sort", errors.New("write what is on your mind first, as it comes"), "/today")
		return
	}
	ask := fmt.Sprintf("Sort this out for me: make each thing to do a task, with its day when I said one, tag the ones that matter most important, and tell me the three to start with, tagged start (and take start off any task that had it before). Ask me only if something cannot be guessed.\n\n%s", words)
	if _, err := s.ChatFor(r).SendFile(context.WithoutCancel(r.Context()), "", ask, ""); err != nil {
		s.ForgetModel()
		s.Failed(w, r, "Not sorted", err, "/today")
		return
	}
	// Back where they were: the page that says what to start with, not the
	// busiest page they have. What it said in full is in the chat.
	s.TellAt(w, r, web.Outcome{Title: "Sorted", Text: "The assistant has made your tasks. Its answer, with the three to start with, is in Chat."}, "/today")
}
