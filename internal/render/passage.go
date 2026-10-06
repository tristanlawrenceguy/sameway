package render

import (
	"html/template"
	"regexp"
	"strings"
)

// A reply's lines that start "- " or "1. " are a list, and are one: drawn
// with their line breaks they looked like a list, but a screen reader and
// an agent reading the page were given one paragraph with the items run
// together. Lines before the list stay a paragraph.

var (
	// A model told to write plain text still writes **this**: it is drawn
	// bold, not shown with its stars.
	strongWords  = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	bulletLine   = regexp.MustCompile(`^[-*•]\s+`)
	numberedLine = regexp.MustCompile(`^\d+[.)]\s+`)
)

// passage is one paragraph of a message as HTML: a p, or a p and a list.
func passage(s string, name func(path string) string) template.HTML {
	lines := strings.Split(s, "\n")
	first := len(lines)
	for i := range lines {
		if listLine(lines[i]) {
			first = i
			break
		}
	}
	rest := lines[first:]
	for _, l := range rest {
		if strings.TrimSpace(l) != "" && !listLine(l) {
			first, rest = len(lines), nil // not a list after all
			break
		}
	}
	var b strings.Builder
	if lead := strings.TrimSpace(strings.Join(lines[:first], "\n")); lead != "" {
		b.WriteString("<p>" + said(lead, name) + "</p>")
	}
	if len(rest) > 0 {
		tag := "ul"
		if numberedLine.MatchString(strings.TrimSpace(rest[0])) {
			tag = "ol"
		}
		b.WriteString("<" + tag + ">")
		for _, l := range rest {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			l = numberedLine.ReplaceAllString(bulletLine.ReplaceAllString(l, ""), "")
			b.WriteString("<li>" + said(l, name) + "</li>")
		}
		b.WriteString("</" + tag + ">")
	}
	return template.HTML(b.String())
}

func listLine(l string) bool {
	l = strings.TrimSpace(l)
	return bulletLine.MatchString(l) || numberedLine.MatchString(l)
}

// said is a line's words as HTML: its links, and what it made bold.
func said(s string, name func(path string) string) string {
	return strongWords.ReplaceAllString(string(linkifyNamed(s, name)), "<strong>$1</strong>")
}
