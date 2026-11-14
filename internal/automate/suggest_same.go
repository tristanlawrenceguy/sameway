package automate

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A suggest action with update set is shown the records of its kind
// already kept, so the second receipt from a service changes the one kept
// (a new price, the next renewal, cancelled) rather than making another;
// what says nothing new suggests nothing.

const sameSystem = `
When the record is about a %s already kept (listed after the record), give its id as "same" and in fields only what the record says that differs from it; when it differs in nothing, say make false. Otherwise leave "same" out.`

// keptMost is how many kept records are shown, newest first.
const keptMost = 60

// keptToChange are the records of the kind an action with update makes,
// by id, newest first; none for an action without it.
func (s *Service) keptToChange(action *store.Record, t *schema.Type) map[string]*store.Record {
	if on, _ := action.Fields["update"].(bool); !on {
		return nil
	}
	recs, err := s.Store.List(t.Name, store.ListOptions{OrderBy: "updated_at", Desc: true, Limit: keptMost})
	if err != nil {
		return nil
	}
	out := map[string]*store.Record{}
	for _, r := range recs {
		out[r.ID] = r
	}
	return out
}

// keptWords is each kept record in a line: its id and the fields the
// model may fill, as they are.
func (s *Service) keptWords(t *schema.Type, kept map[string]*store.Record) string {
	var lines []string
	for id, r := range kept {
		fields := map[string]any{}
		for _, f := range t.Shown() {
			if v, ok := r.Fields[f.Name]; ok && v != nil && v != "" && f.Type != "markdown" && !f.ReadOnly {
				fields[f.Name] = v
			}
		}
		said, _ := json.Marshal(fields)
		lines = append(lines, "- "+id+": "+trim.Line(string(said), 300))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// suggestChange puts to the person a change to the kept record with what
// is new, leaving out what it already says; with nothing new, nothing.
func (s *Service) suggestChange(action *store.Record, from *schema.Type, rec *store.Record, t *schema.Type, same *store.Record, fields map[string]any, why string) Result {
	name := records.Name(s.Store, from, rec)
	for k, v := range fields {
		if fmt.Sprint(same.Fields[k]) == fmt.Sprint(v) || k == t.Title {
			delete(fields, k)
		}
	}
	if len(fields) == 0 {
		return Result{Text: name + ": nothing new for " + records.Name(s.Store, t, same)}
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var said []string
	for _, k := range keys {
		label := k
		if f, ok := t.Field(k); ok {
			label = strings.ToLower(f.Display())
		}
		said = append(said, label+" "+s.fieldWords(t, k, fields[k]))
	}
	summary := fmt.Sprintf("Change %s from %q: %s?", records.Name(s.Store, t, same), name, strings.Join(said, ", "))
	done, err := s.Apply(records.Op{Type: records.ProposalType, After: map[string]any{"summary": summary, "detail": strings.TrimSpace(why),
		"action": map[string]any{"tool": "update_record", "type": t.Name, "id": same.ID, "fields": fields}, "state": "pending",
		"from": from.Name + "/" + rec.ID, "by_action": action.ID, "yes": "Change it", "no": "No"}})
	if err != nil {
		return fail("could not put the suggestion: %v", err)
	}
	p := done[0]
	return Result{Text: "suggested: " + summary, Answer: p.ID, Change: &records.Change{Action: "proposed", ID: p.ID, Detail: trim.Line(summary, 80), Ops: done}}
}

// fieldWords is a suggested value as a person reads it: a day in full,
// yes or no, else as it is.
func (s *Service) fieldWords(t *schema.Type, name string, v any) string {
	if f, ok := t.Field(name); ok {
		switch f.Type {
		case "datetime":
			if str, ok := v.(string); ok {
				return when.Text(str, s.Now().Location(), false)
			}
		case "bool":
			if v == true {
				return "yes"
			}
			return "no"
		}
	}
	return fmt.Sprint(v)
}
