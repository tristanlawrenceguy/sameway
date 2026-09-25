package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A content type changes after it is made, by asking: a pick-list gets a
// choice, a field or a choice is called something else, a field or a type
// is hidden or shown, and, when the person agrees, deleted. Deleting takes
// what it held away on every computer that hosts the workspace, so it is
// always asked, with hiding offered first: that keeps everything.

// Reshaper makes those changes; the app is one. Nil where the workspace
// cannot change its schema from here.
type Reshaper interface {
	AddChoice(typeName, field, value, label string) (*schema.Type, error)
	Relabel(typeName, field, choice, label string) (*schema.Type, error)
	SetHidden(typeName, field string, hidden bool) (*schema.Type, error)
	RemoveField(typeName, field string) (*schema.Type, error)
	RemoveType(typeName string) error
}

var changeFieldTool = llm.Tool{
	Name:        "change_field",
	Description: "Change a content type the person already has: add_choice gives a pick-list (enum) another choice (value, and label for how it reads); label renames how a field, or with value one of its choices, is shown (its name and what is stored stay); hide takes a field, or with no field the whole type, off the pages and out of your hands while keeping everything it holds; show brings it back; delete removes a field, or with no field the whole type and its records, on every computer that hosts the workspace. Delete is always put to the person as a question with hiding offered first; nothing is deleted until they choose. When someone asks to remove something, offer hiding.",
	Schema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"type", "change"}, "properties": map[string]any{
		"type":   map[string]any{"type": "string", "description": "The content type."},
		"field":  map[string]any{"type": "string", "description": "The field; leave out to hide, show or delete the whole type."},
		"change": map[string]any{"type": "string", "enum": []string{"add_choice", "label", "hide", "show", "delete"}},
		"value":  map[string]any{"type": "string", "description": "add_choice: the choice. label: the choice to rename, when renaming a choice."},
		"label":  map[string]any{"type": "string", "description": "How it reads: the new label, or the new choice's label."},
	}},
}

type reshapeArgs struct {
	Type   string `json:"type"`
	Field  string `json:"field"`
	Change string `json:"change"`
	Value  string `json:"value"`
	Label  string `json:"label"`
	Agreed bool   `json:"agreed"`
}

// reshapeCall is the assistant's call; a proposal's answer runs it agreed.
func (s *Service) reshapeCall(raw json.RawMessage) toolResult {
	var a reshapeArgs
	json.Unmarshal(raw, &a)
	return s.reshape(a)
}

func (s *Service) reshape(a reshapeArgs) toolResult {
	if s.Reshape == nil {
		return fail("this workspace cannot change its schema from here")
	}
	a.Type, a.Field = strings.ToLower(strings.TrimSpace(a.Type)), strings.TrimSpace(a.Field)
	what := a.Type
	if a.Field != "" {
		what = a.Field + " on " + a.Type
	}
	var err error
	action := ""
	switch a.Change {
	case "add_choice":
		_, err = s.Reshape.AddChoice(a.Type, a.Field, a.Value, a.Label)
		action, what = "added choice", a.Value+" to "+what
	case "label":
		_, err = s.Reshape.Relabel(a.Type, a.Field, a.Value, a.Label)
		action, what = "relabelled", what+" as "+a.Label
	case "hide", "show":
		_, err = s.Reshape.SetHidden(a.Type, a.Field, a.Change == "hide")
		action = map[string]string{"hide": "hid", "show": "showed"}[a.Change]
	case "delete":
		if !a.Agreed {
			return s.askDelete(a)
		}
		if a.Field == "" {
			err = s.Reshape.RemoveType(a.Type)
		} else {
			_, err = s.Reshape.RemoveField(a.Type, a.Field)
		}
		action = "deleted"
	default:
		return fail("change is add_choice, label, hide, show or delete, not %q", a.Change)
	}
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: action + " " + what, change: &Change{Action: action, Component: "field", Detail: what, Href: "/t/" + a.Type}}
}

// askDelete puts deleting to the person: delete, hide instead, or keep.
func (s *Service) askDelete(a reshapeArgs) toolResult {
	t, ok := s.Store.Types().Get(a.Type)
	if !ok {
		return fail("there is no content type %q", a.Type)
	}
	n, held := 0, 0
	if recs, err := s.Store.List(t.Name, store.ListOptions{}); err == nil {
		n = len(recs)
		for _, r := range recs {
			if a.Field != "" && r.Fields[a.Field] != nil && r.Fields[a.Field] != "" {
				held++
			}
		}
	}
	q := question{ask: fmt.Sprintf("Delete %s and every %s in it?", t.Name, t.Name), yes: "Delete it", no: "Keep it"}
	q.detail = fmt.Sprintf("All %d would be gone, on every computer that hosts this workspace, and it can't be undone. Hiding it instead takes it off the pages and keeps everything, until you show it again.", n)
	if a.Field != "" {
		label := a.Field
		if f, ok := t.Field(a.Field); ok && f.Label != "" {
			label = f.Label
		}
		q.ask = fmt.Sprintf("Delete %s from every %s?", label, t.Name)
		q.detail = fmt.Sprintf("%d of %d have something in it; that would be cleared, on every computer that hosts this workspace, and it can't be undone. Hiding it instead takes it off the pages and keeps what they hold, until you show it again.", held, n)
	}
	del := map[string]any{"tool": "change_field", "type": a.Type, "field": a.Field, "change": "delete", "agreed": true}
	hide := map[string]any{"tool": "change_field", "type": a.Type, "field": a.Field, "change": "hide"}
	r := s.ask(q, del)
	if !r.isErr && r.change != nil {
		s.Store.Update(ProposalType, r.change.ID, map[string]any{"instead": "Hide it instead", "instead_action": hide})
		r.text += " They can also hide it instead."
	}
	return r
}

// Instead answers a question with its other choice, such as hiding a field
// rather than deleting it: that action runs, as the person's.
func (s *Service) Instead(id string) error {
	rec, err := s.Store.Get(ProposalType, id)
	if err != nil {
		return err
	}
	if rec.Fields["state"] != "pending" {
		return fmt.Errorf("that question has been answered")
	}
	action, _ := rec.Fields["instead_action"].(map[string]any)
	if action == nil {
		return fmt.Errorf("that question has no other answer")
	}
	raw, _ := json.Marshal(action)
	tool, _ := action["tool"].(string)
	result := s.runAgreed(llm.ToolCall{Name: tool, Args: raw})
	if result.isErr {
		return fmt.Errorf("%s", result.text)
	}
	s.Store.Update(ProposalType, id, map[string]any{"state": "accepted"})
	label, _ := rec.Fields["instead"].(string)
	Record(s.Store, "human", Change{Action: "chose", Detail: truncate(label, 80)})
	if result.change != nil {
		Record(s.Store, "assistant", *result.change)
	}
	return nil
}
