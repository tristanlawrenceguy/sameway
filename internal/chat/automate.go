package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An action can run on its own when something happens to a record: one is
// added, one comes to match its conditions (a task done, a device on, a
// reminder ringing), or one is removed. What happened fills {{name}} in
// what the action sends: the record's title, page and fields. Then it can
// lead to another action, given what this one answered. All of it runs
// in the background, one at a time, logged as the automation, never as
// the person; the assistant asked by one tells the person what it did.
//
// Nothing runs away: an action never sets itself off within a minute for
// the same record, a chain stops after a few steps, and what the system's
// own types do (the log, the chat, the canvas) sets nothing off.

const (
	mostSteps = 5                // actions one happening may lead to
	quiet     = 60 * time.Second // between runs of one action for one record
)

type automationJob struct {
	action string
	vars   map[string]string
	step   int
	why    string // what set it off, for the log
}

type automation struct {
	jobs chan automationJob
	mu   sync.Mutex
	last map[string]time.Time // action and record → when it last ran for it
}

// StartAutomating listens for what happens to records and runs the
// actions waiting for it, until the program stops.
func (s *Service) StartAutomating() {
	if s.auto != nil {
		return
	}
	s.auto = &automation{jobs: make(chan automationJob, 256), last: map[string]time.Time{}}
	s.Store.OnChange = s.recordChanged
	go func() {
		for j := range s.auto.jobs {
			s.runAutomation(j)
		}
	}()
}

// recordChanged finds the actions waiting for what just happened.
func (s *Service) recordChanged(t *schema.Type, was, now *store.Record) {
	if s.auto == nil || t.Internal || t.Name == ActionType {
		return
	}
	at, ok := s.Store.Types().Get(ActionType)
	if !ok {
		return
	}
	if _, has := at.Field("when"); !has {
		return
	}
	actions, err := s.Store.List(ActionType, store.ListOptions{})
	if err != nil {
		return
	}
	for _, a := range actions {
		when, _ := a.Fields["when"].(string)
		if when == "" || a.Fields["what"] != t.Name {
			continue
		}
		conds, err := query.ParseAll(t, records.StringList(a.Fields["only"]))
		if err != nil {
			continue
		}
		match := func(r *store.Record) bool { return r != nil && query.Match(t, r, conds, time.Now()) }
		var rec *store.Record
		switch {
		case when == "added" && was == nil && match(now):
			rec = now
		case when == "removed" && now == nil && match(was):
			rec = was
		case when == "changed" && was != nil && now != nil && match(now) && (len(conds) == 0 || !match(was)):
			rec = now
		default:
			continue
		}
		if !s.auto.fresh(a.ID, rec.ID) {
			continue
		}
		job := automationJob{action: a.ID, vars: s.recordVars(t, rec), why: whenWords(when, t, Name(s.Store, t, rec))}
		select {
		case s.auto.jobs <- job:
		default: // more than can be kept up with; the log is not flooded
		}
	}
}

// fresh says whether an action may run for a record now, and notes it.
func (a *automation) fresh(action, rec string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := action + "/" + rec
	if time.Since(a.last[key]) < quiet {
		return false
	}
	a.last[key] = time.Now()
	return true
}

func whenWords(when string, t *schema.Type, title string) string {
	verb := map[string]string{"added": "was added", "changed": "came to match", "removed": "was removed"}[when]
	return fmt.Sprintf("%s %s %s", schema.Words(t.Name), title, verb)
}

// runAutomation runs one action, as the automation, and what it leads to.
func (s *Service) runAutomation(j automationJob) {
	rec, err := s.Store.Get(ActionType, j.action)
	if err != nil {
		return
	}
	title, _ := rec.Fields["title"].(string)
	ctx := context.WithValue(context.Background(), automationKey{}, title)
	r := s.runRecord(ctx, filled(rec, j.vars), "")
	for i := range r.changes {
		r.changes[i].By, r.changes[i].Via = title, "because "+j.why
		Record(s.Store, "system", r.changes[i])
	}
	if r.change != nil {
		r.change.By, r.change.Via = title, "because "+j.why
		Record(s.Store, "system", *r.change)
	}
	if r.isErr {
		Record(s.Store, "system", Change{Action: "failed", Component: ActionType, ID: rec.ID, Detail: title + ": " + r.text, Via: "because " + j.why})
		return
	}
	if kind, _ := rec.Fields["kind"].(string); kind == "message" && s.Tell != nil {
		s.Tell(title, "The assistant did this on its own because "+j.why+". Its reply is in the conversation.", "/chat")
	}
	next, _ := rec.Fields["then"].(string)
	if next == "" || next == rec.ID || j.step+1 >= mostSteps {
		return
	}
	vars := map[string]string{}
	for k, v := range j.vars {
		vars[k] = v
	}
	vars["result"] = r.answer
	if vars["result"] == "" {
		vars["result"] = r.text
	}
	select {
	case s.auto.jobs <- automationJob{action: next, vars: vars, step: j.step + 1, why: title + " ran"}:
	default:
	}
}

type automationKey struct{}

// automatedBy is the action a turn was asked for by, or "".
func automatedBy(ctx context.Context) string {
	name, _ := ctx.Value(automationKey{}).(string)
	return name
}

// recordVars are what a record fills {{name}} with: its title, type, id,
// page, and each field as it reads (a ref by its title).
func (s *Service) recordVars(t *schema.Type, rec *store.Record) map[string]string {
	vars := map[string]string{"title": Name(s.Store, t, rec), "type": t.Name, "id": rec.ID, "page": "/t/" + t.Name + "/" + rec.ID}
	for _, f := range t.Fields {
		v := rec.Fields[f.Name]
		switch {
		case v == nil:
			vars[f.Name] = ""
		case f.RefTo() != "":
			var names []string
			for _, id := range idsOf(v) {
				if to, ok := s.Store.Types().Get(f.RefTo()); ok {
					if r, err := s.Store.Get(to.Name, id); err == nil {
						names = append(names, Name(s.Store, to, r))
					}
				}
			}
			vars[f.Name] = strings.Join(names, ", ")
		case f.Type == "list":
			vars[f.Name] = strings.Join(records.StringList(v), ", ")
		default:
			vars[f.Name] = fmt.Sprint(v)
		}
	}
	return vars
}

var placeholder = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// filled puts what happened into a copy of an action: escaped for an
// address in url, for JSON in a body or payload that is JSON, as it is in
// a message. A command is run exactly as the person accepted it, so it is
// never filled.
func filled(action *store.Record, vars map[string]string) *store.Record {
	if len(vars) == 0 {
		return action
	}
	out := *action
	out.Fields = map[string]any{}
	for k, v := range action.Fields {
		out.Fields[k] = v
	}
	put := func(field string, esc func(string) string) {
		text, _ := out.Fields[field].(string)
		out.Fields[field] = placeholder.ReplaceAllStringFunc(text, func(m string) string {
			name := placeholder.FindStringSubmatch(m)[1]
			v, ok := vars[name]
			if !ok {
				return m
			}
			return esc(v)
		})
	}
	put("url", url.QueryEscape)
	for _, f := range []string{"body", "payload"} {
		text, _ := out.Fields[f].(string)
		if t := strings.TrimSpace(text); strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[") {
			put(f, func(v string) string { b, _ := json.Marshal(v); return string(b[1 : len(b)-1]) })
		} else {
			put(f, func(v string) string { return v })
		}
	}
	if msg, _ := out.Fields["message"].(string); placeholder.MatchString(msg) {
		put("message", func(v string) string { return v })
		// What filled it came from a record or a request, written by
		// anyone: the assistant reads it as data (provenance.go).
		out.Fields["message"] = out.Fields["message"].(string) + "\n\n(What was filled into this message came from what set it off: " + Untrusted + ".)"
	}
	return &out
}

func idsOf(v any) []string {
	if s, ok := v.(string); ok && s != "" {
		return []string{s}
	}
	return records.StringList(v)
}

// RunAsWith is RunAs with what set the action off filled in: a request
// to its hook, say. Each value is a field of what was sent.
func (s *Service) RunAsWith(ctx context.Context, actor, id, canvas string, vars map[string]string) (text, proposal string, err error) {
	rec, gerr := s.Store.Get(ActionType, id)
	if gerr != nil {
		return "", "", fmt.Errorf("no action with id %s", id)
	}
	return s.logRun(ctx, actor, s.runRecord(ctx, filled(rec, vars), canvas))
}
