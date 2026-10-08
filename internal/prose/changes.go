package prose

import (
	"html"
	"html/template"
	"regexp"
	"strings"
)

// A change suggested to someone's writing is shown as writing: the words
// formatted as they read, never the Markdown behind them, with the passage
// that changes highlighted; and a change to the formatting alone is said
// in words, "Make it bold", rather than as asterisks moving about.

var tags = regexp.MustCompile(`<[^>]*>`)

// Plain is what Markdown says with its formatting taken away.
func Plain(md string) string {
	out := strings.ReplaceAll(string(Render(md, 2)), NewTab, "") // a link's note is not its words
	return strings.Join(strings.Fields(html.UnescapeString(tags.ReplaceAllString(out, " "))), " ")
}

// Marked renders Markdown with the one passage given highlighted, with
// words for a screen reader where it starts and where it ends. It is a
// plain span, not mark: screen readers say mark differently or not at
// all, and some say its end and some do not, so the words are heard the
// same everywhere, once. Where the passage cannot be marked as one
// stretch of a paragraph, the text is rendered unmarked.
func Marked(md, passage, start, end string, base int) template.HTML {
	at := strings.Index(md, passage)
	if passage == "" || at < 0 {
		return Render(md, base)
	}
	from, to := at, at+len(passage)
	// A mark before a heading's # or a list's - would stop it being one.
	if from == 0 || md[from-1] == '\n' {
		if m := blockMarks.FindString(md[from:to]); m != "" {
			from += len(m)
		}
	}
	const open, shut = "", ""
	out := string(Render(md[:from]+open+md[from:to]+shut+md[to:], base))
	i, j := strings.Index(out, open), strings.Index(out, shut)
	if i < 0 || j < i || blockTag.MatchString(out[i:j]) {
		return template.HTML(strings.NewReplacer(open, "", shut, "").Replace(out))
	}
	mark, unmark := `<span class="sw-prose__changed">`, `</span>`
	if start != "" {
		mark += `<span class="sw-visually-hidden">` + template.HTMLEscapeString(start) + `: </span>`
	}
	if end != "" {
		unmark = `<span class="sw-visually-hidden">, ` + template.HTMLEscapeString(end) + `,</span>` + unmark
	}
	return template.HTML(out[:i] + mark + out[i+len(open):j] + unmark + out[j+len(shut):])
}

var (
	blockMarks = regexp.MustCompile(`^(?:#{1,6} |[-*+] |\d+[.)] |> )+`)
	blockTag   = regexp.MustCompile(`</?(?:p|h[1-6]|li|ul|ol|blockquote|pre|table|div)\b`)
)

// FormatChange says in words what changes between two passages whose
// words are the same, and whether that is all that changes: "Make it
// bold", "Make it a heading". Different words are not a formatting change.
func FormatChange(from, to string) (string, bool) {
	if Plain(from) != Plain(to) || from == to {
		return "", false
	}
	was, now := kinds(from), kinds(to)
	var said []string
	for _, k := range formats {
		switch {
		case now[k.tag] && !was[k.tag]:
			said = append(said, k.on)
		case was[k.tag] && !now[k.tag]:
			said = append(said, k.off)
		}
	}
	if len(said) == 0 {
		return "Change the formatting", true
	}
	words := strings.Join(said, ", and ")
	return strings.ToUpper(words[:1]) + words[1:], true
}

var formats = []struct{ tag, on, off string }{
	{"h", "make it a heading", "make it ordinary text instead of a heading"},
	{"ol", "make it a numbered list", "make it ordinary text instead of a numbered list"},
	{"ul", "make it a list", "make it ordinary text instead of a list"},
	{"blockquote", "make it a quote", "make it ordinary text instead of a quote"},
	{"strong", "make it bold", "take the bold off"},
	{"em", "make it italic", "take the italics off"},
	{"a", "add a link", "take the link off"},
	{"code", "show it as code", "show it as ordinary text"},
	{"del", "strike it through", "take the strike-through off"},
}

var tagName = regexp.MustCompile(`<(h[1-6]|ol|ul|blockquote|strong|em|a|code|del)\b`)

func kinds(md string) map[string]bool {
	out := map[string]bool{}
	for _, m := range tagName.FindAllStringSubmatch(string(Render(md, 2)), -1) {
		k := m[1]
		if k[0] == 'h' && len(k) == 2 {
			k = "h"
		}
		out[k] = true
	}
	return out
}
