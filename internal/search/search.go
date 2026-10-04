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

// Matches is what a search finds, the one way for the search page, the
// assistant, the API, the command line and AI services: everything with
// every word, or, when nothing has them all and there are several, what
// has some of them, most first, with some true so it is said.
func Matches(st *store.Store, types *schema.Set, q string) (hits []Hit, some bool) {
	if hits = find(st, types, q, "", false); len(hits) > 0 || len(Words(q)) < 2 {
		return hits, false
	}
	return find(st, types, q, "", true), true
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

func find(st *store.Store, types *schema.Set, q, only string, some bool) []Hit {
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
		// A record it points at is an id, no word of anyone's; it is left out.
		if f.Type == "ref" || f.RefList() {
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
	// Each value apart, as a reader needs: "Done · Fri 9 Oct · home".
	return title, strings.Join(parts, " · ")
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
