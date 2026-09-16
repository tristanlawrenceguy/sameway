package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
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
		{Name: "find_records", Description: "List records of a type, newest first, to get their ids: all of them, or those whose title contains the query.",
			Schema: obj(map[string]any{
				"type":  typeArg,
				"query": map[string]any{"type": "string", "description": "Text the title should contain. Leave empty for the newest records."},
				"limit": map[string]any{"type": "integer", "description": "How many to list. Defaults to 10."},
			}, "type")},
	}
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
		return fail("%v. Fix the fields and call create_record again; the %s schema is in the catalogue.", err, t.Name)
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
		return fail("nothing to change: pass the fields to change and their new values")
	}
	if _, err := s.Store.Get(t.Name, id); err != nil {
		return fail("no %s with id %s. Use find_records to get the id", t.Name, id)
	}
	rec, err := s.Store.Update(t.Name, id, fields)
	if err != nil {
		return fail("%v. Fix the fields and call update_record again; the %s schema is in the catalogue.", err, t.Name)
	}
	title := recordTitle(t, rec)
	return toolResult{
		text:   fmt.Sprintf("updated %s %s: %q, at /t/%s/%s.", t.Name, rec.ID, title, t.Name, rec.ID),
		change: &Change{Action: "updated", Component: t.Name, ID: rec.ID, Detail: title, Href: "/t/" + t.Name + "/" + rec.ID},
	}
}

func (s *Service) findRecords(typeName, query string, limit int) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if limit <= 0 {
		limit = 10
	}
	recs, err := s.Store.List(t.Name, store.ListOptions{OrderBy: "created_at", Desc: true})
	if err != nil {
		return fail("could not read %s records: %v", t.Name, err)
	}
	query = strings.ToLower(strings.TrimSpace(query))
	var lines []string
	for _, rec := range recs {
		title := recordTitle(t, rec)
		if query != "" && !strings.Contains(strings.ToLower(title), query) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s\t%s", rec.ID, title))
		if len(lines) == limit {
			break
		}
	}
	if len(lines) == 0 {
		if query == "" {
			return toolResult{text: fmt.Sprintf("there are no %s records yet", t.Name)}
		}
		return toolResult{text: fmt.Sprintf("no %s has %q in its title", t.Name, query)}
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
