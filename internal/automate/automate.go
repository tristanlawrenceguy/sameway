// Package automate runs the person's actions, apart from the conversation
// with them (internal/chat): the narrow questions an action puts to the
// model on its own, to tag a record or suggest one from it.
package automate

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Service runs one workspace's actions.
type Service struct {
	// Book is the workspace's records; see internal/records.
	*records.Book
	// Model is the model the conversation talks to now, set by the app;
	// nil when none is connected. The conversation's model changes with
	// its settings, so it is asked each time.
	Model func() llm.Provider
}

// Result is what one run of an action did: words for whoever asked, an
// error flag, the change to log if any, and what it got back.
type Result struct {
	Text  string
	IsErr bool
	// Change is the one change to log; Changes is for a run that makes
	// several, each logged and shown on its own.
	Change  *records.Change
	Changes []records.Change
	// Answer is what an action got back, bare: a webhook's reply, a
	// command's output, the tags given; what the next action is given.
	Answer string
}

func fail(format string, args ...any) Result {
	return Result{Text: fmt.Sprintf(format, args...), IsErr: true}
}
