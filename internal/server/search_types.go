package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// A search covers everything, and its results can be narrowed to one kind
// of thing: search.go draws the page, this is the narrowing.

// searchable is a kind a search can be narrowed to: one a person has a
// page of. What the system keeps for itself, a message or a canvas block,
// is not offered, and one the person has hidden is not either.
func (s *Server) searchable(name string) (*schema.Type, bool) {
	t, ok := s.app.Types.Get(name)
	if !ok || t.Internal || t.Hidden || search.Skip[name] {
		return nil, false
	}
	return t, true
}

// searchFrom is where Search in the navigation goes from this page: from
// a kind's own pages, its list or one of its records, a search of that
// kind, with everything one link away; from anywhere else, everything.
func (s *Server) searchFrom(r *http.Request) string {
	if rest, ok := strings.CutPrefix(r.URL.Path, "/t/"); ok {
		name, _, _ := strings.Cut(rest, "/")
		if _, ok := s.searchable(name); ok {
			return searchURL("", name)
		}
	}
	return "/search"
}

// searchURL is the address of a search for q, narrowed to only when named.
func searchURL(q, only string) string {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	if only != "" {
		v.Set("type", only)
	}
	if len(v) == 0 {
		return "/search"
	}
	return "/search?" + v.Encode()
}

// widen is the way from a narrowed search to the same words everywhere.
func widen(q string) string {
	return fmt.Sprintf(`<a class="sw-link" href="%s">Search everything</a>`, template.HTMLEscapeString(searchURL(q, "")))
}

// kinds is the row of links that narrows the results to one kind, each
// with how many it found, counted from the one search: All first, then
// the most found. A kind with nothing found is not offered, unless it is
// the one shown. It is a row of links, so it needs no script, and the one
// here is marked as the current page.
func (s *Server) kinds(q, only string, all []search.Hit) template.HTML {
	counts := search.Counts(all)
	var names []string
	for name := range counts {
		if _, ok := s.searchable(name); ok && name != only {
			names = append(names, name)
		}
	}
	// One kind found is the same list as All: no row saying so.
	if len(all) == 0 || only == "" && len(names) < 2 {
		return ""
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	if only != "" {
		names = append([]string{only}, names...)
	}
	items := []any{map[string]any{"href": searchURL(q, ""), "label": fmt.Sprintf("All (%d)", len(all)), "current": only == ""}}
	for _, name := range names {
		items = append(items, map[string]any{"href": searchURL(q, name), "label": fmt.Sprintf("%s (%d)", capitalize(plural(name)), counts[name]), "current": name == only})
	}
	return s.component("tabs", map[string]any{"label": "Kinds of result", "items": items})
}

// searchRefused is a search narrowed to a kind there is none of, or one
// kept out of search: said plainly, with the same words everywhere.
func (s *Server) searchRefused(w http.ResponseWriter, r *http.Request, q, only string) {
	body := fmt.Sprintf(`<p>There is no kind of thing called “%s” to search in. %s</p>`, template.HTMLEscapeString(only), widen(q))
	s.page(w, r, "Search", template.HTML(body+searchForm(s, "Search", "", q, "")), pageOptions{Status: http.StatusBadRequest, Said: "Search: no kind called " + only})
}

// apiSearch is the same search for an agent.
func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	hits := search.FindOf(s.app.Store, s.app.Types, q, r.URL.Query().Get("type"))
	if hits == nil {
		hits = []search.Hit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "count": len(hits), "hits": hits})
}

// marked is text with the words searched for marked, so a person sees why
// each result is one: bold and on a wash, not by colour alone.
func marked(text string, words []string) template.HTML {
	var b strings.Builder
	at := 0
	for _, sp := range search.Spans(text, words) {
		b.WriteString(template.HTMLEscapeString(text[at:sp[0]]))
		b.WriteString(`<mark class="sw-search__hit">` + template.HTMLEscapeString(text[sp[0]:sp[1]]) + `</mark>`)
		at = sp[1]
	}
	b.WriteString(template.HTMLEscapeString(text[at:]))
	return template.HTML(b.String())
}
