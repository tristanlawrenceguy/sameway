package web

import "html/template"

// Small pieces of HTML said inside a sentence, the same on every page.

// Link is a link in running text, to words already safe as HTML: a step
// of a story that leads to a service's own page, say. One to another site
// opens in a new tab (prose.Outward).
func Link(href, words string) string {
	return `<a class="sw-link" href="` + href + `">` + words + `</a>`
}

// Time puts what a day says in a <time> holding its value, with the date
// in full as its title when the words leave it out (Today). A screen
// reader reads the words, which say it on their own; the value is for a
// machine, the title for a pointer.
func Time(class, value, full string, inner template.HTML) string {
	b := "<time"
	if class != "" {
		b += ` class="` + class + `"`
	}
	if value != "" {
		b += ` datetime="` + template.HTMLEscapeString(value) + `"`
	}
	if full != "" {
		b += ` title="` + template.HTMLEscapeString(full) + `"`
	}
	return b + ">" + string(inner) + "</time>"
}
