package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// What someone asks the assistant for beyond what they may do is not
// refused: it becomes a request to the owner. The call is kept as it was
// made, and put on Today in Sameway's words, from what the call would
// really do, as a suggestion of its own; the owner's yes runs exactly
// that call, as the owner, and nothing changes before it. A permission is
// no longer a wall but a question for whoever may answer it.

// ByRequest marks a suggestion that is someone's request.
const ByRequest = "request"

// requestable says whether a tool someone may not use may be asked of
// the owner instead: making a record or changing one, the changes a
// request can say exactly in words, so the owner knows what a yes does.
func requestable(name string) bool { return name == "create_record" || name == "update_record" }

// request puts a call someone may not make to the owner.
func (s *Service) request(call llm.ToolCall) toolResult {
	var a struct {
		Type   string         `json:"type"`
		ID     string         `json:"id"`
		Fields map[string]any `json:"fields"`
	}
	json.Unmarshal(call.Args, &a)
	what := s.requestWords(call.Name, a.Type, a.ID, a.Fields)
	if what == "" {
		return fail("that could not be put to the owner: say what to make or change, of a kind of record that is here")
	}
	if err := s.requestFits(call.Name, a.Type, a.ID, a.Fields); err != nil { // request_fits.go
		return fail("that was not put to the owner, as a yes could not make it: %v. Fix it and ask again", err)
	}
	if s.always(call.Name, a.Type) { // request_always.go
		return s.madeAlways(call, what)
	}
	action := map[string]any{}
	json.Unmarshal(call.Args, &action)
	action["tool"] = call.Name
	who := s.who.Who()
	if s.who.Login != "" && !strings.HasPrefix(s.who.Login, "device:") {
		who += " (" + s.who.Login + ")"
	}
	summary := trim.Line(who+" asks you to "+what+". Make it?", 300)
	token := requestToken()
	if _, err := s.Apply(records.Op{Type: records.ProposalType, After: map[string]any{"summary": summary, "detail": "Asked through the assistant",
		"action": action, "state": "pending", "by_action": ByRequest, "yes": "Make it", "no": "No",
		"asked": what, "asked_by": s.forWhom(), "token": token}}); err != nil {
		return fail("could not put it to the owner: %v", err)
	}
	return toolResult{text: "Nothing has changed: it is not theirs to change, so it was put to the owner, who sees: " + summary +
		" Tell them that, in those words, and that they will hear the answer here; what became of it is also at " + RequestPath(token) + "."}
}

// requestWords is what a call would do, as the owner reads it, or "" when
// it names nothing they could say yes to.
func (s *Service) requestWords(tool, typeName, id string, fields map[string]any) string {
	t, err := s.requestType(tool, typeName)
	if err != nil {
		return ""
	}
	said := s.fieldsWords(t, fields)
	switch tool {
	case "create_record":
		if said == "" {
			return ""
		}
		return "make " + withA(schema.Words(t.Name)) + ": " + said
	case "update_record":
		rec, err := s.Store.Get(t.Name, id)
		if err != nil || said == "" {
			return ""
		}
		return fmt.Sprintf("change the %s %q: %s", schema.Words(t.Name), records.Name(s.Store, t, rec), said)
	}
	return ""
}

// fieldsWords is fields as a person reads them: the title first, then each
// by its label, a day in full.
func (s *Service) fieldsWords(t *schema.Type, fields map[string]any) string {
	var parts []string
	if v, ok := fields[t.Title]; ok {
		parts = append(parts, fmt.Sprintf("%q", fmt.Sprint(v)))
	}
	var names []string
	for name := range fields {
		if name != t.Title {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		f, ok := t.Field(name)
		if !ok {
			continue
		}
		v := fmt.Sprint(fields[name])
		if str, ok := fields[name].(string); ok && f.Type == "datetime" {
			v = when.Text(str, s.Now().Location(), false)
		}
		parts = append(parts, strings.ToLower(f.Display())+" "+trim.Line(v, 80))
	}
	return strings.Join(parts, ", ")
}
