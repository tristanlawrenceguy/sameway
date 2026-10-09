package search

import (
	"strings"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Search by meaning (internal/meaning) joins the one search here, so the
// page, the assistant, the API and AI services all find "Boiler repair,
// call Marek" when asked for the plumber. A workspace that has it says so
// with UseMeaning; its matches by meaning come after the matches by word,
// those already found left out, each marked Near.

var nearFor sync.Map // *store.Store -> func(q string) []string

// UseMeaning has a workspace's searches also find records near in meaning:
// find answers type/id keys, nearest first. Nil stops it.
func UseMeaning(st *store.Store, find func(q string) []string) {
	if find == nil {
		nearFor.Delete(st)
		return
	}
	nearFor.Store(st, find)
}

// Doc is a record as words, for meaning to read.
type Doc struct {
	Key, Text string
}

// Docs are every searchable record's words, keyed type/id.
func Docs(st *store.Store, types *schema.Set, r Reader) []Doc {
	var out []Doc
	for _, t := range types.Types {
		if Skip[t.Name] || t.Internal {
			continue
		}
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			title, body := texts(t, rec, r)
			if text := strings.TrimSpace(title + "\n" + body); text != "" {
				out = append(out, Doc{Key: t.Name + "/" + rec.ID, Text: clipText(text, 2000)})
			}
		}
	}
	return out
}

func clipText(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// near adds a workspace's matches by meaning to its matches by word.
func near(st *store.Store, types *schema.Set, q string, hits []Hit, r Reader) []Hit {
	f, ok := nearFor.Load(st)
	if !ok {
		return hits
	}
	have := map[string]bool{}
	for _, h := range hits {
		have[h.Type+"/"+h.ID] = true
	}
	for _, key := range f.(func(string) []string)(q) {
		typ, id, ok := strings.Cut(key, "/")
		if !ok || have[key] {
			continue
		}
		t, ok := types.Get(typ)
		if !ok {
			continue
		}
		rec, err := st.Get(typ, id)
		if err != nil {
			continue
		}
		title, body := texts(t, rec, r)
		hits = append(hits, Hit{Type: typ, ID: id, Title: title, Snippet: clipText(strings.TrimSpace(body), 140), Href: "/t/" + typ + "/" + id, Near: true})
	}
	return hits
}
