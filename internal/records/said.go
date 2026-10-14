package records

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// What someone said, as the log says it. A message holding JSON was cut to
// 80 characters first and cleaned after, by a pattern that needed its
// closing brace: cut, `{"type": "task", "done":…` kept its JSON while a
// message whose braces closed in time said [content omitted]; nested
// braces were missed too; and the API's detail kept the whole raw text
// (crew findings 0749, 0750). Now the JSON is taken out first, however
// deep and wherever it stops, and only then is the line cut.

const omitted = "[content omitted]"

// SaidWords is text as one line of the log says it: JSON in it replaced,
// then the first line cut to n characters.
func SaidWords(text string, n int) string {
	return trim.Line(stripJSONLike(text), n)
}

// stripJSONLike replaces each JSON object or array in s, one that opens
// on a quote, a brace or a bracket, with [content omitted]: to its
// matching close, or to the end when it was cut before closing.
func stripJSONLike(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if (c == '{' || c == '[') && opensJSON(s[i+1:]) {
			b.WriteString(omitted)
			i = closing(s, i)
			continue
		}
		b.WriteByte(c)
		i++
	}
	return strings.ReplaceAll(b.String(), omitted+" "+omitted, omitted)
}

// opensJSON says what follows a brace starts JSON's insides: a quoted key
// or string, or another object or array.
func opensJSON(rest string) bool {
	rest = strings.TrimLeft(rest, " \t\r\n")
	return rest != "" && (rest[0] == '"' || rest[0] == '{' || rest[0] == '[')
}

// closing is where the JSON opening at i ends: just past its matching
// close, quotes respected, or the end of s.
func closing(s string, i int) int {
	depth, quoted := 0, false
	for j := i; j < len(s); j++ {
		switch c := s[j]; {
		case quoted && c == '\\':
			j++
		case c == '"':
			quoted = !quoted
		case quoted:
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
			if depth == 0 {
				return j + 1
			}
		}
	}
	return len(s)
}
