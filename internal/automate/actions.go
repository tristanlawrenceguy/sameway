package automate

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// webhooks sends webhook actions where the service names no client.
var webhooks = &http.Client{Timeout: 10 * time.Second}

// Run carries out one action for whoever asked. A message action is the
// person's to press: it talks to the assistant on their behalf, and the
// canvas is where the reply lands.
func (s *Service) Run(ctx context.Context, id, canvas string) Result {
	rec, err := s.Store.Get(records.ActionType, id)
	if err != nil {
		return fail("no action with id %s; the actions are listed in the prompt, or find_records on %s finds one", id, records.ActionType)
	}
	return s.runRecord(ctx, rec, canvas)
}

// runRecord carries out an action as given: as stored, or filled with
// what set it off (happened.go).
func (s *Service) runRecord(ctx context.Context, rec *store.Record, canvas string) Result {
	id := rec.ID
	title, _ := rec.Fields["title"].(string)
	switch kind, _ := rec.Fields["kind"].(string); kind {
	case "arrangement":
		name, _ := rec.Fields["arrangement"].(string)
		r := s.Chat.AddArrangement(name)
		if !r.IsErr {
			r.Changes = append(r.Changes, records.Change{Action: "ran", Component: records.ActionType, ID: id, Detail: title, Href: relate.Page(records.ActionType, id)})
		}
		return r
	case "command":
		return s.command(ctx, rec, title)
	case "classify":
		vars, _ := ctx.Value(triggerKey{}).(map[string]string)
		return s.Classify(ctx, rec, vars["type"], vars["id"]) // classify.go
	case "suggest":
		vars, _ := ctx.Value(triggerKey{}).(map[string]string)
		return s.Suggest(ctx, rec, vars["type"], vars["id"]) // suggest.go
	case "mqtt":
		return s.mqttAction(ctx, rec, title)
	case "message":
		text, _ := rec.Fields["message"].(string)
		if text == "" {
			return fail("action %s has no message to send", title)
		}
		if _, err := s.Chat.SendOn(ctx, canvas, text); err != nil {
			return fail("could not send %q for action %s: %v", text, title, err)
		}
		return Result{Text: "sent to the assistant: " + text, Change: &records.Change{Action: "ran", Component: records.ActionType, ID: id, Detail: title, Href: relate.Page(records.ActionType, id)}}
	default:
		return s.webhook(ctx, rec, title)
	}
}

// webhook makes the request the action describes and reports what came
// back. With show set, the answer is put on the canvas as a text block,
// made once and updated on every run, so a weather action is a weather
// block that stays current.
func (s *Service) webhook(ctx context.Context, rec *store.Record, title string) Result {
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
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return fail("%s failed: %v", title, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	answer := strings.TrimSpace(string(raw))
	if len([]rune(answer)) > 2000 {
		answer = string([]rune(answer)[:2000])
	}
	out := Result{Text: fmt.Sprintf("%s answered %d", title, resp.StatusCode), Answer: answer}
	if answer != "" {
		out.Text += ": " + answer
	}
	detail := fmt.Sprintf("%s (%d)", title, resp.StatusCode)
	out.Changes = append(out.Changes, records.Change{Action: "ran", Component: records.ActionType, ID: rec.ID, Detail: detail, Href: relate.Page(records.ActionType, rec.ID)})
	if resp.StatusCode >= 400 {
		out.IsErr = true
		return out
	}
	if show, _ := rec.Fields["show"].(bool); show && answer != "" {
		if c, err := s.Chat.Show(rec, title, answer); err == nil {
			out.Changes = append(out.Changes, c)
		} else {
			out.Text += ". Could not put it on the canvas: " + err.Error()
		}
	}
	return out
}

// RunAs runs an action for a person or an agent and records what it did
// under their name, returning what the action answered. When the action
// needs accepting first, proposal is the question now waiting for them.
func (s *Service) RunAs(ctx context.Context, actor, id, canvas string) (text, proposal string, err error) {
	return s.logRun(ctx, actor, s.Run(ctx, id, canvas))
}

// logRun records what a run did under whoever asked for it.
func (s *Service) logRun(ctx context.Context, actor string, r Result) (text, proposal string, err error) {
	for i := range r.Changes {
		r.Changes[i].Via, r.Changes[i].By, r.Changes[i].ByLogin = records.Via(ctx), records.VisitorOf(ctx).Who(), cmp.Or(records.VisitorOf(ctx).Login, s.Owner.Login)
		records.Record(s.Store, actor, r.Changes[i])
	}
	if r.Change != nil {
		if r.Change.Action == "proposed" {
			proposal = r.Change.ID
		}
		records.Record(s.Store, actor, *r.Change)
	}
	if r.IsErr {
		return r.Text, proposal, errors.New(r.Text)
	}
	return r.Text, proposal, nil
}

// httpClient is the client webhook actions are sent with: the service's
// own (a test points it at a server of its own), else webhooks.
func (s *Service) httpClient() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return webhooks
}
