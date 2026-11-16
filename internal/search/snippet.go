package search

import "strings"

// snippet is the words around the first match, so the hit says why it is
// one, in roughly a line.
func snippet(body string, words []string) string {
	body = strings.Join(strings.Fields(body), " ")
	if body == "" {
		return ""
	}
	// Found in the folded text, cut from the text as it is: folding can
	// change a letter's length, so the places are mapped back, never reused.
	at := -1
	if spans := Spans(body, words); len(spans) > 0 {
		at = spans[0][0]
	}
	start := 0
	if at > 40 {
		start = at - 40
		for start > 0 && body[start] != ' ' {
			start--
		}
	}
	end := len(body)
	if end > start+120 {
		end = start + 120
		for end < len(body) && body[end] != ' ' {
			end++
		}
	}
	out := body[start:end]
	if start > 0 {
		out = "…" + strings.TrimLeft(out, " ")
	}
	if end < len(body) {
		out += "…"
	}
	return out
}
