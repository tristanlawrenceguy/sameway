package server_test

import (
	"html"
	"regexp"
	"strings"
)

// eventHeading is how an activity entry's sentence opens: the entry's h3,
// which is the sentence itself, said once.
const eventHeading = `<h3 class="sw-event__text sw-event__heading"`

var (
	scriptOrStyle = regexp.MustCompile(`(?is)<(script|style)\b.*?</(script|style)>`)
	anyTag        = regexp.MustCompile(`(?s)<[^>]*>`)
	h3Element     = regexp.MustCompile(`(?s)<h3\b[^>]*>(.*?)</h3>`)
	spaces        = regexp.MustCompile(`\s+`)
)

// said is the text a page says, as a person reads it: no tags, entities
// read as their characters, and whitespace collapsed to single spaces.
func said(page string) string {
	page = scriptOrStyle.ReplaceAllString(page, " ")
	page = anyTag.ReplaceAllString(page, "")
	page = html.UnescapeString(page)
	return strings.TrimSpace(spaces.ReplaceAllString(page, " "))
}

// saidTimes counts how often a sentence is said on a page.
func saidTimes(page, sentence string) int {
	return strings.Count(said(page), sentence)
}

// h3Said is what each h3 on a page says, in order.
func h3Said(page string) []string {
	var out []string
	for _, m := range h3Element.FindAllStringSubmatch(page, -1) {
		out = append(out, said(m[1]))
	}
	return out
}

// anyH3Says reports whether some h3 on the page says the sentence.
func anyH3Says(page, sentence string) bool {
	for _, t := range h3Said(page) {
		if strings.Contains(t, sentence) {
			return true
		}
	}
	return false
}
