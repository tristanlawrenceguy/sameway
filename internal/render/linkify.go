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
	// markdownLinkRe is a link written the Markdown way, [Seeds to buy](/t/note/abc):
	// the words in brackets are the link's words, the address its target.
	markdownLinkRe = regexp.MustCompile(`\[([^\[\]\n]+)\]\((/t/[a-z]+(?:/[a-zA-Z0-9][a-zA-Z0-9_-]*)?(?:\?[^\s()<>"']*)?|https?://[^\s()<>"']+)\)`)
)

// linkMatch holds a single URL or path match with its position in the source.
type linkMatch struct {
	start, end int
	raw        string
	words      string // the words a Markdown link gave it, if any
}

// validExternalURL checks that an external URL has a safe http(s) scheme.
func validExternalURL(s string) bool {
	parsed, err := url.Parse(s)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

// linkify converts recognized URLs and internal paths in a paragraph into
// clickable anchor tags. Returns template.HTML so the result is not escaped.
func linkify(s string) template.HTML { return linkifyNamed(s, nil) }

// linkifyNamed is linkify with a way to name a page: a link to a record
// reads as the record's name, "Water the plants", not as its address,
// which a screen reader spells out and nobody can remember. A page it
// cannot name keeps its address as its words.
func linkifyNamed(s string, name func(path string) string) template.HTML {
	var buf strings.Builder
	lastEnd := 0
	for _, m := range findLinks(s, name) {
		buf.WriteString(html.EscapeString(s[lastEnd:m.start]))
		fmt.Fprintf(&buf, `<a class="sw-link" href="%s">%s</a>`, html.EscapeString(m.raw), html.EscapeString(m.words))
		lastEnd = m.end
	}
	buf.WriteString(html.EscapeString(s[lastEnd:]))
	return template.HTML(buf.String())
}

// LinkWords is text with its links as the words they show, for where words
// are read out rather than drawn, such as the status that says a reply
// arrived: [Seeds to buy](/t/note/abc) and a bare /t/note/abc to that note
// are both Seeds to buy.
func LinkWords(s string, name func(path string) string) string {
	var buf strings.Builder
	lastEnd := 0
	for _, m := range findLinks(s, name) {
		buf.WriteString(s[lastEnd:m.start])
		buf.WriteString(m.words)
		lastEnd = m.end
	}
	buf.WriteString(s[lastEnd:])
	return buf.String()
}

// findLinks are the links in text, in order and not overlapping, each with
// the words it shows. A link the reply named keeps its name; a bare address
// to a record is named by the record; only what cannot be named shows where
// it goes, and an address reads as where it goes, not as its scheme: a
// screen reader would spell out "h t t p s colon slash slash".
func findLinks(s string, name func(path string) string) []linkMatch {
	// Collect all matches, then sort by position. A Markdown link comes
	// first, so the address inside it is taken as part of it.
	var matches []linkMatch
	for _, idx := range markdownLinkRe.FindAllStringSubmatchIndex(s, -1) {
		raw := s[idx[4]:idx[5]]
		if !strings.HasPrefix(raw, "/") && !validExternalURL(raw) {
			continue
		}
		matches = append(matches, linkMatch{idx[0], idx[1], raw, strings.TrimSpace(s[idx[2]:idx[3]])})
	}
	for _, sub := range []*regexp.Regexp{internalPathRe, externalURLRe} {
		for _, idx := range sub.FindAllStringIndex(s, -1) {
			start, end := idx[0], idx[1]
			raw := s[start:end]
			if sub == externalURLRe && !validExternalURL(raw) {
				continue // reject javascript: and other unsafe schemes
			}
			matches = append(matches, linkMatch{start, end, raw, ""})
		}
	}

	// Sort by start position (stable — preserves regex order for same-start).
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].start < matches[j].start })

	// Deduplicate overlapping matches: keep the outermost one.
	var kept []linkMatch
	for _, m := range matches {
		if n := len(kept); n > 0 && m.start < kept[n-1].end && m.end <= kept[n-1].end {
			continue // inside the match before it, such as the address of a Markdown link
		}
		// Remove any earlier kept match that this one fully contains.
		for len(kept) > 0 && m.start <= kept[len(kept)-1].start && m.end >= kept[len(kept)-1].end {
			kept = kept[:len(kept)-1]
		}
		kept = append(kept, m)
	}

	for i, m := range kept {
		words := readable(m.raw)
		if m.words != "" && !isAddress(m.words) {
			words = m.words
		} else if name != nil && strings.HasPrefix(m.raw, "/t/") {
			if n := strings.TrimSpace(name(m.raw)); n != "" {
				words = n
			}
		}
		kept[i].words = words
	}
	return kept
}

// readable is an address as a person says it: example.com/guide.
func readable(raw string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	s = strings.TrimPrefix(s, "www.")
	if t := strings.TrimSuffix(s, "/"); t != "" {
		s = t
	}
	return s
}

// isAddress says whether a link's words are only an address again, as in
// [/t/note/abc](/t/note/abc): words like that are no name, so the link is
// named as a bare address would be.
func isAddress(words string) bool {
	return internalPathRe.FindString(words) == words || externalURLRe.FindString(words) == words
}
