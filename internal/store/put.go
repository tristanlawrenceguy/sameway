package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Restore puts back a record that was deleted, under the id it had, so
// everything that pointed at it points at it again. It is how an undo
// works, and it refuses an id that is in use.
func (s *Store) Restore(typeName, id string, fields map[string]any) (*Record, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return nil, err
	}
	clean, err := t.Normalize(fields)
	if err != nil {
		return nil, err
	}
	if _, err := s.Get(t.Name, id); err == nil {
		return nil, fmt.Errorf("%s %s already exists", t.Name, id)
	}
	return s.insert(t, id, clean)
}

// Put writes a record whole, under a given id and with given times, over
// whatever was there. It is how the content mirror is read back: the file
// says what the record is and when it changed, and the database agrees.
func (s *Store) Put(typeName, id string, fields map[string]any, created, updated time.Time) (*Record, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return nil, err
	}
	clean, err := t.Normalize(fields)
	if err != nil {
		return nil, err
	}
	var before map[string]any
	if was, err := s.Get(t.Name, id); err == nil {
		before = was.Fields
	}
	rec, err := s.put(t, id, clean, created, updated)
	if err == nil {
		s.stamp(t, id, before, clean, created)
	}
	return rec, err
}

// put writes a record whole, and nothing else: what Put and a stamp from
// another copy both come down to.
func (s *Store) put(t *schema.Type, id string, clean map[string]any, created, updated time.Time) (*Record, error) {
	rec := &Record{ID: id, Type: t.Name, CreatedAt: created.UTC(), UpdatedAt: updated.UTC(), Fields: clean}
	cols := []string{"id", "created_at", "updated_at"}
	args := []any{rec.ID, rec.CreatedAt.Format(time.RFC3339Nano), rec.UpdatedAt.Format(time.RFC3339Nano)}
	for _, f := range t.Fields {
		cols = append(cols, quote(f.Name))
		args = append(args, encode(f, clean[f.Name]))
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)", quote(t.Name), strings.Join(cols, ","), marks)
	if _, err := s.db.Exec(stmt, args...); err != nil {
		return nil, fmt.Errorf("put %s: %w", t.Name, err)
	}
	s.wrote(rec)
	return rec, nil
}

func (s *Store) wrote(rec *Record) {
	if s.AfterWrite != nil {
		s.AfterWrite(rec.Type, rec.ID, rec)
	}
}
