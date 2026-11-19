// Package search finds what a person has, by the words in it: every
// record of every content type they use, and every block on the canvas.
// One implementation serves the search page, the assistant's tool, the
// API and MCP, so "find the plumber" means the same thing everywhere.
package search

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Hit is one thing found: what it is, where it is, and the words around
// the match so a person can tell which one before opening it.
type Hit struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet,omitempty"`
	Href    string `json:"href"`
	// Near says it was found by what it is about, not by its words
	// (near.go).
	Near bool `json:"near,omitempty"`
	// score orders hits: a title match before a body match.
	score int
}

// Skip names the types that are history, not a person's content.
var Skip = map[string]bool{"message": true, "activity": true, "proposal": true, "canvas": true}

// Limit caps how many hits come back.
const Limit = 50

// Reader is who a search is for: the moment it is for them, which says
// what "today" and "Friday" are, and whether they read times on the
// 24-hour clock, so the days found are said as they read them.
type Reader struct {
	Now time.Time
	H24 bool
}

// Matches is what a search finds, the one way for the search page, the
// assistant, the API, the command line and AI services: everything with
// every word, or, when nothing has them all and there are several, what
// has some of them, most first, with some true so it is said.
func Matches(st *store.Store, types *schema.Set, q string, r Reader) (hits []Hit, some bool) {
	if days := daysAsked(q, r.Now); len(days) > 0 { // days.go
		on := onDays(st, types, days, r)
		if len(on) == 0 && namesWeekday(q) {
			on = onDays(st, types, weekLater(days), r) // that weekday, the week after
		}
		if len(on) > 0 {
			return withWords(on, find(st, types, q, "", false, r)), false
		}
	}
	if hits = find(st, types, q, "", false, r); len(hits) > 0 || len(Words(q)) < 2 {
		return near(st, types, q, hits, r), false
	}
	// Some of the words is a weak match ("the plumber" has "the" in "Renew
	// the passport"); what is near in meaning goes before it.
	partial := find(st, types, q, "", true, r)
	nearOnly := near(st, types, q, nil, r)
	have := map[string]bool{}
	for _, h := range nearOnly {
		have[h.Type+"/"+h.ID] = true
	}
	for _, h := range partial {
		if !have[h.Type+"/"+h.ID] {
			nearOnly = append(nearOnly, h)
		}
	}
	return nearOnly, true
}

// withWords is the hits on a day, then those with the words not already
// among them.
func withWords(on, words []Hit) []Hit {
	have := map[string]bool{}
	for _, h := range on {
		have[h.Type+"/"+h.ID] = true
	}
	for _, h := range words {
		if !have[h.Type+"/"+h.ID] {
			on = append(on, h)
		}
	}
	return on
}

// Counts is how many of the hits are of each type: the filters on the
// results page are counted from the one search, not a search per type.
func Counts(hits []Hit) map[string]int {
	n := map[string]int{}
	for _, h := range hits {
		n[h.Type]++
	}
	return n
}

// Of is the hits of one type, in the order they were found.
func Of(hits []Hit, only string) []Hit {
	var out []Hit
	for _, h := range hits {
		if h.Type == only {
			out = append(out, h)
		}
	}
	return out
}

func find(st *store.Store, types *schema.Set, q, only string, some bool, r Reader) []Hit {
	words := Words(q)
	if len(words) == 0 {
		return nil
	}
	var hits []Hit
	for _, t := range types.Types {
		// What the system keeps for itself is not searched, except the
		// canvas's blocks, which are what the person made there. Skip once
		// was the only rule, and it left out conversations and clashes:
		// a search by someone let in to look found the titles of the
		// owner's conversations. Searchable (kinds.go) already said so.
		if Skip[t.Name] || t.Internal && t.Name != "block" || only != "" && t.Name != only {
			continue
		}
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			if h, ok := match(t, rec, words, some, r); ok {
				hits = append(hits, h)
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	return hits
}

func match(t *schema.Type, rec *store.Record, words []string, some bool, r Reader) (Hit, bool) {
	title, body := texts(t, rec, r)
	lt, lb := fold(title), fold(body)
	score, found := 0, 0
	for _, w := range words {
		switch {
		case strings.Contains(lt, w):
			score, found = score+2, found+1
		case strings.Contains(lb, w):
			score, found = score+1, found+1
		default:
			if !some {
				return Hit{}, false
			}
		}
	}
	if found == 0 {
		return Hit{}, false
	}
	h := Hit{Type: t.Name, ID: rec.ID, Title: title, Snippet: snippet(body, words), score: score}
	if t.Name == "block" {
		h.Href = "/canvas/" + rec.ID
	} else {
		h.Href = relate.Page(t.Name, rec.ID)
	}
	return h, true
}

// texts is a record as words: its title, and everything else it says. A
// block's title is human-readable content from its props, not the component name.
func texts(t *schema.Type, rec *store.Record, r Reader) (title, body string) {
	if t.Name == "block" {
		name, _ := rec.Fields["component"].(string)
		props, _ := rec.Fields["props"].(map[string]any)
		return blockTitle(name, props), flatten(rec.Fields["props"])
	}
	var parts []string
	for _, f := range t.Fields {
		v := rec.Fields[f.Name]
		if v == nil {
			continue
		}
		// A record it points at is an id, no word of anyone's; it is left out.
		if f.Type == "ref" || f.RefList() {
			continue
		}
		// A date is shown as a person reads it (relative), a choice as its
		// label and a repeat in words, not as it is stored; yes or no says
		// nothing on its own.
		var s string
		switch f.Type {
		case "bool":
			continue
		case "datetime":
			s = when.Relative(flatten(v), r.Now, r.H24)
		case "enum":
			s = f.ValueLabel(flatten(v))
		case "repeat":
			s = when.RepeatText(flatten(v)) // every month, never FREQ=MONTHLY read out
		default:
			s = flatten(v)
		}
		if s == "" {
			continue
		}
		if f.Name == t.Title || (t.Title == "" && title == "" && (f.Type == "string" || f.Type == "text")) {
			title = s
			continue
		}
		parts = append(parts, s)
	}
	if title == "" {
		title = t.Name + " " + rec.ID
	}
	// Each value apart, as a reader needs: "Done · Fri 9 Oct · home".
	return title, strings.Join(parts, " · ")
}

// blockTitle turns a canvas block's component and props into a human-readable
// label for search results. It uses records.Summarise first (which handles
// heading/card/text → text/title/content, table → caption, etc.), then falls
// back through common content props before returning an empty string — never
// leaking the internal component name to user-facing surfaces.
func blockTitle(component string, props map[string]any) string {
	if said := records.Summarise(component, props); said != "" {
		return said
	}
	for _, k := range []string{"label", "caption", "title", "text", "content"} {
		if v, _ := props[k].(string); strings.TrimSpace(v) != "" {
			return trim.Title(v)
		}
	}
	return ""
}

// flatten turns any field value into words.
func flatten(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var parts []string
		for _, it := range x {
			parts = append(parts, flatten(it))
		}
		return strings.Join(parts, " ")
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			parts = append(parts, flatten(x[k]))
		}
		return strings.Join(parts, " ")
	case bool:
		return ""
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}
