// Package content is the portable form of a workspace: every record as a
// Markdown file with YAML front matter, one folder per content type under
// content/, named by id. The database is the live store; these files are
// what gets read, diffed and shared through git, and they are written as
// records change so they are never a step behind.
package content

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Mirror is one workspace's content folder and the types written to it.
type Mirror struct {
	Dir   string
	Types *schema.Set
	// Skip names the types that are history rather than content: the
	// conversation, its questions, the activity log.
	Skip []string
	// DryRun makes Import say what it would change, and change nothing.
	DryRun bool
}

// Mirrored says whether a type is written to the folder.
func (m Mirror) Mirrored(typeName string) bool {
	if m.Dir == "" || m.Types == nil {
		return false
	}
	if _, ok := m.Types.Get(typeName); !ok {
		return false
	}
	for _, s := range m.Skip {
		if s == typeName {
			return false
		}
	}
	return true
}

// Path is where a record's file lives.
func (m Mirror) Path(typeName, id string) string {
	return filepath.Join(m.Dir, typeName, id+".md")
}

// Changed is the store's write hook: the record's file is written, or
// removed when the record is gone. A file that cannot be written is said
// on the log rather than failing the write that caused it; the database
// is still right, and sameway export catches the folder up.
func (m Mirror) Changed(typeName, id string, rec *store.Record) {
	if !m.Mirrored(typeName) {
		return
	}
	var err error
	if rec == nil {
		err = m.remove(typeName, id)
	} else {
		err = m.write(rec)
	}
	if err != nil {
		log.Printf("content: %v", err)
	}
}

func (m Mirror) write(rec *store.Record) error {
	t, _ := m.Types.Get(rec.Type)
	data, err := Encode(t, rec)
	if err != nil {
		return err
	}
	path := m.Path(rec.Type, rec.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
		return nil
	}
	return os.WriteFile(path, data, 0o644)
}

func (m Mirror) remove(typeName, id string) error {
	err := os.Remove(m.Path(typeName, id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// BodyField is the field written as the Markdown body: the type's first
// markdown or text field. Everything else goes in the front matter.
func BodyField(t *schema.Type) string {
	for _, f := range t.Fields {
		if f.Type == "markdown" || f.Type == "text" {
			return f.Name
		}
	}
	return ""
}

// Encode renders one record as front matter and body.
func Encode(t *schema.Type, rec *store.Record) ([]byte, error) {
	body := BodyField(t)
	front := map[string]any{
		"created": rec.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated": rec.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	for _, f := range t.Fields {
		v, ok := rec.Fields[f.Name]
		if !ok || v == nil || f.Name == body {
			continue
		}
		front[f.Name] = v
	}
	head, err := yaml.Marshal(front)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", rec.Type, rec.ID, err)
	}
	var b bytes.Buffer
	b.WriteString("---\n")
	b.Write(head)
	b.WriteString("---\n")
	if body != "" {
		if s, _ := rec.Fields[body].(string); s != "" {
			b.WriteString(strings.TrimRight(s, "\n"))
			b.WriteString("\n")
		}
	}
	return b.Bytes(), nil
}

// Decode reads a file back into fields and times. The fields are as the
// file has them; the store normalises them on the way in.
func Decode(t *schema.Type, data []byte) (fields map[string]any, created, updated time.Time, err error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, created, updated, errors.New("no front matter: the file must start with ---")
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---\n")
	var head, body string
	switch {
	case strings.HasPrefix(rest, "---\n"):
		head, body = "", rest[4:]
	case end < 0:
		if strings.HasSuffix(rest, "\n---") {
			head = rest[:len(rest)-4]
		} else {
			return nil, created, updated, errors.New("front matter never closes: no --- line after it")
		}
	default:
		head, body = rest[:end], rest[end+5:]
	}
	fields = map[string]any{}
	if strings.TrimSpace(head) != "" {
		if err := yaml.Unmarshal([]byte(head), &fields); err != nil {
			return nil, created, updated, fmt.Errorf("front matter: %w", err)
		}
	}
	created = stamp(fields, "created")
	updated = stamp(fields, "updated")
	// A field that merely looks like a date is still text to the schema.
	for k, v := range fields {
		if ts, ok := v.(time.Time); ok {
			fields[k] = ts.UTC().Format(time.RFC3339Nano)
		}
	}
	if bodyField := BodyField(t); bodyField != "" {
		if body = strings.TrimRight(body, "\n"); body != "" {
			fields[bodyField] = body
		}
	}
	return fields, created, updated, nil
}

// stamp takes a time out of the front matter, now when it is missing. YAML
// reads a bare timestamp as a time already; a quoted one is a string.
func stamp(fields map[string]any, key string) time.Time {
	v := fields[key]
	delete(fields, key)
	switch ts := v.(type) {
	case time.Time:
		return ts.UTC()
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			return parsed
		}
	}
	return time.Now().UTC()
}

// ids lists the record ids that have files for a type, sorted.
func (m Mirror) ids(typeName string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, typeName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".md"); ok && !e.IsDir() && name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}
