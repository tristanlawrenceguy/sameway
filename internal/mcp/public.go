package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A connection from the internet reads what the owner published, and only
// that: describe tells it the published content types, and find_records
// and get_record answer for those types alone. Published, set by the
// command line, says which types those are just now.

// publicCall answers a call from the internet, or says it may not.
func (s *Server) publicCall(ctx context.Context, name string, args json.RawMessage) (string, bool, bool) {
	if !s.reachOf(ctx).public {
		return "", false, false
	}
	types := map[string]bool{}
	if s.Published != nil {
		types = s.Published()
	}
	switch name {
	case "search", "fetch":
		var a struct {
			Query string `json:"query"`
			ID    string `json:"id"`
			Type  string `json:"type"`
			Page  int    `json:"page"`
		}
		json.Unmarshal(args, &a)
		var v any
		var err error
		if name == "search" {
			v, err = s.searchPublished(ctx, types, a.Query, a.Type, a.Page)
		} else {
			v, err = s.fetchPublished(ctx, types, a.ID)
		}
		if err != nil {
			return err.Error(), true, true
		}
		raw, _ := json.Marshal(v)
		return string(raw), false, true
	}
	if name == "describe" {
		var out []any
		var names []string
		for t := range types {
			names = append(names, t)
		}
		sort.Strings(names)
		for _, n := range names {
			if t, ok := s.App.Types.Get(n); ok {
				out = append(out, map[string]any{"name": t.Name, "description": t.Description, "fields": t.JSONSchema()})
			}
		}
		raw, _ := json.MarshalIndent(map[string]any{"published": out, "tools": "find_records and get_record, for these types"}, "", "  ")
		return string(raw), false, true
	}
	var a struct {
		Type string `json:"type"`
	}
	json.Unmarshal(args, &a)
	if !types[a.Type] {
		return "only published content can be read here: " + joined(types), true, true
	}
	text, isErr := s.App.Chat.Call(name, args)
	return text, isErr, true
}

func joined(types map[string]bool) string {
	var names []string
	for t := range types {
		names = append(names, t)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "nothing is published"
	}
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}

// ChatGPT's connectors and deep research, and others after them, look for
// two tools by name: search, which finds documents, and fetch, which reads
// one. A connection from the internet has them over what is published:
// every published record is a document, by type/id, with its page as the
// address to cite.

var publicSearchTools = []tool{
	{Name: "search", Description: "Search what is published in this Sameway workspace: every published record whose words contain the query. Returns results with id, title and url; read one with fetch. " +
		"Search everything first: counts and total say how many were found of each kind, and said says it in words; then, if the counts show where, search again with type for that kind only. " +
		"At most 50 come back at a time; when pages is more than page, ask for the next page.",
		InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"query"},
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Words to find."},
				"type":  map[string]any{"type": "string", "description": "Only this published kind of thing, as describe lists them. Leave it out to search everything."},
				"page":  map[string]any{"type": "integer", "minimum": 1, "description": "Which page of results, from 1."}}}},
	{Name: "fetch", Description: "Read one published record in full, by the id search gave: its title, its text, its fields, and the address of its page.",
		InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"id"},
			"properties": map[string]any{"id": map[string]any{"type": "string", "description": "The id from search, such as note/abc123."}}}},
}

type baseKey struct{}

// siteOf is the address the connection came to, for the pages it cites.
func siteOf(ctx context.Context) string {
	base, _ := ctx.Value(baseKey{}).(string)
	return base
}

// document is a published record as search and fetch have it.
func (s *Server) document(ctx context.Context, typ string, r *store.Record) map[string]any {
	t, _ := s.App.Types.Get(typ)
	title, _ := r.Fields[t.Title].(string)
	if title == "" {
		title = typ
	}
	var lines []string
	meta := map[string]any{"type": typ, "updated": r.UpdatedAt.Format("2006-01-02")}
	for _, f := range t.Shown() {
		v := r.Fields[f.Name]
		if v == nil || v == "" || f.Name == t.Title || f.Type == "ref" {
			continue
		}
		label := f.Label
		if label == "" {
			label = f.Name
		}
		if s, ok := v.(string); ok && (f.Type == "text" || f.Type == "markdown") {
			lines = append(lines, s)
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %v", label, v))
		meta[f.Name] = v
	}
	return map[string]any{"id": typ + "/" + r.ID, "title": title, "text": strings.Join(lines, "\n\n"),
		"url": siteOf(ctx) + "/t/" + typ + "/" + r.ID, "metadata": meta}
}

// searchPublished is the search a person has, over what is published:
// every published record whose words have every word of the query,
// counted by kind, narrowed to one published kind when asked, a page at a
// time. results stays as ChatGPT's connectors read it, id, title and url;
// the counts and pages are beside it. A kind that is not published is
// refused, never answered with nothing found.
func (s *Server) searchPublished(ctx context.Context, types map[string]bool, query, only string, page int) (any, error) {
	var kinds []string
	for _, k := range search.Kinds(s.App.Types) {
		if types[k] {
			kinds = append(kinds, k)
		}
	}
	if only = strings.TrimSpace(only); only != "" && !slices.Contains(kinds, only) {
		return nil, errors.New(search.Refusal(only, kinds))
	}
	var hits []search.Hit
	for _, h := range search.FindAll(s.App.Store, s.App.Types, query) {
		if types[h.Type] && s.shows(ctx, h, search.Words(query)) {
			hits = append(hits, h)
		}
	}
	res := search.Narrow(hits, query, only, page)
	results := []map[string]any{}
	for _, h := range res.Hits {
		results = append(results, map[string]any{"id": h.Type + "/" + h.ID, "title": h.Title, "url": siteOf(ctx) + h.Href})
	}
	out := map[string]any{"results": results, "total": res.Total, "counts": res.Counts, "found": res.Found,
		"page": res.Page, "pages": res.Pages, "said": res.Said()}
	if only != "" {
		out["type"] = only
	}
	return out, nil
}

// shows says whether the words were found in what the internet may read
// of a record, its document: a field hidden from the pages is searched for
// its person, but it does not answer for a stranger.
func (s *Server) shows(ctx context.Context, h search.Hit, words []string) bool {
	r, err := s.App.Store.Get(h.Type, h.ID)
	if err != nil {
		return false
	}
	doc := s.document(ctx, h.Type, r)
	text := fmt.Sprint(doc["title"], " ", doc["text"])
	for _, w := range words {
		if len(search.Spans(text, []string{w})) == 0 {
			return false
		}
	}
	return true
}

// fetchPublished reads one published record by type/id.
func (s *Server) fetchPublished(ctx context.Context, types map[string]bool, id string) (any, error) {
	typ, rid, ok := strings.Cut(id, "/")
	if !ok || !types[typ] {
		return nil, fmt.Errorf("there is no published document %q; use an id from search", id)
	}
	r, err := s.App.Store.Get(typ, rid)
	if err != nil {
		return nil, fmt.Errorf("there is no published document %q; use an id from search", id)
	}
	return s.document(ctx, typ, r), nil
}

// onlyPublished is a tool as the internet is offered it: where it names
// the content types it takes, it names only the published ones, so the
// rest of the workspace is not so much as listed.
func (s *Server) onlyPublished(t tool) tool {
	schema := t.InputSchema
	props, _ := schema["properties"].(map[string]any)
	typ, _ := props["type"].(map[string]any)
	if typ == nil {
		return t
	}
	var names []string
	if s.Published != nil {
		for n := range s.Published() {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	nt := map[string]any{}
	for k, v := range typ {
		nt[k] = v
	}
	nt["enum"] = names
	np := map[string]any{}
	for k, v := range props {
		np[k] = v
	}
	np["type"] = nt
	ns := map[string]any{}
	for k, v := range schema {
		ns[k] = v
	}
	ns["properties"] = np
	t.InputSchema = ns
	return t
}
