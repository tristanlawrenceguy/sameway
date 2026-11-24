package chat

import (
	"encoding/json"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// The owner answers a request through the assistant, with words of their
// own told to whoever asked where they asked (request_answer.go tells them).

var answerRequestOp = Op{Title: "Answer a request",
	Access: ForEditors,
	Doing:  saying("Answering the request"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		a := answerArgsOf(call.Args)
		if a.Say == "" {
			return toolResult{}, false
		}
		p, err := s.Store.Get(records.ProposalType, a.ID)
		if err != nil {
			return fail("there is no request waiting with that id"), true
		}
		asked, _ := p.Fields["asked"].(string)
		answer, button := "no", "Say no, with these words"
		if a.Yes {
			answer, button = "yes", "Say yes, with these words"
		}
		return s.ask(question{"Answer " + answer + " to the request to " + asked + ", and say this?",
			"Whoever asked is told " + answer + ", where they asked, with your words:\n\n" + a.Say, button, "Don't"},
			map[string]any{"tool": "answer_request", "id": a.ID, "yes": a.Yes, "say": a.Say}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.answerRequest(call.Args) },
	Tool: llm.Tool{Name: "answer_request",
		Description: "Answer yes or no a request, by its id: made as the owner would, the asker told where they asked. say: words for them with the answer, in one message (why, or another time to offer), put to the person first.",
		Schema: obj(map[string]any{
			"id":  map[string]any{"type": "string"},
			"yes": map[string]any{"type": "boolean"},
			"say": map[string]any{"type": "string"},
		}, "id", "yes")}}

type answerArgs struct {
	ID  string `json:"id"`
	Yes bool   `json:"yes"`
	Say string `json:"say"`
}

func answerArgsOf(raw json.RawMessage) answerArgs {
	var a answerArgs
	json.Unmarshal(raw, &a)
	a.Say = strings.TrimSpace(trim.Clip(a.Say, 2000))
	return a
}

func (s *Service) answerRequest(raw json.RawMessage) toolResult {
	a := answerArgsOf(raw)
	p, err := s.Store.Get(records.ProposalType, a.ID)
	if err != nil || p.Fields["by_action"] != ByRequest || RequestState(p) != "waiting" {
		return fail("there is no request waiting with that id")
	}
	if !s.owner() {
		return fail("a request is the owner's to answer")
	}
	owner := *s
	owner.who = records.Visitor{Access: records.Owner}
	if a.Say != "" { // told with the answer, tellAsker
		if _, err := owner.Apply(records.Op{Type: records.ProposalType, ID: p.ID, After: map[string]any{"answer": a.Say}}); err != nil {
			return fail("%v", err)
		}
	}
	if !a.Yes {
		if err := owner.Dismiss(p.ID); err != nil {
			return fail("%v", err)
		}
		return toolResult{text: "Turned down; whoever asked is told."}
	}
	if err := owner.Accept(p.ID); err != nil {
		return fail("not made: %v", err)
	}
	return toolResult{text: "Made, as asked; whoever asked is told."}
}
