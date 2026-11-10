package chat

import (
	"context"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// endings is what a turn has been asked again already as it ended: each
// is asked once.
type endings struct {
	nudged, corrected, reviewed bool
}

// ending is a turn's reply when the model answers in words, or what to
// tell the model first, to answer again.
func (s *Service) ending(ctx context.Context, reply, said string, changes []records.Change, end *endings) (string, []llm.Message) {
	_, server := s.Provider.(*llm.OpenAI)
	// A small model ends a turn of tool calls with no words a fifth of
	// the time; asked once, it says what it did. Only a model server's
	// (OpenAI-style): the Claude API takes no second person's turn after
	// a tool's, and Claude says what it did.
	if reply == "" && !end.nudged && server {
		end.nudged = true
		return "", []llm.Message{{Role: llm.RoleUser, Content: fromSameway("your reply had no words. Tell the person in a sentence or two what you did, or answer them; call a tool first only if something is still to do.")}}
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
			return "", []llm.Message{{Role: llm.RoleAssistant, Content: reply}, {Role: llm.RoleUser, Content: wrong}}
		}
		reply += s.stillWrong(ctx, reply, said, changes) // claims_words.go
	}
	// A model server's turn that changed something checks it once against
	// what was asked (review.go).
	if server && !end.reviewed {
		if note := s.reviewNote(changes); note != "" {
			end.reviewed = true
			return "", []llm.Message{{Role: llm.RoleAssistant, Content: reply}, {Role: llm.RoleUser, Content: note}}
		}
	}
	return reply, nil
}
