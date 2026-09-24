package server

import (
	"fmt"
	"html/template"
	"net/http"
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
	title := "Search"
	if q != "" {
		hits := search.Find(s.app.Store, s.app.Types, q)
		title = fmt.Sprintf("Search: %s", q)
		if len(hits) == 0 {
			fmt.Fprintf(&b, `<h2>No results</h2><p class="sw-empty" role="status">No matches for %q — Try different words <a href="/search?q=&prompt=something.">here</a>.</p>`, q)
		} else {
			fmt.Fprintf(&b, `<p class="sw-muted sw-small" role="status">%s</p>`, template.HTMLEscapeString(count(len(hits))))
			fmt.Fprintf(&b, `<h2>Results</h2>`)
			fmt.Fprintf(&b, `<ol class="sw-stack" aria-label="Results for %s">`, template.HTMLEscapeString(q))
			for _, h := range hits {
				titleEsc := template.HTMLEscapeString(h.Title)
				typeEsc := template.HTMLEscapeString(h.Type)
				snippetEsc := template.HTMLEscapeString(h.Snippet)

				bodyHTML := ""
				if snippetEsc != "" {
					bodyHTML = fmt.Sprintf(`<div class="sw-card__body" data-prop="body"><p>%s</p></div>`, snippetEsc)
				}

				linkHTML := fmt.Sprintf(
					`<article class="sw-card" data-component="card">`+
						`<h2 class="sw-card__title" data-prop="title">`+
						`<a href="%s">%s<span class="sw-visually-hidden"> — %s</span></a>`+
						`</h2>`+
						`<p class="sw-card__meta">%s</p>`+
						`%s`+
						`</article>`,
					template.HTMLEscapeString(h.Href), titleEsc, typeEsc, typeEsc, bodyHTML,
				)

				fmt.Fprintf(&b, `<li class="sw-dotted" data-dot="%d">%s</li>`, s.dotOf(h.Type), linkHTML)
			}
			b.WriteString("</ol>")
		}
	}
	s.page(w, r, title, template.HTML(b.String()), pageOptions{JSONURL: "/api/search?q=" + template.URLQueryEscaper(q)})
}

func count(n int) string {
	if n == 1 {
		return "1 thing found"
	}
	return fmt.Sprintf("%d things found", n)
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
