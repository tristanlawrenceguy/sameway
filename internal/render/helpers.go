package render

import (
	"math"
	"net/url"
	"strconv"
)

// Template helpers for the meter and pagination components.

// dict builds a map from pairs, so a template can hand props to a child:
// {{child (dict "component" "meter" "props" (dict "value" .amount))}}.
func dict(pairs ...any) map[string]any {
	m := map[string]any{}
	for i := 0; i+1 < len(pairs); i += 2 {
		if k, ok := pairs[i].(string); ok {
			m[k] = pairs[i+1]
		}
	}
	return m
}

// percent is value against max, 0 to 100, for a bar's width.
func percent(value, max any) int {
	v, m := numberOf(value), numberOf(max)
	if m <= 0 {
		return 0
	}
	return int(math.Round(math.Max(0, math.Min(1, v/m)) * 100))
}

// atMost is value, never past max: a meter's value may not pass its end.
func atMost(value, max any) float64 {
	return math.Min(numberOf(value), numberOf(max))
}

// paging is what the pagination component draws: the step back, the step
// on, and the pages by number with gaps, each with its address.
type paging struct {
	Prev, Next string
	Items      []pageItem
}

type pageItem struct {
	N       int
	Href    string
	Current bool
	Gap     bool
}

func pagesOf(href string, page, pages any) paging {
	p, n := num(page), num(pages)
	var out paging
	if p > 1 {
		out.Prev = pageHref(href, p-1)
	}
	if p < n {
		out.Next = pageHref(href, p+1)
	}
	for _, i := range pageWindow(p, n) {
		out.Items = append(out.Items, pageItem{N: i, Href: pageHref(href, i), Current: i == p, Gap: i == 0})
	}
	return out
}

// pageWindow is the pages to offer by number: the first, the last, and two
// either side of the current one; 0 marks a gap between them.
func pageWindow(p, n int) []int {
	// The first, the last, and one either side of this one; a gap that
	// would stand for a single page shows that page instead, so the row is
	// never more than seven long and a gap always hides two or more.
	near := func(i int) bool { return i == 1 || i == n || (i >= p-1 && i <= p+1) }
	var out []int
	for i := 1; i <= n; i++ {
		if near(i) || (near(i-1) && near(i+1)) {
			out = append(out, i)
		} else if len(out) > 0 && out[len(out)-1] != 0 {
			out = append(out, 0)
		}
	}
	return out
}

// pageHref is the address of page n of href, which may carry a query of its
// own; the first page is href with no page at all.
func pageHref(href string, p int) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	q := u.Query()
	q.Del("page")
	if p > 1 {
		q.Set("page", strconv.Itoa(p))
	}
	u.RawQuery = q.Encode()
	return u.String()
}
