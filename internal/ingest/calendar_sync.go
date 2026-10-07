package ingest

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A calendar kept elsewhere (Google, Outlook, iCloud) was brought in once,
// from a file, and was out of date the next day. A calendar link keeps it
// in step: its events are read as an import reads a file, each known by
// the id its calendar gave it, so a sync adds what is new, changes what
// changed, and takes away what the calendar no longer has, of the events
// it brought itself. An event the person made is never touched.

// Synced is what one sync did.
type Synced struct {
	Added, Changed, Removed int
	// UIDs are the calendar's events now, for the next sync to compare.
	UIDs []string
}

// String says it in a few words, "" when nothing changed.
func (s Synced) String() string {
	var parts []string
	for _, p := range []struct {
		n    int
		word string
	}{{s.Added, "added"}, {s.Changed, "changed"}, {s.Removed, "removed"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.word))
		}
	}
	return strings.Join(parts, ", ")
}

// SyncCalendar keeps the events of t in step with a calendar's table (from
// ReadICS); had are the calendar's ids the last sync brought, the only
// events it may take away.
func SyncCalendar(st *store.Store, t *schema.Type, tb *Table, had []string) (Synced, error) {
	var out Synced
	if _, ok := t.Field("uid"); !ok {
		return out, fmt.Errorf("%s has no uid field to keep a calendar in step by", t.Name)
	}
	byUID := map[string]*store.Record{}
	if recs, err := st.List(t.Name, store.ListOptions{}); err == nil {
		for _, r := range recs {
			if u, _ := r.Fields["uid"].(string); u != "" {
				byUID[u] = r
			}
		}
	}
	m := Guess(t, tb.Columns)
	now := map[string]bool{}
	for _, row := range tb.Rows {
		uid := strings.TrimSpace(row["uid"])
		if uid == "" || now[uid] {
			continue
		}
		now[uid] = true
		out.UIDs = append(out.UIDs, uid)
		fields := map[string]any{}
		for col, name := range m {
			f, ok := t.Field(name)
			if !ok || name == "" || name == "people" {
				continue
			}
			if v := strings.TrimSpace(row[col]); v != "" {
				fields[name] = coerce(*f, v)
			}
		}
		was := byUID[uid]
		if was == nil {
			if _, err := st.Create(t.Name, fields); err == nil {
				out.Added++
			}
			continue
		}
		changed := map[string]any{}
		for k, v := range fields {
			if fmt.Sprint(was.Fields[k]) != fmt.Sprint(v) {
				changed[k] = v
			}
		}
		if len(changed) > 0 {
			if _, err := st.Update(t.Name, was.ID, changed); err == nil {
				out.Changed++
			}
		}
	}
	for _, uid := range had {
		if r := byUID[uid]; r != nil && !now[uid] {
			if err := st.Delete(t.Name, r.ID); err == nil {
				out.Removed++
			}
		}
	}
	return out, nil
}
