package server_test

import (
	"regexp"
	"strings"
)

// eventHeading is how an activity entry's sentence opens: the entry's h3,
// which is the sentence itself, said once.
const eventHeading = `<h3 class="sw-event__text sw-event__heading"`

var (
	h3Element = regexp.MustCompile(`(?s)<h3\b[^>]*>(.*?)</h3>`)
)

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
