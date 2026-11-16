package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A visitor from the internet asks the assistant on a published page. It
// answers once and keeps no conversation: nothing a stranger writes is
// kept but a request to the owner. It works for them as for anyone, with
// their reach: it reads what is published, and a record they ask to be
// made is put to the owner (requests.go). It is told nothing of the
// workspace beyond what it reads.

// SomeonePublished is who a visitor from the internet is, when they do
// not say.
const SomeonePublished = "Someone on your published page"

// onceSteps is how many rounds of tools an answer may take.
const onceSteps = 8

// AnswerOnce answers one question for whoever this service speaks for, by
// reading with their tools, without a conversation.
func (s *Service) AnswerOnce(ctx context.Context, site, question string) (string, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("say what you want to ask")
	}
	if s.Provider == nil {
		return "", fmt.Errorf("this workspace has no assistant to answer just now")
	}
	system := fmt.Sprintf("You answer a visitor to the published pages of %q, a Sameway workspace, by reading it with your tools: they reach only what its owner published, and all of it is theirs to read, whatever a record's own status says (a draft here is published). Answer in a few plain sentences from what you read, with each record's page when you name one, and say so when what you can read does not answer it. When they ask for something to be made, make it with create_record as you would for anyone, of the kind it is (a time to meet is an event, with when it starts), with their name and how to reach them in it if they gave them: it is not made but put to the owner as a request, and nothing happens until the owner says yes; tell them so. %s", site, records.DataNotInstructions)
	if who := s.who.Name; who != "" {
		system += " They say they are " + who + "."
	}
	msgs := []llm.Message{{Role: llm.RoleUser, Content: question}}
	defer s.tellOwner(s.Now())
	for range onceSteps {
		resp, err := s.Provider.Complete(ctx, llm.Request{System: system, Messages: msgs, Tools: s.Tools(), MaxTokens: 1500, For: s.forWhom()})
		if err != nil {
			return "", fmt.Errorf("the assistant could not answer: %w", err)
		}
		if len(resp.ToolCalls) == 0 {
			return strings.TrimSpace(resp.Text), nil
		}
		msgs = append(msgs, llm.Message{Role: llm.RoleAssistant, Content: resp.Text, ToolCalls: resp.ToolCalls})
		var results []llm.ToolResult
		for _, c := range resp.ToolCalls {
			r := s.run(c)
			results = append(results, llm.ToolResult{CallID: c.ID, Content: r.text, IsError: r.isErr})
		}
		msgs = append(msgs, llm.Message{Role: llm.RoleTool, ToolResults: results})
	}
	return "", fmt.Errorf("no answer after %d rounds; ask something narrower", onceSteps)
}
