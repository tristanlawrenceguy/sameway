// Package search finds what a person has, by the words in it: every
// record of every content type they use, and every block on the canvas.
// One implementation serves the search page, the assistant's tool, the
// API and MCP, so "find the plumber" means the same thing everywhere.
package search

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
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
	// score orders hits: a title match before a body match.
	score int
}

// Skip names the types that are history, not a person's content.
var Skip = map[string]bool{"message": true, "activity": true, "proposal": true, "canvas": true}

// Limit caps how many hits come back.
const Limit = 50

// Find looks for every word of q in every record's text fields and every
// canvas block's props. Every word must appear; case and accents do not
// matter, and a plural finds its one. At most Limit come back.
func Find(st *store.Store, types *schema.Set, q string) []Hit {
	return FindOf(st, types, q, "")
}

// FindOf is Find within one type, when only is named: picking a person
// from thousands finds people, not the fifty notes that also match.
func FindOf(st *store.Store, types *schema.Set, q, only string) []Hit {
	hits := find(st, types, q, only, false)
	if len(hits) > Limit {
		hits = hits[:Limit]
	}
	return hits
}

// FindAll is every hit, for the search page, which shows them a page at a
// time and says how many there are.
func FindAll(st *store.Store, types *schema.Set, q string) []Hit {
	return find(st, types, q, "", false)
}

// FindSome is what has some of the words, when nothing has every one of
// them: most words matched first.
func FindSome(st *store.Store, types *schema.Set, q string) []Hit {
	return find(st, types, q, "", true)
}

func find(st *store.Store, types *schema.Set, q, only string, some bool) []Hit {
	words := Words(q)
	if len(words) == 0 {
		return nil
	}
	var hits []Hit
	for _, t := range types.Types {
		if Skip[t.Name] || only != "" && t.Name != only {
			continue
		}
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			if h, ok := match(t, rec, words, some); ok {
				hits = append(hits, h)
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	return hits
}

func match(t *schema.Type, rec *store.Record, words []string, some bool) (Hit, bool) {
	title, body := texts(t, rec)
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
		h.Href = "/t/" + t.Name + "/" + rec.ID
	}
	return h, true
}

// texts is a record as words: its title, and everything else it says. A
// block's title is its component; its words are its props.
func texts(t *schema.Type, rec *store.Record) (title, body string) {
	if t.Name == "block" {
		name, _ := rec.Fields["component"].(string)
		return name, flatten(rec.Fields["props"])
	}
	var parts []string
	for _, f := range t.Fields {
		v := rec.Fields[f.Name]
		if v == nil {
			continue
		}
		// A date and a choice are found and shown as a person reads them,
		// not as they are stored; yes or no says nothing on its own.
		var s string
		switch f.Type {
		case "bool":
			continue
		case "datetime":
			s = when.Text(flatten(v))
		case "enum":
			s = f.ValueLabel(flatten(v))
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
	return title, strings.Join(parts, " ")
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

// snippet is the words around the first match, so the hit says why it is
// one, in roughly a line.
func snippet(body string, words []string) string {
	body = strings.Join(strings.Fields(body), " ")
	if body == "" {
		return ""
	}
	// Found in the folded text, cut from the text as it is: folding can
	// change a letter's length, so the places are mapped back, never reused.
	at := -1
	if spans := Spans(body, words); len(spans) > 0 {
		at = spans[0][0]
	}
	start := 0
	if at > 40 {
		start = at - 40
		for start > 0 && body[start] != ' ' {
			start--
		}
	}
	end := len(body)
	if end > start+120 {
		end = start + 120
		for end < len(body) && body[end] != ' ' {
			end++
		}
	}
	out := body[start:end]
	if start > 0 {
		out = "…" + strings.TrimLeft(out, " ")
	}
	if end < len(body) {
		out += "…"
	}
	return out
}
