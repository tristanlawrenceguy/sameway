// Package automate runs the person's actions: from a button, a hook, a
// tool call, on a schedule (schedule.go), or on its own when something
// happens to a record (happened.go, worker.go); and the narrow questions
// an action puts to the model on its own, to tag a record or suggest one
// from it. The conversation with the person is internal/chat; what an
// action needs from it is Chat.
package automate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Service runs one workspace's actions.
type Service struct {
	// Book is the workspace's records; see internal/records.
	*records.Book
	// Chat is the conversation, set by the app; see Chat.
	Chat Chat
	// Model is the model the conversation talks to now, set by the app;
	// nil when none is connected. The conversation's model changes with
	// its settings, so it is asked each time.
	Model func() llm.Provider
	// Allow names the programs a command action may run, by name (curl,
	// python). Empty means none; see command.go for the boundary.
	Allow []string
	// Workdir is where commands run when an action names no folder: the
	// workspace, set by the app.
	Workdir string
	// HTTP sends webhook actions; nil is a client with a ten second limit.
	HTTP *http.Client
	// Publish sends to an MQTT topic, when the workspace has a broker;
	// nil means it has none. See mqtt.go.
	Publish func(topic, payload string) error
	// Tell lets the person know, beyond the page, what was done for them
	// while they were away: an automation's turn (happened.go).
	Tell func(title, text, url string)
	auto *automation // actions waiting for something to happen; happened.go
}

// Chat is what running an action needs from the conversation: a message
// action talks to the assistant, an arrangement lays out a page, an
// answer shown goes on the canvas, and a command asks first.
type Chat interface {
	SendOn(ctx context.Context, canvas, text string) (*store.Record, error)
	AddArrangement(name string) Result
	Show(action *store.Record, title, answer string) (records.Change, error)
	Ask(q Question, call map[string]any) Result
}

// Question is put to the person on a card, in the code's words, with the
// two answers.
type Question struct{ Ask, Detail, Yes, No string }

// For is the service running actions for one conversation: what a
// message action says, and where a shown answer lands, are that one's.
func (s *Service) For(c Chat) *Service {
	cp := *s
	cp.Chat = c
	return &cp
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
