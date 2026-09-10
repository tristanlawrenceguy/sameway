package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ProposalType is the content type holding changes waiting for an answer.
const ProposalType = "proposal"

// A proposal is how the assistant asks instead of acting. It stores the tool
// call it would have made; accepting runs that call through exactly the same
// code the assistant's own tool calls take, so there is no second path that
// could behave differently from the one under test.

// proposals lists the questions still waiting for an answer, oldest first.
func (s *Service) Proposals() []*store.Record {
	if _, ok := s.Store.Types().Get(ProposalType); !ok {
		return nil
	}
	recs, err := s.Store.List(ProposalType, store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return nil
	}
	var out []*store.Record
	for _, r := range recs {
		if r.Fields["state"] == "pending" {
			out = append(out, r)
		}
	}
	return out
}

// propose records a question rather than making the change.
func (s *Service) propose(summary string, action map[string]any) toolResult {
	if _, ok := s.Store.Types().Get(ProposalType); !ok {
		return fail("this workspace has no proposal type, so changes cannot be offered for approval; make the change directly or run `sameway init --force`")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fail("a proposal needs a summary: the question to put to the person")
	}
	tool, _ := action["tool"].(string)
	if !proposable[tool] {
		return fail("proposals can only carry %s", strings.Join(proposableNames(), ", "))
	}
	rec, err := s.Store.Create(ProposalType, map[string]any{"summary": summary, "action": action, "state": "pending"})
	if err != nil {
		return fail("could not save the proposal: %v", err)
	}
	return toolResult{
		text:   "asked the person: " + summary + " (proposal " + rec.ID + ", nothing has changed yet)",
		change: &Change{Action: "proposed", ID: rec.ID, Detail: truncate(summary, 80)},
	}
}

// proposable are the tools a proposal may carry. Asking to ask, or asking to
// clear everything, is not a question worth deferring.
var proposable = map[string]bool{
	"add_component": true, "update_component": true, "remove_component": true,
}

func proposableNames() []string {
	out := make([]string, 0, len(proposable))
	for k := range proposable {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// Accept runs a pending proposal and records who agreed to it.
func (s *Service) Accept(id string) error {
	rec, err := s.Store.Get(ProposalType, id)
	if err != nil {
		return err
	}
	if rec.Fields["state"] != "pending" {
		return errors.New("that question has already been answered")
	}
	action, _ := rec.Fields["action"].(map[string]any)
	tool, _ := action["tool"].(string)
	if !proposable[tool] {
		return fmt.Errorf("this proposal carries an action that cannot be run: %q", tool)
	}
	args, err := json.Marshal(action)
	if err != nil {
		return err
	}
	result := s.runTool(llm.ToolCall{Name: tool, Args: args})
	if result.isErr {
		return errors.New(result.text)
	}
	if _, err := s.Store.Update(ProposalType, id, map[string]any{"state": "accepted"}); err != nil {
		return err
	}
	// The assistant made the change, but the person is why it happened.
	summary, _ := rec.Fields["summary"].(string)
	Record(s.Store, "human", Change{Action: "agreed to", Detail: truncate(summary, 80)})
	if result.change != nil {
		Record(s.Store, "assistant", *result.change)
	}
	return nil
}

// Dismiss answers no. Nothing changes except the question going away.
func (s *Service) Dismiss(id string) error {
	rec, err := s.Store.Get(ProposalType, id)
	if err != nil {
		return err
	}
	if rec.Fields["state"] != "pending" {
		return errors.New("that question has already been answered")
	}
	if _, err := s.Store.Update(ProposalType, id, map[string]any{"state": "dismissed"}); err != nil {
		return err
	}
	summary, _ := rec.Fields["summary"].(string)
	Record(s.Store, "human", Change{Action: "declined", Detail: truncate(summary, 80)})
	return nil
}
