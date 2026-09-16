package look

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// problems is the structural contract: what the tests hold every
// component to, applied to a whole page or a fragment.
func problems(doc *htmltest.Doc, o *Outline, page bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, n := range doc.WithAttr("id", "") {
		id, _ := htmltest.Attr(n, "id")
		if seen[id] {
			out = append(out, fmt.Sprintf("duplicate id %q", id))
		}
		seen[id] = true
	}
	for _, key := range []string{"aria-describedby", "aria-labelledby", "aria-controls"} {
		for _, n := range doc.WithAttr(key, "") {
			refs, _ := htmltest.Attr(n, key)
			for _, id := range strings.Fields(refs) {
				if doc.ByID(id) == nil {
					out = append(out, fmt.Sprintf("%s points at missing id %q", key, id))
				}
			}
		}
	}
	for _, l := range doc.Elements("label") {
		if target, ok := htmltest.Attr(l, "for"); ok && doc.ByID(target) == nil {
			out = append(out, fmt.Sprintf("label for=%q has no target", target))
		}
	}
	for _, tag := range []string{"input", "select", "textarea"} {
		for _, n := range doc.Elements(tag) {
			if t, _ := htmltest.Attr(n, "type"); tag == "input" && (t == "hidden" || t == "submit" || t == "button") {
				continue
			}
			if doc.AccessibleName(n) == "" {
				out = append(out, fmt.Sprintf("<%s> has no label", tag))
			}
		}
	}
	for _, n := range doc.Elements("table") {
		// A table is named by its caption or a label, never by its cells.
		named := false
		for _, key := range []string{"aria-label", "aria-labelledby"} {
			if v, ok := htmltest.Attr(n, key); ok && v != "" {
				named = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "caption" && htmltest.Text(c) != "" {
				named = true
			}
		}
		if !named {
			out = append(out, "table has no caption")
		}
	}
	for _, n := range doc.Elements("img") {
		if _, ok := htmltest.Attr(n, "alt"); !ok {
			out = append(out, "img without alt")
		}
	}
	for _, c := range o.Controls {
		if c.Name == "" && (c.Kind == "link" || c.Kind == "button") {
			out = append(out, fmt.Sprintf("a %s with no name", c.Kind))
		}
	}
	doc.Walk(func(n *html.Node) {
		if v, _ := htmltest.Attr(n, "aria-hidden"); v == "true" && htmltest.Focusable(n) {
			out = append(out, fmt.Sprintf("<%s> is focusable but aria-hidden", n.Data))
		}
	})
	last := 0
	for _, h := range o.Headings {
		if last > 0 && h.Level > last+1 {
			out = append(out, fmt.Sprintf("heading level skips from %d to %d at %q", last, h.Level, h.Text))
		}
		last = h.Level
	}
	if page {
		h1 := 0
		for _, h := range o.Headings {
			if h.Level == 1 {
				h1++
			}
		}
		if h1 != 1 {
			out = append(out, fmt.Sprintf("%d h1 headings; a page has one", h1))
		}
		main := false
		for _, l := range o.Landmarks {
			main = main || l.Role == "main"
		}
		if !main {
			out = append(out, "no main landmark")
		}
		if o.Title == "" {
			out = append(out, "no title")
		}
	}
	return out
}
