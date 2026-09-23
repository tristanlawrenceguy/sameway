package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ActionType is the content type that holds a person's own actions: a
// button that does something, inside Sameway or outside it.
const ActionType = "action"

// HTTPClient sends webhook actions. Tests point it at a local server.
var HTTPClient = &http.Client{Timeout: 10 * time.Second}

var actionTool = llm.Tool{
	Name:        "run_action",
	Description: "Run one of the person's actions now, by id: a webhook they set up (an alarm, a weather update), a command on their machine (the first run asks them once, on a card), or an arrangement. Actions of kind message are for the person to press, not for you. The result goes in the activity log; a webhook or command with show set puts its answer on the canvas.",
	Schema: map[string]any{"type": "object", "properties": map[string]any{
		"id": map[string]any{"type": "string", "description": "The action's id, from the actions listed in the prompt or find_records."},
	}, "required": []string{"id"}, "additionalProperties": false},
}

// Run carries out one action for whoever asked. A message action is the
// person's to press: it talks to the assistant on their behalf, and the
// canvas is where the reply lands.
func (s *Service) Run(ctx context.Context, id, canvas string) toolResult {
	rec, err := s.Store.Get(ActionType, id)
	if err != nil {
		return fail("no action with id %s; the actions are listed in the prompt, or find_records on %s finds one", id, ActionType)
	}
	title, _ := rec.Fields["title"].(string)
	switch kind, _ := rec.Fields["kind"].(string); kind {
	case "arrangement":
		name, _ := rec.Fields["arrangement"].(string)
		r := s.addArrangement(name, nil)
		if !r.isErr {
			r.changes = append(r.changes, Change{Action: "ran", Component: ActionType, ID: id, Detail: title, Href: "/t/" + ActionType + "/" + id})
		}
		return r
	case "command":
		return s.command(ctx, rec, title)
	case "mqtt":
		return s.mqttAction(ctx, rec, title)
	case "message":
		text, _ := rec.Fields["message"].(string)
		if text == "" {
			return fail("action %s has no message to send", title)
		}
		if _, err := s.SendOn(ctx, canvas, text); err != nil {
			return fail("could not send %q for action %s: %v", text, title, err)
		}
		return toolResult{text: "sent to the assistant: " + text, change: &Change{Action: "ran", Component: ActionType, ID: id, Detail: title, Href: "/t/" + ActionType + "/" + id}}
	default:
		return s.webhook(ctx, rec, title)
	}
}

// webhook makes the request the action describes and reports what came
// back. With show set, the answer is put on the canvas as a text block,
// made once and updated on every run, so a weather action is a weather
// block that stays current.
func (s *Service) webhook(ctx context.Context, rec *store.Record, title string) toolResult {
	url, _ := rec.Fields["url"].(string)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fail("action %s needs a url starting with http:// or https://", title)
	}
	method, _ := rec.Fields["method"].(string)
	if method == "" {
		method = http.MethodPost
	}
	body, _ := rec.Fields["body"].(string)
	var reader io.Reader
	if body != "" && method != http.MethodGet {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return fail("action %s: %v", title, err)
	}
	if reader != nil {
		ct := "text/plain; charset=utf-8"
		if json.Valid([]byte(body)) {
			ct = "application/json"
		}
		req.Header.Set("Content-Type", ct)
	}
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fail("%s failed: %v", title, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	answer := strings.TrimSpace(string(raw))
	if len([]rune(answer)) > 2000 {
		answer = string([]rune(answer)[:2000])
	}
	out := toolResult{text: fmt.Sprintf("%s answered %d", title, resp.StatusCode)}
	if answer != "" {
		out.text += ": " + answer
	}
	detail := fmt.Sprintf("%s (%d)", title, resp.StatusCode)
	out.changes = append(out.changes, Change{Action: "ran", Component: ActionType, ID: rec.ID, Detail: detail, Href: "/t/" + ActionType + "/" + rec.ID})
	if resp.StatusCode >= 400 {
		out.isErr = true
		return out
	}
	if show, _ := rec.Fields["show"].(bool); show && answer != "" {
		if c, err := s.show(rec, title, answer); err == nil {
			out.changes = append(out.changes, c)
		} else {
			out.text += ". Could not put it on the canvas: " + err.Error()
		}
	}
	return out
}

// show puts an action's answer on the canvas: a text block kept by id on
// the action, so the second run updates the first block rather than
// adding another.
func (s *Service) show(rec *store.Record, title, answer string) (Change, error) {
	props := map[string]any{"content": answer}
	if id, _ := rec.Fields["block"].(string); id != "" {
		if was, err := s.Store.Get(BlockType, id); err == nil {
			if _, err := s.Store.Update(BlockType, id, s.fields(BlockType, map[string]any{"props": props, "actor": "assistant"})); err != nil {
				return Change{}, err
			}
			return Change{Action: "updated", Component: "text", ID: id, Detail: title, Href: "/canvas/" + id, Before: was.Fields}, nil
		}
	}
	r := s.addComponent("text", props, look{})
	if r.isErr || r.change == nil {
		return Change{}, errors.New(r.text)
	}
	s.Store.Update(ActionType, rec.ID, map[string]any{"block": r.change.ID})
	return *r.change, nil
}

// actionsDigest lists the person's actions for the prompt, so the model
// can put one on a button or run one when asked.
func (s *Service) actionsDigest() string {
	if _, ok := s.Store.Types().Get(ActionType); !ok {
		return ""
	}
	recs, err := s.Store.List(ActionType, store.ListOptions{OrderBy: "created_at"})
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

// RunAs runs an action for a person or an agent and records what it did
// under their name, returning what the action answered. When the action
// needs accepting first, proposal is the question now waiting for them.
func (s *Service) RunAs(ctx context.Context, actor, id, canvas string) (text, proposal string, err error) {
	r := s.Run(ctx, id, canvas)
	for i := range r.changes {
		r.changes[i].Via = Via(ctx)
		Record(s.Store, actor, r.changes[i])
	}
	if r.change != nil {
		if r.change.Action == "proposed" {
			proposal = r.change.ID
		}
		Record(s.Store, actor, *r.change)
	}
	if r.isErr {
		return r.text, proposal, errors.New(r.text)
	}
	return r.text, proposal, nil
}
