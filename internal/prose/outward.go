package prose

import (
	"regexp"
	"strings"
)

// A link to another site opened in Sameway's own tab, and the person had
// to find the browser's Back button to return (the key page on
// console.anthropic.com, met in the first minute). So a link to anywhere
// else opens in a new tab, and says so to a screen reader. A link inside
// Sameway is a path, never an address, and stays where it is.

var outwardLink = regexp.MustCompile(`(?s)<a ([^>]*?)href="(https?://[^"]*)"([^>]*)>(.*?)</a>`)

// NewTab is the note every link to another site carries.
const NewTab = `<span class="sw-visually-hidden"> (opens in a new tab)</span>`

// Outward makes each link to another site in html open in a new tab; one
// that already says where it opens is left as it is.
func Outward(html string) string {
	if !strings.Contains(html, `href="http`) {
		return html
	}
	return outwardLink.ReplaceAllStringFunc(html, func(a string) string {
		m := outwardLink.FindStringSubmatch(a)
		if strings.Contains(m[1]+m[3], "target=") {
			return a
		}
		return `<a ` + m[1] + `href="` + m[2] + `"` + m[3] + ` target="_blank" rel="noopener">` + m[4] + NewTab + `</a>`
	})
}
