package look

import (
	"fmt"
	"regexp"
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
		alt, ok := htmltest.Attr(n, "alt")
		if !ok {
			out = append(out, "img without alt")
		} else if fileNamed(alt) {
			// A file's name says nothing of what a picture shows.
			out = append(out, fmt.Sprintf("img whose alt is a file name: %q", alt))
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
	out = append(out, setUpWrong(doc)...)
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

// fileNamed says an alt is a file's name rather than words: a camera's
// name for a picture, or one ending in an image file's extension.
func fileNamed(alt string) bool {
	a := strings.ToLower(strings.TrimSpace(alt))
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".svg"} {
		if strings.HasSuffix(a, ext) {
			return true
		}
	}
	return cameraName.MatchString(a)
}

var cameraName = regexp.MustCompile(`^(img|dsc|dscn|pxl|photo|image|screenshot)[_ -]?\d{2,}`)

// setUpWrong is every block on the page that could only say it is set up
// wrong, by block, with what is wrong: a person reads "Ask the assistant
// to fix it", and an agent that looks should hear the same, not a page
// that holds together. A problem outside a block, such as an example on
// the design page, is not one.
func setUpWrong(doc *htmltest.Doc) []string {
	var out []string
	for _, n := range doc.WithAttr("data-component", "problem") {
		where := ""
		for p := n.Parent; p != nil && where == ""; p = p.Parent {
			if id, ok := htmltest.Attr(p, "data-block-id"); ok {
				component, _ := htmltest.Attr(p, "data-block-component")
				where = fmt.Sprintf("block %s (%s)", id, component)
			} else if class, _ := htmltest.Attr(p, "class"); strings.Contains(" "+class+" ", " sw-focus__body ") {
				where = "this block"
			}
		}
		if where == "" {
			continue
		}
		why := ""
		for _, d := range doc.Elements("details") {
			if within(d, n) {
				for c := d.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "p" {
						why = htmltest.Text(c)
					}
				}
			}
		}
		out = append(out, fmt.Sprintf("%s cannot be shown as it is set up: %s", where, why))
	}
	return out
}

// within says whether n is inside of.
func within(n, of *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p == of {
			return true
		}
	}
	return false
}
