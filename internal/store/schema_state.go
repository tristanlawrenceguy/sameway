package store

import (
	"encoding/json"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The workspace's content types travel between the computers that host it
// the way records do: each type is a record of SchemaType, named after
// it, whose fields are the type itself (_type) and each of its fields, so
// two computers adding different fields to one type both keep theirs.
// A schema only ever grows (a type or a field is added, never changed or
// removed), so a copy takes only what it lacks, through OnSchema, and a
// copy running an older program can never take anything away.

// SchemaType is what a content type's definition is stamped under.
const SchemaType = "_schema"

const fieldType = "_type"

// syncsSchema says whether a type's definition travels: the system's own
// types come with the program, and local types stay.
func (s *Store) syncsSchema(t *schema.Type) bool {
	return !t.Internal && s.shared(t.Name)
}

// StampSchema stamps what is new in a type's definition: the type itself
// the first time, then each field this copy has not stamped as it is now.
func (s *Store) StampSchema(t *schema.Type) {
	if !s.syncsSchema(t) {
		return
	}
	meta := *t
	meta.Fields = nil
	s.stampDef(t.Name, fieldType, meta)
	for _, f := range t.Fields {
		s.stampDef(t.Name, f.Name, f)
	}
}

func (s *Store) stampDef(typeName, field string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	var have string
	s.db.QueryRow(`SELECT value FROM _state WHERE type = ? AND id = ? AND field = ?`, SchemaType, typeName, field).Scan(&have)
	if have == string(raw) {
		return
	}
	clock := s.clock.next(s.origin)
	s.db.Exec(`INSERT OR REPLACE INTO _state (type, id, field, value, clock) VALUES (?, ?, ?, ?, ?)`, SchemaType, typeName, field, string(raw), clock)
	s.db.Exec(`INSERT OR REPLACE INTO _seen (origin, clock) VALUES (?, ?)`, s.origin, clock)
}

// adoptSchema offers a type as the other copies know it to OnSchema, which
// adds to this workspace what it lacks, then writes the records of that
// type that were waiting for it.
func (s *Store) adoptSchema(typeName string) error {
	if s.OnSchema == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT field, value FROM _state WHERE type = ? AND id = ?`, SchemaType, typeName)
	if err != nil {
		return err
	}
	var t schema.Type
	var fields []schema.Field
	for rows.Next() {
		var f, v string
		if err := rows.Scan(&f, &v); err != nil {
			rows.Close()
			return err
		}
		if f == fieldType {
			json.Unmarshal([]byte(v), &t)
			continue
		}
		var fd schema.Field
		if json.Unmarshal([]byte(v), &fd) == nil && fd.Name != "" {
			fields = append(fields, fd)
		}
	}
	rows.Close()
	if t.Name == "" {
		return nil // the type's own stamp has not arrived yet
	}
	t.Fields = fields
	if err := s.OnSchema(&t); err != nil {
		return fmt.Errorf("content type %s from another computer: %w", typeName, err)
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
