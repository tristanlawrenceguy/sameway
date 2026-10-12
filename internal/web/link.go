package web

// Link is a link in running text, to words already safe as HTML: a step
// of a story that leads to a service's own page, say. One to another site
// opens in a new tab (prose.Outward).
func Link(href, words string) string {
	return `<a class="sw-link" href="` + href + `">` + words + `</a>`
}
