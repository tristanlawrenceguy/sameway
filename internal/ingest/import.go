package ingest

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Mapping is which column feeds which field. A column absent from it, or
// mapped to nothing, is left out; a column named email or phone still
// serves to link the row to a person.
type Mapping map[string]string

// synonyms are the other names a column often has for a field.
var synonyms = map[string][]string{
	"name":         {"full name", "fn", "display name", "contact", "person", "who"},
	"email":        {"e mail", "email address", "e mail address", "mail", "from email"},
	"phone":        {"telephone", "mobile", "phone number", "tel", "cell", "number"},
	"organisation": {"organization", "company", "org", "employer"},
	"role":         {"title", "job title", "position"},
	"notes":        {"note", "comment", "comments", "body", "description"},
	"summary":      {"subject", "what", "about"},
	"at":           {"date", "when", "time", "date time", "datetime", "timestamp", "sent", "received"},
	"kind":         {"type", "channel", "call type"},
	"tags":         {"tag", "labels", "label", "groups", "categories"},
	"due":          {"due date", "deadline"},
	"done":         {"completed", "finished"},
}

// Guess matches columns to a type's fields: by the same name, by a name
// the field is often called, and a date column to the type's first date
// field. The title field takes a name or full name column.
func Guess(t *schema.Type, columns []string) Mapping {
	m := Mapping{}
	used := map[string]bool{}
	take := func(col, field string) {
		if used[field] || m[col] != "" {
			return
		}
		m[col], used[field] = field, true
	}
	for _, col := range columns {
		n := normal(col)
		for _, f := range t.Fields {
			if normal(f.Name) == n || normal(f.Label) == n {
				take(col, f.Name)
			}
		}
	}
	for _, col := range columns {
		n := normal(col)
		for _, f := range t.Fields {
			for _, alias := range synonyms[f.Name] {
				if alias == n {
					take(col, f.Name)
				}
			}
		}
		if t.Title != "" && (n == "name" || n == "full name" || n == "fn") {
			take(col, t.Title)
		}
	}
	for _, col := range columns {
		n := normal(col)
		if n == "date" || n == "when" || n == "at" || n == "time" || n == "sent" {
			for _, f := range t.Fields {
				if f.Type == "datetime" {
					take(col, f.Name)
					break
				}
			}
		}
	}
	return m
}

// Report says how an import went.
type Report struct {
	Made     int
	Skipped  int
	Linked   int
	Problems []string
	IDs      []string
}

// String says it in a sentence.
func (r Report) String() string {
	s := fmt.Sprintf("imported %d", r.Made)
	if r.Skipped > 0 {
		s += fmt.Sprintf(", skipped %d", r.Skipped)
	}
	if r.Linked > 0 {
		s += fmt.Sprintf(", %d linked to people", r.Linked)
	}
	if len(r.Problems) > 0 {
		s += ": " + strings.Join(r.Problems, "; ")
	}
	return s
}

// Import makes one record of t per row, as the mapping says. A row whose
// email is already on a record of t is left out rather than made twice.
// When t has a ref to a type with an email or phone field (interactions
// to people), each row is linked to the person its email, phone or name
// picks out, made when there is none.
func Import(st *store.Store, t *schema.Type, tb *Table, m Mapping) Report {
	var r Report
	emailField := fieldOfType(t, "email")
	existing := map[string]bool{}
	if emailField != "" {
		if recs, err := st.List(t.Name, store.ListOptions{}); err == nil {
			for _, rec := range recs {
				if e, _ := rec.Fields[emailField].(string); e != "" {
					existing[strings.ToLower(e)] = true
				}
			}
		}
	}
	people := newLinker(st, t)
	for _, row := range tb.Rows {
		fields := map[string]any{}
		for col, name := range m {
			f, ok := t.Field(name)
			// A field Sameway keeps is never filled from a file.
			if !ok || name == "" || f.ReadOnly {
				continue
			}
			if v := strings.TrimSpace(row[col]); v != "" {
				fields[name] = coerce(*f, v)
			}
		}
		if emailField != "" {
			if e, _ := fields[emailField].(string); e != "" && existing[strings.ToLower(e)] {
				r.Skipped++
				r.problem("already here: " + e)
				continue
			}
		}
		who := ""
		if people != nil {
			if id, name := people.link(row, m); id != "" {
				fields[people.field] = id
				r.Linked++
				who = name
			}
		}
		// A row with no title of its own is named for what it is and who
		// it was with: "Call with Sandra Lee", or the name or address.
		if t.Title != "" && fields[t.Title] == nil {
			if guess := firstOf(cell(row, "name"), cell(row, "from_name"), cell(row, "summary"), cell(row, "subject"), cell(row, "email"), cell(row, "from_email")); guess != "" && who == "" {
				fields[t.Title] = guess
			} else if label := firstOf(who, cell(row, "phone"), cell(row, "number")); label != "" {
				kind, _ := fields["kind"].(string)
				fields[t.Title] = capitalize(firstOf(kind, t.Name)) + " with " + label
			}
		}
		rec, err := st.Create(t.Name, fields)
		if err != nil {
			r.Skipped++
			r.problem(err.Error())
			continue
		}
		if emailField != "" {
			if e, _ := fields[emailField].(string); e != "" {
				existing[strings.ToLower(e)] = true
			}
		}
		r.Made++
		r.IDs = append(r.IDs, rec.ID)
	}
	return r
}

func (r *Report) problem(p string) {
	if len(r.Problems) < 5 {
		r.Problems = append(r.Problems, p)
	} else if len(r.Problems) == 5 {
		r.Problems = append(r.Problems, "and more")
	}
}

// fieldOfType is the name of the field that plainly holds a kind of
// thing, such as an email, by its name.
func fieldOfType(t *schema.Type, what string) string {
	for _, f := range t.Fields {
		if normal(f.Name) == what {
			return f.Name
		}
	}
	for _, f := range t.Fields {
		for _, alias := range synonyms[what] {
			if normal(f.Name) == alias {
				return f.Name
			}
		}
	}
	return ""
}

// coerce shapes a cell for a field: a list from a separated string, a
// bool from a word, a moment as RFC 3339 when it is written another way.
func coerce(f schema.Field, v string) any {
	switch f.Type {
	case "list":
		var out []any
		for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' || r == '|' }) {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case "bool":
		switch strings.ToLower(v) {
		case "true", "yes", "y", "1", "done", "x":
			return true
		default:
			return false
		}
	case "int":
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
		return v
	case "float":
		var x float64
		if _, err := fmt.Sscanf(v, "%g", &x); err == nil {
			return x
		}
		return v
	case "datetime":
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04", "02/01/2006 15:04", "02/01/2006", "2/1/2006 15:04", time.RFC1123Z, time.RFC1123, "Mon, 2 Jan 2006 15:04:05 -0700", "January 2, 2006 15:04", "2 Jan 2006 15:04", "2006/01/02 15:04:05"} {
			if ts, err := time.ParseInLocation(layout, v, time.Local); err == nil {
				return ts.UTC().Format(time.RFC3339)
			}
		}
		return v
	}
	return v
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// cell is the value of a column found by its plain name, however the
// column is written: Number, number and NUMBER are one column.
func cell(row map[string]string, name string) string {
	if v, ok := row[name]; ok {
		return strings.TrimSpace(v)
	}
	for col, v := range row {
		if normal(col) == normal(name) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
