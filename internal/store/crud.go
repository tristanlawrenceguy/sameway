package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// ErrNotFound is returned when a record id does not exist.
var ErrNotFound = errors.New("not found")

// ListOptions controls List. OrderBy must be a field name, "created_at", or
// "updated_at". Limit 0 means no limit.
type ListOptions struct {
	OrderBy string
	Desc    bool
	Limit   int
}

// Create validates fields against the type and inserts a new record.
func (s *Store) Create(typeName string, fields map[string]any) (*Record, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return nil, err
	}
	clean, err := t.Normalize(fields)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rec := &Record{ID: NewID(), Type: t.Name, CreatedAt: now, UpdatedAt: now, Fields: clean}
	cols := []string{"id", "created_at", "updated_at"}
	args := []any{rec.ID, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}
	for _, f := range t.Fields {
		cols = append(cols, quote(f.Name))
		args = append(args, encode(f, clean[f.Name]))
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(t.Name), strings.Join(cols, ","), marks)
	if _, err := s.db.Exec(stmt, args...); err != nil {
		return nil, fmt.Errorf("insert %s: %w", t.Name, err)
	}
	return rec, nil
}

// Get returns one record by id.
func (s *Store) Get(typeName, id string) (*Record, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", selectList(t), quote(t.Name)), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrNotFound
	}
	return scan(t, rows)
}

// List returns records of a type, newest first by default.
func (s *Store) List(typeName string, opts ListOptions) ([]*Record, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return nil, err
	}
	order := "created_at"
	if opts.OrderBy != "" {
		if _, ok := t.Field(opts.OrderBy); !ok && opts.OrderBy != "created_at" && opts.OrderBy != "updated_at" {
			return nil, fmt.Errorf("cannot order %s by unknown field %q", t.Name, opts.OrderBy)
		}
		order = opts.OrderBy
	}
	dir := "ASC"
	if opts.Desc || opts.OrderBy == "" {
		dir = "DESC"
	}
	// rowid breaks ties, so records written in the same clock tick keep
	// insertion order instead of falling back to a random id.
	q := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s %s, rowid %s", selectList(t), quote(t.Name), quote(order), dir, dir)
	if opts.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*Record, 0)
	for rows.Next() {
		rec, err := scan(t, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// Update merges fields into an existing record and validates the result.
func (s *Store) Update(typeName, id string, fields map[string]any) (*Record, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return nil, err
	}
	current, err := s.Get(typeName, id)
	if err != nil {
		return nil, err
	}
	merged := map[string]any{}
	for k, v := range current.Fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	clean, err := t.Normalize(merged)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sets := []string{"updated_at = ?"}
	args := []any{now.Format(time.RFC3339Nano)}
	for _, f := range t.Fields {
		sets = append(sets, quote(f.Name)+" = ?")
		args = append(args, encode(f, clean[f.Name]))
	}
	args = append(args, id)
	stmt := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", quote(t.Name), strings.Join(sets, ", "))
	if _, err := s.db.Exec(stmt, args...); err != nil {
		return nil, fmt.Errorf("update %s: %w", t.Name, err)
	}
	current.Fields = clean
	current.UpdatedAt = now
	return current, nil
}

// Delete removes one record. Deleting a missing record is ErrNotFound.
func (s *Store) Delete(typeName, id string) error {
	t, err := s.typ(typeName)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", quote(t.Name)), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAll removes every record of a type.
func (s *Store) DeleteAll(typeName string) error {
	t, err := s.typ(typeName)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(fmt.Sprintf("DELETE FROM %s", quote(t.Name)))
	return err
}

// Count returns how many records a type has.
func (s *Store) Count(typeName string) (int, error) {
	t, err := s.typ(typeName)
	if err != nil {
		return 0, err
	}
	var n int
	err = s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", quote(t.Name))).Scan(&n)
	return n, err
}

func (s *Store) typ(name string) (*schema.Type, error) {
	t, ok := s.types.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown content type %q (known: %s)", name, strings.Join(s.types.Names(), ", "))
	}
	return t, nil
}

func selectList(t *schema.Type) string {
	cols := []string{"id", "created_at", "updated_at"}
	for _, f := range t.Fields {
		cols = append(cols, quote(f.Name))
	}
	return strings.Join(cols, ", ")
}

func scan(t *schema.Type, rows *sql.Rows) (*Record, error) {
	dest := make([]any, 3+len(t.Fields))
	var id, created, updated string
	dest[0], dest[1], dest[2] = &id, &created, &updated
	raw := make([]sql.NullString, len(t.Fields))
	for i := range t.Fields {
		dest[3+i] = &raw[i]
	}
	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}
	rec := &Record{ID: id, Type: t.Name, Fields: map[string]any{}}
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	for i, f := range t.Fields {
		rec.Fields[f.Name] = decode(f, raw[i])
	}
	return rec, nil
}

// encode turns a normalized Go value into a SQLite value.
func encode(f schema.Field, v any) any {
	switch f.Type {
	case "bool":
		if b, _ := v.(bool); b {
			return 1
		}
		return 0
	case "list", "json":
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return v
	}
}

// decode turns a SQLite value back into the canonical Go value. A NULL
// column means the field was added to the schema after this record was
// written, so it reads back as the field's default.
func decode(f schema.Field, v sql.NullString) any {
	if !v.Valid {
		return schema.DefaultValue(f)
	}
	switch f.Type {
	case "int":
		var n int64
		fmt.Sscan(v.String, &n)
		return n
	case "float":
		var x float64
		fmt.Sscan(v.String, &x)
		return x
	case "bool":
		return v.String == "1"
	case "list":
		var l []any
		if json.Unmarshal([]byte(v.String), &l) != nil || l == nil {
			return []any{}
		}
		return l
	case "json":
		var j any
		if json.Unmarshal([]byte(v.String), &j) != nil {
			return nil
		}
		return j
	default:
		return v.String
	}
}
