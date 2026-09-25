package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The workspace's content types travel between the computers that host it
// the way records do: each type is a record of SchemaType, named after
// it, and every part of it that can change on its own is a field of that
// record, so changes to different parts merge. A field's definition is
// one; how it is labelled, whether it is hidden and whether it was
// deleted are one each (the latest wins); every choice a pick-list offers
// is one of its own, so two people adding choices both keep theirs.

// SchemaType is what a content type's definition is stamped under.
const SchemaType = "_schema"

const fieldType = "_type"

// Schema is a content type as the other copies have it: the type, with
// its fields, choices, labels and what is hidden, and what was deleted.
type Schema struct {
	Type    *schema.Type
	Deleted []string // fields deleted
	Gone    bool     // the whole type deleted
}

// syncsSchema says whether a type's definition travels: the system's own
// types come with the program, and local types stay.
func (s *Store) syncsSchema(t *schema.Type) bool {
	return !t.Internal && s.shared(t.Name)
}

// StampSchema stamps what is new in a type's definition.
func (s *Store) StampSchema(t *schema.Type) {
	if !s.syncsSchema(t) {
		return
	}
	meta := *t
	meta.Fields = nil
	s.stampDef(t.Name, fieldType, meta)
	s.stampDef(t.Name, fieldType+"|deleted", false)
	for _, f := range t.Fields {
		core := f
		core.Values, core.Labels, core.Label, core.Hidden = nil, nil, "", false
		s.stampDef(t.Name, f.Name, core)
		s.stampDef(t.Name, f.Name+"|label", f.Label)
		s.stampDef(t.Name, f.Name+"|hidden", f.Hidden)
		s.stampDef(t.Name, f.Name+"|deleted", false)
		for _, v := range f.Values {
			s.stampDef(t.Name, f.Name+"|choice|"+v, f.Labels[v])
		}
	}
}

// StampDeleted stamps that a field (or, with field "", the whole type)
// was deleted here, so it is deleted on the other copies too.
func (s *Store) StampDeleted(typeName, field string) {
	if field == "" {
		s.stampDef(typeName, fieldType+"|deleted", true)
		return
	}
	s.stampDef(typeName, field+"|deleted", true)
}

func (s *Store) stampDef(typeName, field string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	var have string
	s.db.QueryRow(`SELECT value FROM _state WHERE type = ? AND id = ? AND field = ?`, SchemaType, typeName, field).Scan(&have)
	// A choice is stamped whatever its label, since being there is what it
	// says; other parts are not stamped for being empty or false, their
	// default, until they have been something else.
	choice := strings.Contains(field, "|choice|")
	if have == string(raw) || !choice && have == "" && (string(raw) == "false" || string(raw) == `""`) {
		return
	}
	clock := s.clock.next(s.origin)
	s.db.Exec(`INSERT OR REPLACE INTO _state (type, id, field, value, clock) VALUES (?, ?, ?, ?, ?)`, SchemaType, typeName, field, string(raw), clock)
	s.db.Exec(`INSERT OR REPLACE INTO _seen (origin, clock) VALUES (?, ?)`, s.origin, clock)
}

// readSchema is a type as _state has it.
func (s *Store) readSchema(typeName string) (*Schema, error) {
	rows, err := s.db.Query(`SELECT field, value, clock FROM _state WHERE type = ? AND id = ? ORDER BY clock`, SchemaType, typeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &Schema{Type: &schema.Type{}}
	fields := map[string]*schema.Field{}
	var order []string
	extra := map[string]map[string]string{}
	choices := map[string][]string{} // each field's choices, in the order stamped
	for rows.Next() {
		var f, v, clock string
		if err := rows.Scan(&f, &v, &clock); err != nil {
			return nil, err
		}
		switch {
		case f == fieldType:
			json.Unmarshal([]byte(v), out.Type)
		case f == fieldType+"|deleted":
			json.Unmarshal([]byte(v), &out.Gone)
		case !strings.Contains(f, "|"):
			var fd schema.Field
			if json.Unmarshal([]byte(v), &fd) == nil && fd.Name != "" {
				fields[fd.Name], order = &fd, append(order, fd.Name)
			}
		default:
			name, part, _ := strings.Cut(f, "|")
			if extra[name] == nil {
				extra[name] = map[string]string{}
			}
			extra[name][part] = v
			if strings.HasPrefix(part, "choice|") {
				choices[name] = append(choices[name], part)
			}
		}
	}
	for _, name := range order {
		fd := fields[name]
		var deleted bool
		json.Unmarshal([]byte(extra[name]["deleted"]), &deleted)
		if deleted {
			out.Deleted = append(out.Deleted, name)
			continue
		}
		json.Unmarshal([]byte(extra[name]["label"]), &fd.Label)
		json.Unmarshal([]byte(extra[name]["hidden"]), &fd.Hidden)
		fd.Values, fd.Labels = nil, nil
		for _, part := range choices[name] {
			v := strings.TrimPrefix(part, "choice|")
			fd.Values = append(fd.Values, v)
			var label string
			if json.Unmarshal([]byte(extra[name][part]), &label); label != "" {
				if fd.Labels == nil {
					fd.Labels = map[string]string{}
				}
				fd.Labels[v] = label
			}
		}
		out.Type.Fields = append(out.Type.Fields, *fd)
	}
	return out, rows.Err()
}

// adoptSchema offers a type as the other copies know it to OnSchema, which
// makes this workspace's the same, then writes the records of that type
// that were waiting for it.
func (s *Store) adoptSchema(typeName string) error {
	if s.OnSchema == nil {
		return nil
	}
	sc, err := s.readSchema(typeName)
	if err != nil || sc.Type.Name == "" {
		return err // the type's own stamp has not arrived yet
	}
	if err := s.OnSchema(sc); err != nil {
		return fmt.Errorf("content type %s from another computer: %w", typeName, err)
	}
	if sc.Gone {
		return nil
	}
	return s.waiting(typeName)
}

// waiting writes every record of a type that arrived before this copy
// knew the type or one of its fields.
func (s *Store) waiting(typeName string) error {
	rows, err := s.db.Query(`SELECT DISTINCT id FROM _state WHERE type = ?`, typeName)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := s.materialize(typeName, id); err != nil {
			return err
		}
	}
	return nil
}

// adoptAll takes the types that arrived, and any still waiting from
// before (one that points at a type that had not come yet), in as many
// passes as it takes for each to find what it points at.
func (s *Store) adoptAll(touched map[[2]string]bool) {
	todo := map[string]bool{}
	for k := range touched {
		if k[0] == SchemaType {
			todo[k[1]] = true
		}
	}
	rows, err := s.db.Query(`SELECT DISTINCT id FROM _state WHERE type = ?`, SchemaType)
	if err == nil {
		for rows.Next() {
			var id string
			rows.Scan(&id)
			if _, known := s.types.Get(id); !known {
				todo[id] = true
			}
		}
		rows.Close()
	}
	for len(todo) > 0 {
		progress := false
		for name := range todo {
			if s.adoptSchema(name) == nil {
				delete(todo, name)
				progress = true
			}
		}
		if !progress {
			return // tried again when more arrives
		}
	}
}
