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
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// MessageType is the content type that holds conversation turns.
const MessageType = "message"

// Service holds the dependencies for one workspace's chat.
type Service struct {
	// SetSetting changes one line of workspace.yaml, when there is one:
	// the pace, which lists show, the model, the name. Set by the app.
	SetSetting func(key, value string) error
	// Setting reads one line of workspace.yaml as it is now, for the
	// questions that say what would change from what; set by the app.
	Setting func(key string) string
	// Update looks for a new version of the sameway program and installs
	// it when told to, set by the app; nil when this build cannot update
	// itself. See internal/update.
	Update func(ctx context.Context, install bool) (update.Outcome, error)
	// AddField and AddType change the workspace's schema while it runs, set
	// by the app; nil when the workspace cannot be changed from here.
	AddField     func(typeName string, f schema.Field) (*schema.Type, error)
	AddType      func(t *schema.Type) (*schema.Type, error)
	Store        *store.Store
	Registry     *render.Registry
	Provider     llm.Provider
	ProviderErr  error
	HistoryLimit int
	// Allow names the programs a command action may run, by name (curl,
	// python). Empty means none; see command.go for the boundary.
	Allow []string
	// Look reads a page of the workspace the way the person gets it, with
	// its scripts run where a browser is at hand; set by the server, nil
	// where there is none. See look.go.
	Look func(ctx context.Context, ask map[string]any) (string, error)
	// Publish sends to an MQTT topic, when the workspace has a broker;
	// nil means it has none. See mqtt.go.
	Publish func(topic, payload string) error
	// Tailnet waits up to the given time for the next step of joining the
	// person's Tailscale network and says it in their words; nil where the
	// workspace is not being served. See settings.go.
	Tailnet func(wait time.Duration) string
	// convo is the chat that is open, once known; see Current.
	convo       string
	ExtraPrompt string
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
	mine, err := s.message(fields)
	if err != nil {
		return nil, err
	}
	if on != nil {
		on(Event{Kind: "said", ID: mine.ID})
	}
	said := Record(s.Store, "human", Change{Action: "said", Detail: truncate(text, 80), Via: Via(ctx)})
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
	var p progress
	for {
		resp, err := s.complete(ctx, req, on)
		if err != nil {
			if ctx.Err() != nil {
				// The person stopped the turn; what it did stays.
				return s.reply(said, stoppedText, changes, tools)
			}
			return s.failAfter(said, err, changes, tools)
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
			return s.reply(said, reply, changes, tools)
		}
		if why := p.check(resp.ToolCalls); why != "" {
			return s.wrapUp(ctx, req, why, said, changes, tools, on)
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
		if why := p.round(results.ToolResults); why != "" {
			return s.wrapUp(ctx, req, why, said, changes, tools, on)
		}
	}
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
	recs, err := s.Messages()
	if err != nil {
		return nil, err
	}
	if s.HistoryLimit > 0 && len(recs) > s.HistoryLimit {
		recs = recs[len(recs)-s.HistoryLimit:]
	}
	var out []llm.Message
	for i := range recs {
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
	rec, storeErr := s.message(map[string]any{"role": "error", "content": err.Error()})
	if storeErr != nil {
		return nil, storeErr
	}
	return rec, err
}
