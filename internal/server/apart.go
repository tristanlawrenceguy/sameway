package server

import (
	"path"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Two records with one title make two controls with one name: two boxes
// "Done Call plumber" in one list, two links "Call plumber" to different
// pages, two "Undo created task Call plumber" in the log. A person moving
// by links or controls, or an agent finding one by its role and name,
// cannot tell them apart (WCAG 2.4.6 and 2.4.9; a Playwright getByRole
// that matches two refuses to guess). So where a list shows records whose
// titles repeat, each of the repeated ones is told apart by the first fact
// that differs: the day it is due, then when it was added, then its id.
// Only when needed: a title that is its own stays as it is, and the words
// are read after the name, never shown twice.

// apart gives each of names the words that tell it from the others with
// the same name, or "" where the name is its own. ways(i) are the words
// item i could be told apart by, best first; the first way whose words
// differ across every item sharing the name is used for all of them. A
// way may leave one item without words, which is then the plain one.
func apart(names []string, ways func(i int) []string) []string {
	out := make([]string, len(names))
	groups := map[string][]int{}
	var order []string
	for i, n := range names {
		k := strings.ToLower(strings.Join(strings.Fields(n), " "))
		if k == "" {
			continue // nameless, or not shown with the others
		}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], i)
	}
	for _, k := range order {
		g := groups[k]
		if len(g) < 2 {
			continue
		}
		cands := make([][]string, len(g))
		most := 0
		for j, i := range g {
			cands[j] = ways(i)
			most = max(most, len(cands[j]))
		}
		for w := 0; w < most; w++ {
			seen, differ := map[string]bool{}, true
			for _, c := range cands {
				v := ""
				if w < len(c) {
					v = c[w]
				}
				if seen[v] {
					differ = false
					break
				}
				seen[v] = true
			}
			if !differ {
				continue
			}
			for j, i := range g {
				if w < len(cands[j]) {
					out[i] = cands[j][w]
				}
			}
			break
		}
	}
	return out
}

// recordWays is what tells one record from another with its title: the
// day that matters to it (due Fri 25 Sep), when it was added, its id.
func recordWays(t *schema.Type, rec *store.Record) []string {
	return []string{dayWords(t, rec), "added " + momentWords(rec.CreatedAt), "id " + shortID(rec.ID), "id " + rec.ID}
}

// shortID is the start of an id, enough to tell a few apart.
func shortID(id string) string {
	if len(id) > 6 {
		return id[:6]
	}
	return id
}

// recordsApart is apart for one type's records shown together, by id.
func (s *Server) recordsApart(t *schema.Type, recs []*store.Record) map[string]string {
	names := make([]string, len(recs))
	for i, rec := range recs {
		names[i] = s.title(t, rec)
	}
	said := apart(names, func(i int) []string { return recordWays(t, recs[i]) })
	out := map[string]string{}
	for i, rec := range recs {
		if said[i] != "" {
			out[rec.ID] = said[i]
		}
	}
	return out
}

// dayWords is the record's first day as a few words after its title, the
// field named: due Fri 25 Sep, with the year when it is not this one.
func dayWords(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Shown() {
		if f.Type != "datetime" {
			continue
		}
		if v, _ := rec.Fields[f.Name].(string); v != "" {
			if w := whenWords(v); w != "" {
				return strings.ToLower(fieldLabel(f)) + " " + w
			}
		}
	}
	return ""
}

// whenWords is a stored day or moment, short: Fri 25 Sep, or with its
// time, Fri 25 Sep, 14:05.
func whenWords(v string) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return shortDay(ts.UTC())
	}
	return momentWords(ts)
}

func momentWords(at time.Time) string {
	return shortDay(at.Local()) + ", " + at.Local().Format("15:04")
}

func shortDay(d time.Time) string {
	if d.Year() != time.Now().Year() {
		return d.Format("Mon 2 Jan 2006")
	}
	return d.Format("Mon 2 Jan")
}

// withContext is a name and the words that tell it apart: Call plumber
// (due Fri 25 Sep). In brackets, because a browser puts a space before
// words hidden in a span of their own, so a comma would be heard alone.
func withContext(name, context string) string {
	if context == "" {
		return name
	}
	return name + " (" + context + ")"
}

// markApart names a list item's mark after its record and the words that
// tell it apart, so the box reads Done Call plumber, due Fri 25 Sep.
func markApart(actions []any, context string) {
	if context == "" {
		return
	}
	for _, a := range actions {
		if m, ok := a.(map[string]any); ok && m["component"] == "mark" {
			if p, ok := m["props"].(map[string]any); ok {
				p["context"] = withContext(str(p["context"], ""), context)
			}
		}
	}
}

// eventsApart tells apart a calendar's events with one label, two called
// Call plumber, by their kind, their day and time, then by the record
// they lead to.
// Only those in the month shown are met together; month "" is them all.
func eventsApart(events []any, month string) []any {
	names := make([]string, len(events))
	for i, e := range events {
		ev, _ := e.(map[string]any)
		names[i] = str(ev["label"], "")
		if !strings.HasPrefix(str(ev["date"], ""), month) {
			names[i] = "" // not met here
		}
	}
	told := apart(names, func(i int) []string {
		ev, _ := events[i].(map[string]any)
		on := ""
		if d, err := time.Parse("2006-01-02", str(ev["date"], "")); err == nil {
			on = "on " + shortDay(d)
			if at := str(ev["time"], ""); at != "" {
				on += ", " + at
			}
		}
		id := path.Base(str(ev["href"], ""))
		// On a calendar of several kinds a task and a reminder of one name
		// are told apart by their kind first, which is not said otherwise.
		kind := schema.Words(str(ev["kind"], ""))
		both := on
		if kind != "" {
			both = kind + ", " + on
		}
		return []string{kind, on, both, "id " + shortID(id), "id " + id}
	})
	for i, e := range events {
		if ev, ok := e.(map[string]any); ok && told[i] != "" {
			ev["context"] = told[i]
			actions, _ := ev["actions"].([]any)
			markApart(actions, told[i])
		}
	}
	return events
}

// blockName is what a block's own controls are named after: Remove Up
// next, Expand Up next, from the label the block shows, and the plain
// name of its component only when it has none. Two lists would otherwise
// both be Remove collection.
func blockName(component string, props map[string]any) string {
	if said := chat.Summarise(component, props); said != "" {
		return said
	}
	for _, k := range []string{"label", "caption", "title"} {
		if v, _ := props[k].(string); strings.TrimSpace(v) != "" {
			return trim.Title(v)
		}
	}
	return component
}
