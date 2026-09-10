// Package htmltest holds small helpers for asserting on rendered HTML in
// tests: parse a fragment or page, find elements, read attributes, and
// compute the accessible name the way a screen reader or an agent would.
package htmltest

import (
	"strings"

	"golang.org/x/net/html"
)

// Doc is a parsed HTML tree.
type Doc struct{ Root *html.Node }

// Parse parses a page or fragment.
func Parse(src string) (*Doc, error) {
	n, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	return &Doc{Root: n}, nil
}

// Attr returns an attribute value and whether it is present.
func Attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// Walk visits every element node.
func (d *Doc) Walk(fn func(n *html.Node)) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			fn(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(d.Root)
}

// Elements returns every element with the given tag name.
func (d *Doc) Elements(tag string) []*html.Node {
	var out []*html.Node
	d.Walk(func(n *html.Node) {
		if n.Data == tag {
			out = append(out, n)
		}
	})
	return out
}

// WithAttr returns every element whose attribute equals value. An empty
// value matches any element that has the attribute.
func (d *Doc) WithAttr(key, value string) []*html.Node {
	var out []*html.Node
	d.Walk(func(n *html.Node) {
		if v, ok := Attr(n, key); ok && (value == "" || v == value) {
			out = append(out, n)
		}
	})
	return out
}

// ByID finds one element by id.
func (d *Doc) ByID(id string) *html.Node {
	els := d.WithAttr("id", id)
	if len(els) == 0 {
		return nil
	}
	return els[0]
}

// Text returns the concatenated, space-normalised text content of a node.
func Text(n *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// Focusable reports whether an element is in the keyboard tab order.
func Focusable(n *html.Node) bool {
	if _, disabled := Attr(n, "disabled"); disabled {
		return false
	}
	switch n.Data {
	case "button", "input", "select", "textarea":
		return true
	case "a":
		_, ok := Attr(n, "href")
		return ok
	}
	ti, ok := Attr(n, "tabindex")
	return ok && ti != "-1"
}

// AccessibleName approximates the accessible name computation for the
// elements Sameway renders: aria-labelledby, aria-label, an associated
// label, a caption, or the text content.
func (d *Doc) AccessibleName(n *html.Node) string {
	if ids, ok := Attr(n, "aria-labelledby"); ok {
		var parts []string
		for _, id := range strings.Fields(ids) {
			if el := d.ByID(id); el != nil {
				parts = append(parts, Text(el))
			}
		}
		return strings.Join(parts, " ")
	}
	if v, ok := Attr(n, "aria-label"); ok {
		return v
	}
	if id, ok := Attr(n, "id"); ok {
		for _, l := range d.Elements("label") {
			if f, _ := Attr(l, "for"); f == id {
				return Text(l)
			}
		}
	}
	if n.Data == "table" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Data == "caption" {
				return Text(c)
			}
		}
	}
	return Text(n)
}
