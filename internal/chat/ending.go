package chat

import (
	"context"
	"log"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// endings is what a turn has been asked again already as it ended: each
// is asked once. asked is the person's message; reply is the reply a
// review is checking.
type endings struct {
	nudged, corrected, reviewed bool
	asked, reply                string
}

// ending is a turn's reply when the model answers in words, or what to
// tell the model first, to answer again.
func (s *Service) ending(ctx context.Context, reply, said string, changes []records.Change, end *endings, on func(Event)) (string, []llm.Message) {
	_, server := s.Provider.(*llm.OpenAI)
	// A small model ends a turn of tool calls with no words a fifth of
	// the time; asked once, it says what it did. Only a model server's
	// (OpenAI-style): the Claude API takes no second person's turn after
	// a tool's, and Claude says what it did.
	if reply == "" && !end.nudged && server {
		end.nudged = true
		return askAgain(on, llm.Message{Role: llm.RoleUser, Content: fromSameway("your reply had no words. Tell the person in a sentence or two what you did, or answer them; call a tool first only if something is still to do.")})
	}
	if reply == "" {
		reply = "(The model returned an empty reply.)"
	}
	// A reply that says it made something it did not is told so once and
	// asked again; still saying so, it says nothing changed under it
	// (claims.go).
	if wrong := s.unmade(ctx, reply, said, changes); wrong != "" {
		if !end.corrected {
			end.corrected = true
			return askAgain(on, llm.Message{Role: llm.RoleAssistant, Content: reply}, llm.Message{Role: llm.RoleUser, Content: wrong})
		}
		reply += s.stillWrong(ctx, reply, said, changes) // claims_words.go
	}
	// A model server's turn that changed something checks it once against
	// what was asked (review.go).
	if server && !end.reviewed {
		if note := s.reviewNote(end.asked, changes); note != "" {
			end.reviewed, end.reply = true, reply
			return askAgain(on, llm.Message{Role: llm.RoleAssistant, Content: reply}, llm.Message{Role: llm.RoleUser, Content: note})
		}
	}
	return reply, nil
}

// askAgain is what to tell the model before it answers again. The
// person has been reading the reply as it came: a step says it is being
// checked, and the words after it are the new reply's (17-live.js).
func askAgain(on func(Event), ask ...llm.Message) (string, []llm.Message) {
	if on != nil {
		on(Event{Kind: "tool", Label: "Checking the reply"})
	}
	return "", ask
}

// broke ends a turn whose model did not answer. Stopped, what it did
// stays. While a reply it had already given was being checked, that
// reply stands: the work is done, and an error with Send again would
// have the person do it twice.
func (s *Service) broke(ctx context.Context, err error, said string, changes []records.Change, tools []map[string]any, end endings) (*store.Record, error) {
	if end.reply != "" {
		log.Printf("chat: the reply stands; checking it failed: %v", err)
		return s.reply(said, end.reply, changes, tools)
	}
	if ctx.Err() != nil {
		return s.reply(said, stoppedText, changes, tools)
	}
	return s.failAfter(said, err, changes, tools)
}
