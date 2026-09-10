// Package store persists content records in SQLite.
//
// One table per content type, one column per field, so the database stays
// readable with any SQLite tool. Tables and columns are created and added on
// open from the schema; columns are never dropped automatically.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sameway-dev/sameway/internal/schema"
)

// Record is one stored item of a content type.
type Record struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Fields    map[string]any `json:"fields"`
}

// Store is an open database bound to a schema set.
type Store struct {
	db    *sql.DB
	types *schema.Set
}

// Open opens (or creates) the database at path and migrates it to match types.
// Use ":memory:" for tests.
func Open(path string, types *schema.Set) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, types: types}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

// Types returns the schema set this store was opened with.
func (s *Store) Types() *schema.Set { return s.types }

func (s *Store) migrate() error {
	for _, t := range s.types.Types {
		cols := []string{"id TEXT PRIMARY KEY", "created_at TEXT NOT NULL", "updated_at TEXT NOT NULL"}
		for _, f := range t.Fields {
			cols = append(cols, quote(f.Name)+" "+columnType(f))
		}
		stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", quote(t.Name), strings.Join(cols, ", "))
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("create table %s: %w", t.Name, err)
		}
		existing, err := s.columns(t.Name)
		if err != nil {
			return err
		}
		for _, f := range t.Fields {
			if existing[f.Name] {
				continue
			}
			alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", quote(t.Name), quote(f.Name), columnType(f))
			if _, err := s.db.Exec(alter); err != nil {
				return fmt.Errorf("add column %s.%s: %w", t.Name, f.Name, err)
			}
		}
	}
	return nil
}

func (s *Store) columns(table string) (map[string]bool, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", quote(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func columnType(f schema.Field) string {
	switch f.Type {
	case "int", "bool":
		return "INTEGER"
	case "float":
		return "REAL"
	default:
		return "TEXT"
	}
}

// quote wraps an identifier in double quotes. Names are already validated by
// the schema package, so this only guards against SQL keywords.
func quote(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// NewID returns a 16 character, URL safe, lowercase random id.
func NewID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]))
}
