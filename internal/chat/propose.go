package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

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

// proposeByModel is propose_change: a question in the model's own words.
// It asks only about the canvas. What cannot be taken back is asked by
// Sameway, in words the code writes, when the tool itself is called: a
// question carrying a setting or a command under a summary the model wrote
// would be answered for something other than what it said.
func (s *Service) proposeByModel(summary string, raw json.RawMessage) toolResult {
	var action map[string]any
	json.Unmarshal(raw, &action)
	delete(action, "summary")
	if tool, _ := action["tool"].(string); !modelProposable[tool] {
		return fail("propose_change carries only add_component, update_component, remove_component or remove_canvas; for anything else call the tool itself, and Sameway asks the person first when it must")
	}
	return s.propose(summary, action)
}

// proposable are the tools a proposal may carry. Asking to ask, or asking to
// clear everything, is not a question worth deferring.
var proposable = map[string]bool{
	"add_component": true, "update_component": true, "remove_component": true, "remove_canvas": true,
	// accept_action is what a person's Yes does to a command action: it is
	// accepted for good, then run.
	"accept_action": true,
	// What cannot be taken back is asked first, by the code: see consent.go.
	"run_action": true, "set_setting": true, "let_in": true, "change_field": true,
}

// modelProposable are the calls the model may carry in a question of its
// own wording: changes to the canvas, which are also undoable.
var modelProposable = map[string]bool{"add_component": true, "update_component": true, "remove_component": true, "remove_canvas": true}

// answering lets one answer at a time through, so a question answered Yes
// and No at once, or sent twice, is answered once: the second finds it
// answered, rather than running while the first still runs.
var answering sync.Mutex

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
	answering.Lock()
	defer answering.Unlock()
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
	result := s.runAgreed(llm.ToolCall{Name: tool, Args: args})
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
	// A tool that made several changes, such as a command that also put
	// its answer on the canvas, is logged as the person's: they said yes.
	for i := range result.changes {
		Record(s.Store, "human", result.changes[i])
	}
	return nil
}

// Dismiss answers no. Nothing changes except the question going away.
func (s *Service) Dismiss(id string) error {
	answering.Lock()
	defer answering.Unlock()
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
