package web

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Every action a person takes from a page ends the same way: they are back
// on the page they were on, and one message says what happened, in the
// same place on every page, once. A failure says what went wrong in plain
// words, and what they typed is not lost: the draft of an edit is kept
// until the edit is said to be saved (18-drafts.js). Before this, each
// action had its own way, or none: a flag left in the address, a message
// in the chat on a page with no chat, a line in the activity log, a bare
// error page. The crew's person-facing findings were mostly those. The
// server tells it (Deps.Tell) and shows it on the next page.

// An Outcome is what happened, said once on the next page.
type Outcome struct {
	// Failed is a failure, announced as an alert; otherwise it is a
	// status, announced politely.
	Failed bool   `json:"f,omitempty"`
	Title  string `json:"t"`
	Text   string `json:"x,omitempty"`
	// For is the form the outcome answers, by its action, so the page
	// knows which draft is now saved and can let it go.
	For string `json:"o,omitempty"`
	// Undo is the activity entry that takes it back, when it can be: the
	// message carries the Undo, where the person is looking.
	Undo string `json:"u,omitempty"`
	// Of is what the Undo takes back, when the text says more than that:
	// "Undo Tea", not "Undo Tea rings at 19:00".
	Of string `json:"w,omitempty"`
	// Problems are what stopped a form, each about one field: the message
	// is then an error summary, each problem leading to its field.
	Problems []Problem `json:"p,omitempty"`
}

// A Problem is one answer a form could not take.
type Problem struct {
	Field string `json:"f"`
	Text  string `json:"t"`
}

// PlainError is an error as a person reads it.
func PlainError(err error) string {
	if errors.Is(err, store.ErrNotFound) {
		return "It is not there any more; it may have been deleted."
	}
	text := err.Error()
	for _, prefix := range []string{"invalid: ", "validation failed: ", "bad request: "} {
		text = strings.TrimPrefix(text, prefix)
	}
	if text != "" {
		text = strings.ToUpper(text[:1]) + text[1:]
		if !strings.HasSuffix(text, ".") {
			text += "."
		}
	}
	return text
}

// BackOf is the page a person acted from: the from the form carries, or
// the page the request came from, on this server; fallback otherwise.
// Only a path here is ever a way back.
func BackOf(r *http.Request, fallback string) string {
	if from := r.FormValue("from"); Local(from) {
		return from
	}
	if ref, err := url.Parse(r.Referer()); err == nil && ref.Host == r.Host && Local(ref.Path) {
		q := ref.Query()
		q.Del("saved")
		ref.RawQuery = q.Encode()
		return ref.RequestURI()
	}
	return fallback
}

// Local says a path is a page on this server, not somewhere else.
func Local(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.HasPrefix(path, "/\\")
}
