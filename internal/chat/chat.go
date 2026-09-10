// Package chat runs the conversation between a person and a model that can
// edit the canvas through tools. It is used by the web page and by the CLI.
package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// MessageType is the content type that holds conversation turns.
const MessageType = "message"

// maxToolRounds bounds how many tool-call rounds one turn may take.
const maxToolRounds = 8

// Service holds the dependencies for one workspace's chat.
type Service struct {
	Store        *store.Store
	Registry     *render.Registry
	Provider     llm.Provider
	ProviderErr  error
	HistoryLimit int
	ExtraPrompt  string
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

// Send records the person's message, runs the model with tools until it
// produces a final answer, records that answer with the list of canvas
// changes it made, and returns it. Failures are recorded as an error
// message in the conversation and also returned.
func (s *Service) Send(ctx context.Context, text string) (*store.Record, error) {
	if err := s.Available(); err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("message is empty")
	}
	if _, err := s.Store.Create(MessageType, map[string]any{"role": "user", "content": text}); err != nil {
		return nil, err
	}
	Record(s.Store, "human", Change{Action: "said", Detail: truncate(text, 80)})
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
	req := llm.Request{System: s.systemPrompt(), Messages: history, Tools: s.tools()}
	var changes []Change
	for round := 0; round <= maxToolRounds; round++ {
		resp, err := s.Provider.Complete(ctx, req)
		if err != nil {
			return s.fail(err)
		}
		if len(resp.ToolCalls) == 0 {
			reply := strings.TrimSpace(resp.Text)
			if reply == "" {
				reply = "(The model returned an empty reply.)"
			}
			return s.Store.Create(MessageType, s.fields(MessageType, map[string]any{"role": "assistant", "content": reply, "changes": changes}))
		}
		req.Messages = append(req.Messages, llm.Message{Role: llm.RoleAssistant, Content: resp.Text, ToolCalls: resp.ToolCalls})
		results := llm.Message{Role: llm.RoleTool}
		for _, call := range resp.ToolCalls {
			r := s.runTool(call)
			results.ToolResults = append(results.ToolResults, llm.ToolResult{CallID: call.ID, Content: r.text, IsError: r.isErr})
			if r.change != nil {
				changes = append(changes, *r.change)
				Record(s.Store, "assistant", *r.change)
			}
		}
		req.Messages = append(req.Messages, results)
		// The canvas changed, so refresh the system prompt for the next round.
		req.System = s.systemPrompt()
	}
	return s.fail(fmt.Errorf("stopped after %d tool rounds without a final answer", maxToolRounds))
}

// fields drops keys the workspace's schema does not define, so a workspace
// created before a field existed keeps working after an upgrade.
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
			out = append(out, llm.Message{Role: llm.RoleUser, Content: content})
		case "assistant":
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
	return s.Store.DeleteAll(MessageType)
}
