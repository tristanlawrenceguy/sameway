package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/automate"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An action is run by internal/automate. What is here is the
// conversation's side of it: the tools that run one and accept a command,
// the actions listed in the prompt, and what running one needs from the
// conversation (automate.Chat), which this service gives.

var actionOp = Op{Title: "Run an action", Traits: Traits{OpenWorld: true},
	Words: []string{"run", "press", "button", "action", "webhook", "send to"},
	Doing: saying("Running an action"),
	Asks:  (*Service).askAction,
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		return ran(s.Automate.For(s).Run(context.Background(), a.ID, s.current))
	},
	Tool: llm.Tool{
		Name:        "run_action",
		Description: "Run one of the person's actions now, by id: a webhook they set up (an alarm, a weather update), a command on their machine (the first run asks them once, on a card), or an arrangement. Actions of kind message are for the person to press, not for you. The result goes in the activity log; a webhook or command with show set puts its answer on the canvas.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{
			"id": map[string]any{"type": "string", "description": "The action's id, from the actions listed in the prompt or find_records."},
		}, "required": []string{"id"}, "additionalProperties": false},
	}}

// acceptOp is a command card's Yes, carried by a question: never offered
// to the model, which may only ask.
var acceptOp = Op{Title: "Accept a command", Traits: Traits{OpenWorld: true}, Tool: llm.Tool{Name: "accept_action"},
	Offered: func(*Service, *llm.Tool) bool { return false },
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		return ran(s.Automate.For(s).Accept(context.Background(), a.ID))
	}}

// AddArrangement lays out an arrangement action's page.
func (s *Service) AddArrangement(name string) automate.Result {
	return s.addArrangement(name, nil).result()
}

// Ask puts a question to the person, with the call their Yes runs.
func (s *Service) Ask(q automate.Question, call map[string]any) automate.Result {
	return s.ask(question{q.Ask, q.Detail, q.Yes, q.No}, call).result()
}

// Show puts an action's answer on the canvas: a text block kept by id on
// the action, so the second run updates the first block rather than
// adding another.
func (s *Service) Show(rec *store.Record, title, answer string) (records.Change, error) {
	props := map[string]any{"content": answer}
	if id, _ := rec.Fields["block"].(string); id != "" {
		if _, err := s.Store.Get(records.BlockType, id); err == nil {
			done, err := s.Apply(records.Op{Type: records.BlockType, ID: id, After: s.fields(records.BlockType, map[string]any{"props": props, "actor": "assistant"})})
			if err != nil {
				return records.Change{}, err
			}
			return records.Change{Action: "updated", Component: "text", ID: id, Detail: title, Href: "/canvas/" + id, Ops: done}, nil
		}
	}
	r := s.addComponent("text", props, look{})
	if r.isErr || r.change == nil {
		return records.Change{}, errors.New(r.text)
	}
	s.Apply(records.Op{Type: records.ActionType, ID: rec.ID, After: map[string]any{"block": r.change.ID}})
	return *r.change, nil
}

// actionsDigest lists the person's actions for the prompt, so the model
// can put one on a button or run one when asked.
func (s *Service) actionsDigest() string {
	if _, ok := s.Store.Types().Get(records.ActionType); !ok {
		return ""
	}
	recs, err := s.Store.List(records.ActionType, store.ListOptions{OrderBy: "created_at"})
	if err != nil || len(recs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nThe person's actions (id: title, kind). A button block with action set to an id runs it when pressed; run_action runs one now:\n")
	for _, r := range recs {
		fmt.Fprintf(&b, "%s: %s, %s\n", r.ID, r.Fields["title"], r.Fields["kind"])
	}
	return b.String()
}
