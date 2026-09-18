// Package chat runs the conversation between a person and a model that can
// edit the canvas through tools. It is used by the web page and by the CLI.
package chat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// MessageType is the content type that holds conversation turns.
const MessageType = "message"

// maxToolRounds bounds how many tool-call rounds one turn may take. A
// model that makes one call per round needs a round for each item of a
// list; when the bound is reached the turn ends in words, not an error.
const maxToolRounds = 12

// Service holds the dependencies for one workspace's chat.
type Service struct {
	// SetSetting changes one line of workspace.yaml, when there is one:
	// the pace, which lists show, the model, the name. Set by the app.
	SetSetting func(key, value string) error
	// AddField and AddType change the workspace's schema while it runs, set
	// by the app; nil when the workspace cannot be changed from here.
	AddField     func(typeName string, f schema.Field) (*schema.Type, error)
	AddType      func(t *schema.Type) (*schema.Type, error)
	Store        *store.Store
	Registry     *render.Registry
	Provider     llm.Provider
	ProviderErr  error
	HistoryLimit int
	ExtraPrompt  string
	// Now tells the model what day it is, so a calendar for "this month"
	// is this month. Defaults to time.Now; tests pin it.
	Now func() time.Time

	// current is the tab the person is looking at while a turn runs: "" is
	// Home. New blocks land there, and the prompt describes that tab.
	current string
}

// Available reports whether the workspace has the content types the chat
// needs. The error explains what is missing.
func (s *Service) Available() error {
	for _, name := range []string{MessageType, BlockType} {
		if _, ok := s.Store.Types().Get(name); !ok {
			return fmt.Errorf("content type %q is missing from schema/; run `sameway init --force` to restore it", name)
		}
	}
	return nil
}

// Send is SendOn for a person looking at Home.
func (s *Service) Send(ctx context.Context, text string) (*store.Record, error) {
	return s.SendOn(ctx, "", text)
}

// SendOn records the person's message, runs the model with tools until it
// produces a final answer, records that answer with the list of canvas
// changes it made, and returns it. canvas is the tab the person is looking
// at, "" for Home: new blocks go there and the prompt describes it.
// Failures are recorded as an error message in the conversation and also
// returned.
func (s *Service) SendOn(ctx context.Context, canvas, text string) (*store.Record, error) {
	return s.SendFile(ctx, canvas, text, "")
}

// SendFile is SendOn with a file attached: the message carries the file's
// id, the model gets the file's text with the message, and the person
// sees which file went with what they said.
func (s *Service) sendTurn(ctx context.Context, canvas, text, fileID string, on func(Event)) (*store.Record, error) {
	if err := s.Available(); err != nil {
		return nil, err
	}
	if !s.HasCanvas(canvas) {
		return nil, fmt.Errorf("there is no canvas %q", canvas)
	}
	s.current = canvas
	text = strings.TrimSpace(text)
	if text == "" && fileID != "" {
		text = "Here is a file."
	}
	if text == "" {
		return nil, errors.New("message is empty")
	}
	fields := map[string]any{"role": "user", "content": text}
	if fileID != "" {
		fields["file"] = fileID
	}
	mine, err := s.Store.Create(MessageType, s.fields(MessageType, fields))
	if err != nil {
		return nil, err
	}
	if on != nil {
		on(Event{Kind: "said", ID: mine.ID})
	}
	said := Record(s.Store, "human", Change{Action: "said", Detail: truncate(text, 80)})
	if s.Provider == nil {
		err := s.ProviderErr
		if err == nil {
			err = llm.ErrNotConfigured
		}
		return s.fail(err)
	}
	history, err := s.history()
	if err != nil {
		return nil, err
	}
	req := llm.Request{System: s.systemPrompt(), Messages: history, Tools: s.Tools()}
	if on != nil && s.runsToolsOutside() {
		// The tools run in another program; the log is where their
		// changes show, so it is watched while the turn runs.
		defer s.watch(ctx, said, on)()
	}
	var changes []Change
	var tools []map[string]any
	corrected := false
	for round := 0; round <= maxToolRounds; round++ {
		resp, err := s.complete(ctx, req, on)
		if err != nil {
			return s.fail(err)
		}
		if len(resp.ToolCalls) == 0 {
			reply := strings.TrimSpace(resp.Text)
			if reply == "" {
				reply = "(The model returned an empty reply.)"
			}
			// A reply that names pages which do not exist made nothing:
			// the model answered in words where a tool was needed. It is
			// told so once, with each page it named, and asked again.
			if !corrected {
				claims, _ := VerifyClaims(ctx, s.Store, reply)
				var missing []string
				for _, c := range claims {
					if !c.Exists {
						missing = append(missing, c.URL)
					}
				}
				if len(missing) > 0 {
					corrected = true
					log.Printf("chat: the reply named %s, which does not exist; asking the model to make it", strings.Join(missing, ", "))
					req.Messages = append(req.Messages,
						llm.Message{Role: llm.RoleAssistant, Content: reply},
						llm.Message{Role: llm.RoleUser, Content: "There is no page at " + strings.Join(missing, ", ") + ": nothing was made. Make it with the tools, then say where it is."})
					continue
				}
			}
			if s.runsToolsOutside() {
				// The tools ran in the program the person is signed in to;
				// the log knows what they did.
				changes = s.changesAfter(said)
			}
			return s.Store.Create(MessageType, s.fields(MessageType, map[string]any{"role": "assistant", "content": reply, "changes": changes, "tools": tools}))
		}
		req.Messages = append(req.Messages, llm.Message{Role: llm.RoleAssistant, Content: resp.Text, ToolCalls: resp.ToolCalls})
		results := llm.Message{Role: llm.RoleTool}
		for _, call := range resp.ToolCalls {
			if on != nil {
				on(Event{Kind: "tool", Tool: call.Name, Label: describe(call)})
			}
			r := s.run(call)
			results.ToolResults = append(results.ToolResults, llm.ToolResult{CallID: call.ID, Content: r.text, IsError: r.isErr})
			tools = append(tools, used(call, r))
			if r.change != nil {
				changes = append(changes, *r.change)
			}
			changes = append(changes, r.changes...)
			if on != nil {
				if r.change != nil {
					on(Event{Kind: "change", Change: r.change})
				}
				for i := range r.changes {
					on(Event{Kind: "change", Change: &r.changes[i]})
				}
			}
		}
		req.Messages = append(req.Messages, results)
		// The canvas changed, so refresh the system prompt for the next round.
		req.System = s.systemPrompt()
	}
	// Enough tools for one turn. The model is asked to stop and say where
	// things stand, so the person hears what was done and what was not
	// rather than an error; what it changed is kept either way.
	req.Tools = nil
	req.Messages = append(req.Messages, llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("That is %d rounds of tools, the most one turn may use. Stop here and say, in a few words, what you did and what is still to do.", maxToolRounds)})
	resp, err := s.complete(ctx, req, on)
	if err != nil || strings.TrimSpace(resp.Text) == "" {
		return s.fail(fmt.Errorf("stopped after %d tool rounds without a final answer", maxToolRounds))
	}
	if s.runsToolsOutside() {
		changes = s.changesAfter(said)
	}
	return s.Store.Create(MessageType, s.fields(MessageType, map[string]any{"role": "assistant", "content": strings.TrimSpace(resp.Text), "changes": changes, "tools": tools}))
}

// fields drops keys the workspace's schema does not define, so a record
// written by newer code still saves into an older workspace. Internal
// types are completed from the built-in preset when the app loads
// (schema.Set.Complete), so a layout field the tools offer is never lost
// this way; what remains is a safety net for a field a workspace has
// deliberately removed.
func (s *Service) fields(typeName string, in map[string]any) map[string]any {
	t, ok := s.Store.Types().Get(typeName)
	if !ok {
		return in
	}
	out := map[string]any{}
	for k, v := range in {
		if _, known := t.Field(k); known {
			out[k] = v
		}
	}
	return out
}

// history returns the recent user and assistant turns as model messages.
// Error notices are shown to the person but not sent to the model.
func (s *Service) history() ([]llm.Message, error) {
	recs, err := s.Store.List(MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: s.HistoryLimit})
	if err != nil {
		return nil, err
	}
	var out []llm.Message
	for i := len(recs) - 1; i >= 0; i-- {
		role, _ := recs[i].Fields["role"].(string)
		content, _ := recs[i].Fields["content"].(string)
		switch role {
		case "user":
			// A file that came with the message comes with it to the model too.
			if fileID, _ := recs[i].Fields["file"].(string); fileID != "" {
				content += s.attachment(fileID)
			}
			out = append(out, llm.Message{Role: llm.RoleUser, Content: content})
		case "assistant":
			// The tools this reply used come first, as the turn they were.
			out = append(out, replay(recs[i].Fields["tools"])...)
			out = append(out, llm.Message{Role: llm.RoleAssistant, Content: content})
		}
	}
	// Providers require the conversation to start with a user turn.
	for len(out) > 0 && out[0].Role != llm.RoleUser {
		out = out[1:]
	}
	return out, nil
}

// fail stores an error notice in the conversation and returns it with the error.
func (s *Service) fail(err error) (*store.Record, error) {
	Record(s.Store, "system", Change{Action: "failed", Detail: truncate(err.Error(), 200)})
	rec, storeErr := s.Store.Create(MessageType, map[string]any{"role": "error", "content": err.Error()})
	if storeErr != nil {
		return nil, storeErr
	}
	return rec, err
}

// Clear deletes the conversation. The canvas is left alone.
func (s *Service) Clear() error {
	if err := s.Available(); err != nil {
		return err
	}
	Record(s.Store, "human", Change{Action: "cleared", Component: "conversation"})
	// The questions the assistant asked were part of the conversation; a
	// cleared one has no questions still waiting under it.
	for _, p := range s.Proposals() {
		s.Store.Update(ProposalType, p.ID, map[string]any{"state": "dismissed"})
	}
	return s.Store.DeleteAll(MessageType)
}
