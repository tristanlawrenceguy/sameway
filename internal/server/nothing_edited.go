package server

import (
	"errors"
	"net/http"
	"slices"
	"strings"
)

// nothingEdited answers an edit that named no field the way the form
// names them, prop-<name>. It used to send the asker back with nothing
// done and nothing said: in the agent evaluation (C-t5) an agent posted
// label=Up next to a block's props, and a JSON body without the JSON
// Accept header, got an empty 303 both times and only found out by
// reading the page again. Now it says what the form takes and, where the
// names it got are props or fields, what to send instead. A browser gets
// it as the outcome on the page it came from; an agent asking for JSON
// gets it as JSON; anything else gets it as plain text, with 400.
func (s *Server) nothingEdited(w http.ResponseWriter, r *http.Request, has func(string) bool, fallback string) {
	var got, instead []string
	for k := range r.PostForm {
		if k == "from" || k == "version" || k == "was" {
			continue
		}
		got = append(got, k)
		if has(k) {
			instead = append(instead, "prop-"+k)
		}
	}
	slices.Sort(got)
	slices.Sort(instead)
	msg := "nothing was changed: this form takes each field as prop-<name>, such as prop-label=Up next"
	if len(got) > 0 {
		msg += "; it got " + strings.Join(got, ", ")
	}
	if len(instead) > 0 {
		msg += ", so send " + strings.Join(instead, ", ")
	}
	if pageAction(r) || strings.Contains(r.Header.Get("Accept"), "text/html") {
		s.failed(w, r, "Not saved", errors.New(msg), fallback)
		return
	}
	http.Error(w, "Not saved: "+msg+".", http.StatusBadRequest)
}
