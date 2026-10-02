package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// A search covers everything, and its results can be narrowed to one kind
// of thing: search.go draws the page, this is the narrowing.

// searchable is a kind a search can be narrowed to: one a person has a
// page of. What the system keeps for itself, a message or a canvas block,
// is not offered, and one the person has hidden is not either.
// The rule is search's own, so an agent narrowing is held to it too.
func (s *Server) searchable(name string) (*schema.Type, bool) {
	return search.Searchable(s.app.Types, name)
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
// the one shown. It is the filters component's links, one choice among a
// few, so it needs no script, and the one here is marked as the current
// page.
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
	opts := []any{map[string]any{"href": searchURL(q, ""), "label": "All", "count": len(all), "selected": only == ""}}
	for _, name := range names {
		opts = append(opts, map[string]any{"href": searchURL(q, name), "label": capitalize(plural(name)), "count": counts[name], "selected": name == only})
	}
	return s.component("filters", map[string]any{"shape": "links", "label": "Kinds of result", "choices": []any{map[string]any{"label": "Kind", "options": opts}}})
}

// searchRefused is a search narrowed to a kind there is none of, or one
// kept out of search: said plainly, with the same words everywhere.
func (s *Server) searchRefused(w http.ResponseWriter, r *http.Request, q, only string) {
	body := fmt.Sprintf(`<p>There is no kind of thing called “%s” to search in. %s</p>`, template.HTMLEscapeString(only), widen(q))
	s.page(w, r, "Search", template.HTML(body+searchForm(s, "Search", "", q, "")), pageOptions{Status: http.StatusBadRequest, Said: "Search: no kind called " + only})
}

// apiSearch is the same search for an agent: everything, counted by kind,
// narrowed by ?type= when asked, a page of search.Limit at a time with
// ?page=. A kind it cannot narrow to is a 400 in the page's words, never
// an empty list an agent would take for nothing there.
func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	only := strings.TrimSpace(r.URL.Query().Get("type"))
	if _, ok := s.searchable(only); only != "" && !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": apiError{Code: "bad_request", Message: search.Refusal(only, search.Kinds(s.app.Types))}})
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	found, some := search.Matches(s.app.Store, s.app.Types, q)
	res := search.Narrow(found, q, only, page)
	res.Some = some
	// Each hit as it always was, with who wrote it beside it.
	type written struct {
		search.Hit
		WrittenBy string `json:"written_by"`
	}
	writers := s.app.Chat.Writers()
	hits := make([]written, 0, len(res.Hits))
	for _, h := range res.Hits {
		hits = append(hits, written{h, writers.OfID(h.Type, h.ID).Words})
	}
	out := map[string]any{"query": res.Query, "count": len(res.Hits), "hits": hits, "total": res.Total, "counts": res.Counts,
		"found": res.Found, "page": res.Page, "pages": res.Pages, "said": res.Said(), "some": res.Some,
		"untrusted": "each hit's title and snippet were written by its written_by: " + chat.Untrusted}
	if only != "" {
		out["type"] = only
	}
	if res.Page < res.Pages {
		v := r.URL.Query()
		v.Set("page", strconv.Itoa(res.Page+1))
		out["next"] = "/api/search?" + v.Encode()
	}
	writeJSON(w, http.StatusOK, out)
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
