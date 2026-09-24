package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
)

// Long lists are read a page at a time; an agent reading the JSON gets them
// whole.
const (
	listPageSize   = 50
	searchPageSize = 20
)

// paged is one page of a list of n things: which page, how many pages, and
// the slice [lo:hi] of the things on it. A page past the end is the last.
type paged struct {
	page, pages, lo, hi int
}

func pageOf(r *http.Request, n, size int) paged {
	pages := (n + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	p = min(max(p, 1), pages)
	return paged{page: p, pages: pages, lo: (p - 1) * size, hi: min(p*size, n)}
}

// title says which page this is when there is more than one, so each has
// its own title.
func (p paged) title(t string) string {
	if p.pages < 2 {
		return t
	}
	return fmt.Sprintf("%s, page %d of %d", t, p.page, p.pages)
}

// nav is the pagination component for these pages, or nothing for one.
func (s *Server) pageNav(r *http.Request, p paged, label string) template.HTML {
	if p.pages < 2 {
		return ""
	}
	return s.component("pagination", map[string]any{"href": r.URL.RequestURI(), "page": p.page, "pages": p.pages, "label": label})
}
