package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// searchPage is one search over everything the person has: every record
// of every content type and every block on the canvas. A form is the
// better thing here: one field, one job. It is a GET, so a search is a
// link that can be kept and shared.
func (s *Server) searchPage(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var b strings.Builder
	b.WriteString(`<form method="get" action="/search" role="search" class="sw-stack sw-compose">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Search", "name": "q", "value": q, "type": "search", "hint": "Any words in a note, an event, an action, or a block on the canvas."})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Search", "type": "submit"})))
	b.WriteString(`</form>`)
	title, said := "Search", ""
	pg := paged{page: 1, pages: 1}
	if q != "" {
		hits := search.Find(s.app.Store, s.app.Types, q)
		title = trimTitle(fmt.Sprintf("Search: %s", q))
		// The window title names the page and says what the search found,
		// Search: plumber, no results. It is the first
		// thing a screen reader says when the results page arrives, which a
		// status region on a fresh page is not.
		said = trimTitle(fmt.Sprintf("Search: %s, %s", q, strings.ToLower(results(len(hits)))))
		if len(hits) == 0 {
			b.WriteString(string(s.component("empty", map[string]any{
				"title": "No results", "message": fmt.Sprintf("Nothing matches “%s”. Try different words, or", q),
				"action": map[string]any{"href": "/chat?prompt=" + url.QueryEscape("Find "+q), "label": "ask the assistant"},
			})))
		} else {
			fmt.Fprintf(&b, `<p class="sw-muted sw-small">%s</p>`, template.HTMLEscapeString(count(len(hits))))
			fmt.Fprintf(&b, `<h2>Results</h2>`)
			fmt.Fprintf(&b, `<ol class="sw-stack" aria-label="Results for %s">`, template.HTMLEscapeString(q))
			pg = pageOf(r, len(hits), searchPageSize)
			for _, h := range hits[pg.lo:pg.hi] {
				typeEsc := template.HTMLEscapeString(capitalize(h.Type))
				snippetEsc := template.HTMLEscapeString(h.Snippet)
				// The whole title: a result is recognised by it, and a
				// tooltip holding the rest is out of reach of a keyboard or
				// a finger.
				titleEsc := template.HTMLEscapeString(h.Title)

				bodyHTML := ""
				if snippetEsc != "" {
					bodyHTML = fmt.Sprintf(`<div class="sw-card__body" data-prop="body"><p>%s</p></div>`, snippetEsc)
				}

				linkHTML := fmt.Sprintf(
					`<article class="sw-card" data-component="card">`+
						`<h3 class="sw-card__title" data-prop="title">`+
						`<a href="%s">%s<span class="sw-visually-hidden"> — %s</span></a>`+
						`</h3>`+
						`<p class="sw-card__meta">%s</p>`+
						`%s`+
						`</article>`,
					template.HTMLEscapeString(h.Href), titleEsc, typeEsc, typeEsc, bodyHTML,
				)

				fmt.Fprintf(&b, `<li class="sw-dotted" data-dot="%d">%s</li>`, s.dotOf(h.Type), linkHTML)
			}
			b.WriteString("</ol>")
			b.WriteString(string(s.pageNav(r, pg, "Pages of results")))
		}
	}
	s.page(w, r, pg.title(title), template.HTML(b.String()), pageOptions{JSONURL: "/api/search?q=" + template.URLQueryEscaper(q), Said: said})
}

func count(n int) string {
	if n == 1 {
		return "1 thing found"
	}
	return fmt.Sprintf("%d things found", n)
}

// results is how many a search found, as the window title ends: No
// results, 1 result, 3 results.
func results(n int) string {
	switch n {
	case 0:
		return "No results"
	case 1:
		return "1 result"
	}
	return fmt.Sprintf("%d results", n)
}

// apiSearch is the same search for an agent.
func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	hits := search.Find(s.app.Store, s.app.Types, q)
	if hits == nil {
		hits = []search.Hit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "count": len(hits), "hits": hits})
}
