package render

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	internalPathRe = regexp.MustCompile(`/t/[a-z]+/[a-zA-Z0-9][a-zA-Z0-9_-]*`)
	externalURLRe  = regexp.MustCompile(`https?://[^\s<>"']+[a-zA-Z0-9]`)
)

// linkMatch holds a single URL or path match with its position in the source.
type linkMatch struct {
	start, end int
	raw        string
}

// validExternalURL checks that an external URL has a safe http(s) scheme.
func validExternalURL(s string) bool {
	parsed, err := url.Parse(s)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

// linkify converts recognized URLs and internal paths in a paragraph into
// clickable anchor tags. Returns template.HTML so the result is not escaped.
func linkify(s string) template.HTML {
	// Collect all matches from both regexes, then sort by position.
	var matches []linkMatch
	for _, sub := range []*regexp.Regexp{internalPathRe, externalURLRe} {
		for _, idx := range sub.FindAllStringIndex(s, -1) {
			start, end := idx[0], idx[1]
			raw := s[start:end]
			if sub == externalURLRe && !validExternalURL(raw) {
				continue // reject javascript: and other unsafe schemes
			}
			matches = append(matches, linkMatch{start, end, raw})
		}
	}

	// Sort by start position (stable — preserves regex order for same-start).
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].start < matches[j].start })

	// Deduplicate overlapping matches: keep the outermost one.
	var kept []linkMatch
	for i := 0; i < len(matches); i++ {
		if i > 0 && matches[i].end <= matches[i-1].end {
			continue // contained in previous outer match, skip
		}
		// Remove any earlier kept match that this one fully contains.
		for len(kept) > 0 && matches[i].start <= kept[len(kept)-1].start && matches[i].end >= kept[len(kept)-1].end {
			kept = kept[:len(kept)-1]
		}
		kept = append(kept, matches[i])
	}

	if len(kept) == 0 {
		return template.HTML(html.EscapeString(s))
	}

	var buf strings.Builder
	lastEnd := 0
	for _, m := range kept {
		if m.start > lastEnd {
			buf.WriteString(html.EscapeString(s[lastEnd:m.start]))
		}
		fmt.Fprintf(&buf, `<a class="sw-link" href="%s">%s</a>`, m.raw, m.raw)
		lastEnd = m.end
	}
	if lastEnd < len(s) {
		buf.WriteString(html.EscapeString(s[lastEnd:]))
	}

	return template.HTML(buf.String())
}
