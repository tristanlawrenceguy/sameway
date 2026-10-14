package server_test

import (
	"regexp"
	"strings"
)

// More of what the crawl (machine_words_test.go) fails on, each a class
// the crew had fixed one instance at a time: names with commas joined by
// commas (0748), a download's size too small to decide on (0711), the
// schema's own words (JSON Schema keywords the model read and repeated),
// and a word as code writes it (snake_case) that no field of this
// workspace explains.
var (
	tinySize      = regexp.MustCompile(`\b\d+ bytes\b|\((?:CSV|XLSX|vCard|ICS|Markdown|HTML|DOCX|PDF|ZIP|SRT|TXT), (?:about )?(?:\d+ KB|under 1 KB)\)`)
	schemaKeyword = regexp.MustCompile(`\b(?:additionalProperties|minLength|maxLength|minItems|maxItems|anyOf|oneOf|allOf|enum)\b|\$ref|\$schema|"type"\s*:`)
	codeWord      = regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)+\b`)
)

// plainWords is each such word in one run of text, as [kind, word].
func (m machineSeen) plainWords(text string) [][2]string {
	var out [][2]string
	for a, ac := range m.names {
		for b, bc := range m.names {
			if (ac || bc) && strings.Contains(text, a+", "+b) {
				out = append(out, [2]string{"names with commas joined by commas", a + ", " + b})
			}
		}
	}
	for _, w := range tinySize.FindAllString(text, -1) {
		out = append(out, [2]string{"a size too small to decide on", w})
	}
	for _, w := range schemaKeyword.FindAllString(text, -1) {
		out = append(out, [2]string{"a schema's words", w})
	}
	for _, w := range codeWord.FindAllString(text, -1) {
		if !m.fields[w] { // a field's name is said as one above
			out = append(out, [2]string{"a word as code writes it", w})
		}
	}
	return out
}
