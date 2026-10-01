package ingest

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A row that names a person is linked to that person. When the type being
// imported has one ref field to a type with an email or a phone (an
// interaction has a person), each row is matched to an existing person
// by email, then phone, then name, and a new person is made when none
// matches and the row knows a name or an email.
type linker struct {
	st      *store.Store
	field   string
	to      *schema.Type
	byEmail map[string]string
	byPhone map[string]string
	byName  map[string]string
	title   string
	email   string
	phone   string
}

func newLinker(st *store.Store, t *schema.Type) *linker {
	var refs []schema.Field
	for _, f := range t.Fields {
		if f.Type == "ref" {
			refs = append(refs, f)
		}
	}
	if len(refs) != 1 {
		return nil
	}
	return linkerFor(st, refs[0])
}

// linkerFor finds and makes the records one field links to, by email,
// phone or name: a row's one person, or each of a meeting's people.
func linkerFor(st *store.Store, f schema.Field) *linker {
	to, ok := st.Types().Get(f.To)
	if !ok {
		return nil
	}
	l := &linker{st: st, field: f.Name, to: to, title: to.Title, email: fieldOfType(to, "email"), phone: fieldOfType(to, "phone"),
		byEmail: map[string]string{}, byPhone: map[string]string{}, byName: map[string]string{}}
	if l.email == "" && l.phone == "" {
		return nil
	}
	recs, err := st.List(to.Name, store.ListOptions{})
	if err != nil {
		return nil
	}
	for _, rec := range recs {
		l.remember(rec)
	}
	return l
}

func (l *linker) remember(rec *store.Record) {
	if e, _ := rec.Fields[l.email].(string); e != "" {
		l.byEmail[strings.ToLower(e)] = rec.ID
	}
	if p, _ := rec.Fields[l.phone].(string); digits(p) != "" {
		l.byPhone[digits(p)] = rec.ID
	}
	if n, _ := rec.Fields[l.title].(string); n != "" {
		l.byName[normal(n)] = rec.ID
	}
}

// link is the id and name of the person a row is with, made if need be,
// or nothing when the row names no one. Columns already mapped into the
// record are not read as the person.
func (l *linker) link(row map[string]string, m Mapping) (id, who string) {
	// Columns are found by their plain names, however they are written,
	// and a column already feeding a field is not the person.
	mapped := map[string]bool{}
	for col, f := range m {
		if f != "" {
			mapped[normal(col)] = true
		}
	}
	pick := func(names ...string) string {
		for _, n := range names {
			if mapped[n] {
				continue
			}
			if v := cell(row, n); v != "" {
				return v
			}
		}
		return ""
	}
	return l.find(pick("from_name", "name", "person", "contact", "who"), pick("from_email", "email", "e-mail", "email address"),
		pick("phone", "number", "telephone", "mobile", "from_phone"))
}

// find is the person with this email, phone or name, made when there is
// none, and what to call them.
func (l *linker) find(name, rawEmail, rawPhone string) (id, who string) {
	email, phone := strings.ToLower(strings.TrimSpace(rawEmail)), digits(rawPhone)
	if email != "" {
		if id, ok := l.byEmail[email]; ok {
			return id, l.nameOf(id, name, email)
		}
	}
	if phone != "" {
		if id, ok := l.byPhone[phone]; ok {
			return id, l.nameOf(id, name, phone)
		}
	}
	if name != "" {
		if id, ok := l.byName[normal(name)]; ok {
			return id, name
		}
	}
	if name == "" && email == "" && phone == "" {
		return "", ""
	}
	fields := map[string]any{}
	who = firstOf(name, email, rawPhone)
	if l.title != "" {
		fields[l.title] = who
	}
	if l.email != "" && email != "" {
		fields[l.email] = email
	}
	if l.phone != "" && phone != "" {
		fields[l.phone] = rawPhone
	}
	rec, err := l.st.Create(l.to.Name, fields)
	if err != nil {
		return "", ""
	}
	l.remember(rec)
	return rec.ID, who
}

// nameOf is what to call a person found by id: their name here, or the
// name or address the row had.
func (l *linker) nameOf(id string, fallback ...string) string {
	if rec, err := l.st.Get(l.to.Name, id); err == nil {
		if n, _ := rec.Fields[l.title].(string); n != "" {
			return n
		}
	}
	return firstOf(fallback...)
}

// digits is a phone number as the digits alone, so 07700 900123 and
// +44 7700 900123 meet.
func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if len(d) > 10 {
		d = d[len(d)-10:]
	}
	return d
}

// nameAndEmail splits "Ann Lee <ann@example.com>" into its two halves;
// an address alone is the email, anything else the name.
func nameAndEmail(s string) (name, email string) {
	s = strings.TrimSpace(s)
	if i, j := strings.LastIndex(s, "<"), strings.LastIndex(s, ">"); i >= 0 && j > i {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1 : j])
	}
	if strings.Contains(s, "@") && !strings.Contains(s, " ") {
		return "", s
	}
	return s, ""
}

// linkLists finds, or makes, each record a list of refs names: a
// meeting's people, by their email or name.
func linkLists(st *store.Store, t *schema.Type, fields map[string]any) {
	for _, f := range t.Fields {
		items, _ := fields[f.Name].([]any)
		if !f.RefList() || len(items) == 0 {
			continue
		}
		l := linkerFor(st, f)
		if l == nil {
			delete(fields, f.Name)
			continue
		}
		ids := make([]any, 0, len(items))
		for _, it := range items {
			name, email := nameAndEmail(fmt.Sprint(it))
			if id, _ := l.find(name, email, ""); id != "" {
				ids = append(ids, id)
			}
		}
		fields[f.Name] = ids
	}
}
