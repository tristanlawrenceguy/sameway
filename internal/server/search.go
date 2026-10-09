package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// searchPage is one search over everything the person has: every record
// of every content type and every block on the canvas. A form is the
// better thing here: one field, one job. It is a GET, so a search is a
// link that can be kept and shared.
//
// Everything is always searched; ?type=task narrows what is shown to one
// kind, counted from the same search, and says so in the heading, the
// window title and a sentence with the way back to everything first.
// Search from one of a type's pages arrives narrowed to it (searchFrom).
func (s *Server) searchPage(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	only := strings.TrimSpace(r.URL.Query().Get("type"))
	if only != "" {
		if _, ok := s.searchable(only); !ok {
			s.searchRefused(w, r, q, only)
			return
		}
	}
	// lead is said above the box: the scope, and the way to everything.
	var b strings.Builder
	lead := ""
	label, hint, in := "Search", "Any words in a note, an event, an action, or a block on the canvas.", ""
	if only != "" {
		label, hint, in = "Search "+schema.Plural(only), "Any words in your "+schema.Plural(only)+".", " in "+schema.Plural(only)
		if q == "" {
			lead = fmt.Sprintf(`<p class="sw-search__scope">Searching %s only. %s</p>`, template.HTMLEscapeString(schema.Plural(only)), widen(q))
		}
	}
	title, said := label, label
	pg := paged{page: 1, pages: 1}
	if q != "" {
		all, some := search.Matches(s.app.Store, s.app.Types, q, s.reader())
		hits := all
		if only != "" {
			hits = search.Of(all, only)
		}
		words := search.Words(q)
		title = trim.Title(fmt.Sprintf("Search: %s%s", q, in))
		// The window title names the page and says what the search found,
		// Search: plumber in notes, 3 results. It is the first thing a
		// screen reader says when the results page arrives, which a status
		// region on a fresh page is not.
		said = trim.Title(fmt.Sprintf("Search: %s%s, %s", q, in, strings.ToLower(results(len(hits)))))
		if only != "" && len(all) > 0 {
			// The scope in words, and the way back to everything first:
			// a narrowed search nobody noticed looks like a missing thing.
			lead = fmt.Sprintf(`<p class="sw-search__scope">Showing %s only, %d of %d found. %s</p>`,
				template.HTMLEscapeString(schema.Plural(only)), len(hits), len(all), widen(q))
		}
		if some && len(all) > 0 {
			b.WriteString(`<p class="sw-muted">` + search.SomeWords + `</p>`)
		}
		b.WriteString(string(s.kinds(q, only, all)))
		switch {
		case len(all) == 0:
			where := ""
			if only != "" {
				where = ", in " + schema.Plural(only) + " or anywhere else"
			}
			b.WriteString(string(s.component("empty", map[string]any{
				"title": "No results", "message": fmt.Sprintf("Nothing matches “%s”%s. Try different words, or", q, where),
				"action": map[string]any{"href": "/chat?prompt=" + url.QueryEscape("Find "+q), "label": "ask the assistant"},
			})))
		case len(hits) == 0:
			b.WriteString(string(s.component("empty", map[string]any{
				"title": "No " + schema.Plural(only) + " match", "message": fmt.Sprintf("Nothing in %s matches “%s”, but %s. You can", schema.Plural(only), q, elsewhere(len(all))),
				"action": map[string]any{"href": searchURL(q, ""), "label": "search everything"},
			})))
		default:
			if only == "" {
				fmt.Fprintf(&b, `<p class="sw-muted sw-small">%s</p>`, template.HTMLEscapeString(count(len(hits))))
			}
			b.WriteString(`<h2>Results</h2>`)
			// Page 2 counts on from where page 1 ended, 21, not from 1 again;
			// the pages keep the kind, as the kinds keep the words.
			pg = pageOf(r, len(hits), searchPageSize)
			fmt.Fprintf(&b, `<ol class="sw-stack" start="%d" aria-label="Results for %s">`, pg.lo+1, template.HTMLEscapeString(q))
			shown := hits[pg.lo:pg.hi]
			told := s.hitsApart(shown)
			for i, h := range shown {
				b.WriteString(s.hitItem(h, words, told[i]))
			}
			b.WriteString("</ol>")
			b.WriteString(string(s.pageNav(r, pg, "Pages of results")))
		}
	}
	jsonURL := "/api/search?q=" + template.URLQueryEscaper(q)
	if only != "" {
		jsonURL += "&type=" + template.URLQueryEscaper(only)
	}
	body := lead + searchForm(s, label, hint, q, only) + b.String()
	s.page(w, r, title, template.HTML(body), pageOptions{JSONURL: jsonURL, Said: pg.title(said)})
}

// searchForm is the page's own search: a labelled field, the kind it is
// narrowed to carried along unseen, and the button.
func searchForm(s *Server, label, hint, q, only string) string {
	var b strings.Builder
	b.WriteString(`<form method="get" action="/search" role="search" class="sw-stack sw-compose">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": label, "name": "q", "value": q, "type": "search", "hint": hint})))
	if only != "" {
		fmt.Fprintf(&b, `<input type="hidden" name="type" value="%s">`, template.HTMLEscapeString(only))
	}
	b.WriteString(string(s.component("button", map[string]any{"label": "Search", "type": "submit"})))
	b.WriteString(`</form>`)
	return b.String()
}

// hitItem is one result: its title with the words marked, what kind it is,
// and the words around the match. told tells it from another result with
// its title and kind, or is "".
func (s *Server) hitItem(h search.Hit, words []string, told string) string {
	typeEsc := template.HTMLEscapeString(capitalize(schema.Words(h.Type)))
	if h.Near {
		typeEsc += ", close in meaning" // found by what it is about (search_meaning.go)
	}
	bodyHTML := ""
	if snippet := string(marked(h.Snippet, words)); snippet != "" {
		bodyHTML = fmt.Sprintf(`<div class="sw-card__body" data-prop="body"><p>%s</p></div>`, snippet)
	}
	// The whole title: a result is recognised by it, and a tooltip holding
	// the rest is out of reach of a keyboard or a finger.
	card := fmt.Sprintf(
		`<article class="sw-card" data-component="card">`+
			`<h3 class="sw-card__title" data-prop="title">`+
			`<a href="%s">%s<span class="sw-visually-hidden"> — %s%s</span></a>`+
			`</h3>`+
			`<p class="sw-card__meta">%s</p>`+
			`%s`+
			`</article>`,
		template.HTMLEscapeString(h.Href), marked(h.Title, words), typeEsc, template.HTMLEscapeString(withContext("", told)), typeEsc, bodyHTML,
	)
	return fmt.Sprintf(`<li class="sw-dotted" data-dot="%d">%s</li>`, s.dotOf(h.Type), card)
}

func count(n int) string {
	if n == 1 {
		return "1 thing found"
	}
	return fmt.Sprintf("%d things found", n)
}

// elsewhere is how many were found outside the kind shown: 1 thing
// elsewhere does, 4 things elsewhere do.
func elsewhere(n int) string {
	if n == 1 {
		return "1 thing elsewhere does"
	}
	return fmt.Sprintf("%d things elsewhere do", n)
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
