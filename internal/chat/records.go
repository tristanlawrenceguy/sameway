package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Records are the person's content: the notes, the tasks, whatever the
// workspace's schema declares. The canvas tools place components; these
// tools make and change the things a person then finds on /t/<type>. They
// are generated from the schema, so a workspace that adds a content type
// gives the assistant the tool for it without a line of code.

// contentTypes lists the types the assistant may write to: everything the
// schema declares except the system's own (messages, blocks, activity).
func (s *Service) contentTypes() []*schema.Type {
	var out []*schema.Type
	for _, t := range s.Store.Types().Types {
		switch {
		case t.Internal, t.Name == MessageType, t.Name == BlockType, t.Name == ActivityType:
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *Service) typeNames() []string {
	var names []string
	for _, t := range s.contentTypes() {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return names
}

// recordTools are offered only when the workspace has something to write to.
func (s *Service) recordTools() []llm.Tool {
	names := s.typeNames()
	if len(names) == 0 {
		return nil
	}
	obj := func(props map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	typeArg := map[string]any{"type": "string", "enum": names, "description": "A content type from the catalogue."}
	return []llm.Tool{
		{Name: "create_record", Description: "Make a record of a content type: a note, a task, whatever the workspace declares. It appears on its own page at /t/<type> and in the listing there. Fields must match the type's schema in the catalogue. Returns the new record's id and page.",
			Schema: obj(map[string]any{
				"type":   typeArg,
				"fields": map[string]any{"type": "object", "description": "Field values matching the type's schema. Leave a field out to take its default."},
			}, "type", "fields")},
		{Name: "update_record", Description: "Change fields on a record that exists. Only the fields given change. Use find_records first to get the id.",
			Schema: obj(map[string]any{
				"type":   typeArg,
				"id":     map[string]any{"type": "string", "description": "The record's id, from find_records or from a page URL /t/<type>/<id>."},
				"fields": map[string]any{"type": "object", "description": "The fields to change and their new values."},
			}, "type", "id", "fields")},
		{Name: "find_records", Description: "List records of a type to get their ids: all of them, those whose title contains the query, or those matching where. The same where and order a collection block takes.",
			Schema: obj(map[string]any{
				"type":  typeArg,
				"query": map[string]any{"type": "string", "description": "Text the title should contain. Leave empty for every record."},
				"where": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Conditions that must all hold. " + query.Grammar},
				"order": map[string]any{"type": "string", "description": "A field, or -field for the largest or newest first. Newest first when left out."},
				"limit": map[string]any{"type": "integer", "description": "How many to list. Defaults to 10."},
			}, "type")},
		{Name: "get_record", Description: "Read one record with every field, by id: a note's body, a file's text. Use it before answering from what a record says.",
			Schema: obj(map[string]any{
				"type": typeArg,
				"id":   map[string]any{"type": "string", "description": "The record's id, from find_records or from a page URL /t/<type>/<id>."},
			}, "type", "id")},
	}
}

// getRecord gives the model a record's fields, so it can answer from what
// a note or a file says rather than from its title alone.
func (s *Service) getRecord(typeName, id string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s with id %s. Use find_records to get an id", t.Name, id)
	}
	raw, err := json.MarshalIndent(map[string]any{"id": rec.ID, "type": t.Name, "page": "/t/" + t.Name + "/" + rec.ID, "fields": rec.Fields}, "", "  ")
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: string(raw)}
}

func (s *Service) contentType(name string) (*schema.Type, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, t := range s.contentTypes() {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("unknown content type %q. The workspace has: %s", name, strings.Join(s.typeNames(), ", "))
}

// recordTitle is what a record is called: its title field, or its id.
func recordTitle(t *schema.Type, rec *store.Record) string {
	if v, ok := rec.Fields[t.Title].(string); ok && strings.TrimSpace(v) != "" {
		return truncate(v, 60)
	}
	return rec.ID
}

func (s *Service) createRecord(typeName string, fields map[string]any) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if fields == nil {
		fields = map[string]any{}
	}
	rec, err := s.Store.Create(t.Name, fields)
	if err != nil {
		return fail("%v — fix these and try again", err)
	}
	title := recordTitle(t, rec)
	return toolResult{
		text:   fmt.Sprintf("created %s %s: %q. The person can open it at /t/%s/%s.", t.Name, rec.ID, title, t.Name, rec.ID),
		change: &Change{Action: "created", Component: t.Name, ID: rec.ID, Detail: title, Href: "/t/" + t.Name + "/" + rec.ID},
	}
}

func (s *Service) updateRecord(typeName, id string, fields map[string]any) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if len(fields) == 0 {
		return fail("pass the fields you want to change and their new values")
	}
	was, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s there — use find_records to look it up", t.Name)
	}
	rec, err := s.Store.Update(t.Name, id, fields)
	if err != nil {
		return fail("%v — fix these and try again", err)
	}
	title := recordTitle(t, rec)
	return toolResult{
		text:   fmt.Sprintf("updated %s %s: %q, at /t/%s/%s.", t.Name, rec.ID, title, t.Name, rec.ID),
		change: &Change{Action: "updated", Component: t.Name, ID: rec.ID, Detail: title, Href: "/t/" + t.Name + "/" + rec.ID, Before: was.Fields},
	}
}

func (s *Service) findRecords(typeName, words string, where []string, order string, limit int) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if limit <= 0 {
		limit = 10
	}
	recs, err := query.Filter(s.Store, t, where, order, 0, time.Now())
	if err != nil {
		return fail("%v", err)
	}
	words = strings.ToLower(strings.TrimSpace(words))
	var lines []string
	for _, rec := range recs {
		title := recordTitle(t, rec)
		if words != "" && !strings.Contains(strings.ToLower(title), words) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s\t%s", rec.ID, title))
		if len(lines) == limit {
			break
		}
	}
	if len(lines) == 0 {
		if words == "" && len(where) == 0 {
			return toolResult{text: fmt.Sprintf("there are no %s records yet", t.Name)}
		}
		return toolResult{text: fmt.Sprintf("no %s matches %s", t.Name, strings.TrimSpace(strings.Join(append(where, words), " ")))}
	}
	return toolResult{text: fmt.Sprintf("%s records, newest first (id, title):\n%s", t.Name, strings.Join(lines, "\n"))}
}

// contentCatalogue renders the content types for the system prompt: what a
// person can have, and the fields each takes.
func (s *Service) contentCatalogue() string {
	types := s.contentTypes()
	if len(types) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nContent types (name: description, then fields schema). A record of one of these is what the person finds on its page at /t/<type>; make it with create_record, never as a card on the canvas:\n")
	for _, t := range types {
		raw, _ := json.Marshal(t.JSONSchema())
		fmt.Fprintf(&b, "\n%s: %s\n%s\n", t.Name, t.Description, raw)
	}
	return b.String()
}

// deleteRecord is not a tool: a record goes when a person deletes it, or
// when its creation is undone. Either way the log keeps what it was.
func (s *Service) deleteRecord(typeName, id string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s there", t.Name)
	}
	if err := s.Store.Delete(t.Name, id); err != nil {
		return fail("delete failed for %s %s: %v", t.Name, id, err)
	}
	return toolResult{
		text:   fmt.Sprintf("deleted %s %s", t.Name, id),
		change: &Change{Action: "deleted", Component: t.Name, ID: id, Detail: recordTitle(t, rec), Before: rec.Fields},
	}
}
