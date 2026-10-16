package look

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// Passage is one run of a page's words as a person reads them: a
// paragraph, an item of a list, a cell, a term or its description, a
// caption, under the heading it falls under. Agents checking a page asked
// fourteen times for what a page says, its ledes, badges, dates and
// messages, and got only its headings and controls.
type Passage struct {
	Under string `json:"under,omitempty"`
	Text  string `json:"text"`
	// Links are the links in the passage's words, each by its words and
	// where it goes: read as text, an agent could not tell which words
	// were a link, or to what (crew, task 0279).
	Links []PassageLink `json:"links,omitempty"`
}

// PassageLink is one link inside a passage.
type PassageLink struct {
	Words string `json:"words"`
	Href  string `json:"href"`
}

// blocks are the elements whose words read as one passage.
var blocks = map[string]bool{"p": true, "div": true, "li": true, "dt": true, "dd": true, "td": true, "th": true,
	"caption": true, "figcaption": true, "blockquote": true, "pre": true, "legend": true, "summary": true}

// passages is every passage of n, in reading order. What is hidden from
// everyone (hidden, aria-hidden, a script) is not read; what is hidden
// only from the eye is, as a screen reader says it.
func passages(n *html.Node) []Passage {
	var out []Passage
	under := ""
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if skipped(n) {
				return
			}
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				under = htmltest.Text(n)
				return
			}
			if blocks[n.Data] && !holdsBlock(n) {
				// A passage that is only a link or a button is already among
				// the controls; read twice it buries the words.
				if t := readText(n); t != "" && (n.Data != "li" || wordsBeside(n)) {
					out = append(out, Passage{Under: under, Text: t, Links: passageLinks(n)})
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func skipped(n *html.Node) bool {
	switch n.Data {
	case "script", "style", "template", "head", "noscript":
		return true
	}
	if _, ok := htmltest.Attr(n, "hidden"); ok {
		return true
	}
	if v, _ := htmltest.Attr(n, "aria-hidden"); v == "true" {
		return true
	}
	_, unseen := htmltest.Attr(n, "data-look-unseen")
	return unseen
}

// holdsBlock says a passage has passages inside it, which are read on
// their own instead.
func holdsBlock(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && !skipped(c) && (blocks[c.Data] || holdsBlock(c)) {
			return true
		}
	}
	return false
}

// readText is an element's words with a space wherever two elements meet,
// as a reader hears them apart, and no run of spaces.
func readText(n *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(n.Data)
		case html.ElementNode:
			if skipped(n) {
				return
			}
			if box := boxSaid(n); box != "" {
				b.WriteString(" " + box + " ")
				return
			}
			if n.Data == "br" || n.Data == "img" {
				if alt, _ := htmltest.Attr(n, "alt"); alt != "" {
					b.WriteString(" " + alt)
				}
				b.WriteString(" ")
				return
			}
			b.WriteString(" ")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			b.WriteString(" ")
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c)
	}
	s := strings.Join(strings.Fields(b.String()), " ")
	// A space put where two elements meet is not one before punctuation.
	for _, p := range []string{".", ",", ":", ";", ")", "!", "?"} {
		s = strings.ReplaceAll(s, " "+p, p)
	}
	return strings.ReplaceAll(s, "( ", "(")
}

// operable are the elements whose words a reader meets as a control.
var operable = map[string]bool{"a": true, "button": true, "input": true, "select": true, "textarea": true}

// wordsBeside says a passage has words of its own beside its controls.
func wordsBeside(n *html.Node) bool {
	var has bool
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if has {
			return
		}
		if n.Type == html.TextNode && strings.TrimSpace(n.Data) != "" {
			has = true
			return
		}
		if n.Type == html.ElementNode && (operable[n.Data] || skipped(n)) {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c)
	}
	return has
}

// passageLinks is every link inside n that is read, by its words and where
// it goes.
func passageLinks(n *html.Node) []PassageLink {
	var out []PassageLink
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if skipped(n) {
			return
		}
		if href, ok := htmltest.Attr(n, "href"); ok && n.Data == "a" {
			out = append(out, PassageLink{Words: readText(n), Href: href})
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c)
	}
	return out
}

// boxSaid is a checkbox or radio as text says it where it sits, [x] or
// [ ], and "" for anything else. The words beside a box were read alone:
// "Done Save" beside an empty box, and an agent reading a task's page took
// a task not done for one done.
func boxSaid(n *html.Node) string {
	if n.Data != "input" {
		return ""
	}
	if kind, _ := htmltest.Attr(n, "type"); kind != "checkbox" && kind != "radio" {
		return ""
	}
	if _, on := htmltest.Attr(n, "checked"); on {
		return "[x]"
	}
	return "[ ]"
}
