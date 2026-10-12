package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// What waits for the person, read by the assistant, MCP and the API from
// the same list Today shows and the morning brief counts (waiting.go).

var waitingOp = Op{Title: "See what waits for you", Traits: Traits{ReadOnly: true, Idempotent: true},
	Access: ForViewers,
	Words:  []string{"waiting", "waits", "needs me", "to answer", "to reply", "to sort", "inbox"},
	Doing:  saying("Looking at what waits for you"),
	Run:    func(s *Service, _ toolArgs, _ llm.ToolCall) toolResult { return s.whatWaits() },
	Tool: llm.Tool{Name: "what_waits", Description: "What waits for the person, as Today shows it and the morning brief counts it: the assistant's questions, emails and shared messages to sort, emails where it is their turn to reply, what actions suggested and tags to check. Each with its words, its page, and the answers a person presses there; an agent outside answers one by POSTing that path with the id, and only when the person asked it to: the rest is theirs to answer.",
		Schema: obj(map[string]any{})},
	// Offered where something can wait: questions, or email.
	Offered: func(s *Service, _ *llm.Tool) bool {
		return s.hasTypes(records.ProposalType) || s.hasTypes(records.EmailType)
	}}

// whatWaits is the list as a model reads it: how many, then one line each.
func (s *Service) whatWaits() toolResult {
	list := s.Waiting()
	if len(list) == 0 {
		return toolResult{text: "nothing waits for the person"}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d waiting for the person:", len(list))
	for _, w := range list {
		var answers []string
		for _, p := range w.Presses {
			a := p.Label + " (POST " + p.Post
			if p.ID != "" {
				a += " id=" + p.ID
			}
			if p.Takes != "" {
				a += " " + p.Takes + "=…"
			}
			answers = append(answers, a+")")
		}
		fmt.Fprintf(&b, "\n- %s: %s, at %s", w.Kind, w.Words, w.Link)
		if len(answers) > 0 {
			b.WriteString("; answers: " + strings.Join(answers, ", "))
		}
	}
	return toolResult{text: b.String()}
}
