package prose

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
)

// A long piece of writing has its contents at the top: each heading is
// given an id of its own, numbered after the record it is in so two pieces
// on one page never share one, and the contents list leads to them.

// Heading is one heading of a piece, as the contents list names it.
type Heading struct {
	ID    string
	Text  string
	Level int
}

var (
	headingTag = regexp.MustCompile(`<h([1-6])>(.*?)</h[1-6]>`)
	anyTag     = regexp.MustCompile(`<[^>]*>`)
)

// Anchored renders Markdown as Render does, each heading with an id made
// from prefix and its place, and returns the headings in order.
func Anchored(md string, base int, prefix string) (template.HTML, []Heading) {
	var heads []Heading
	out := headingTag.ReplaceAllStringFunc(string(Render(md, base)), func(tag string) string {
		m := headingTag.FindStringSubmatch(tag)
		id := fmt.Sprintf("%s-%d", prefix, len(heads)+1)
		heads = append(heads, Heading{ID: id, Text: strings.TrimSpace(html.UnescapeString(anyTag.ReplaceAllString(m[2], ""))), Level: int(m[1][0] - '0')})
		return fmt.Sprintf(`<h%s id="%s">%s</h%s>`, m[1], id, m[2], m[1])
	})
	return template.HTML(out), heads
}

// Headings are the headings Anchored gives, without rendering for a page.
func Headings(md string, base int, prefix string) []Heading {
	_, h := Anchored(md, base, prefix)
	return h
}
