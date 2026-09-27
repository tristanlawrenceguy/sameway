package server

import "strings"

// Shortening words for places that have one line. A title is shortened by
// trim.Title, the one rule the server and the chat share.

// trimLabel cuts a label to at most three words.
func trimLabel(s string) string {
	fields := strings.Fields(s)
	if len(fields) <= 3 {
		return s
	}
	return strings.Join(fields[:3], " ")
}
