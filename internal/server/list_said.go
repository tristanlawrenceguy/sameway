package server

import "github.com/tristanlawrenceguy/sameway/internal/schema"

// listSaid is a narrowed or ordered list's line, in the words a collection
// block says what it shows: "12 tasks, not done, by due, earliest first".
// It was "%d matching %s%s", and with no condition read "37 matching , at
// largest or newest first" (crew finding 0751).
func listSaid(t *schema.Type, n int, where, order string) string {
	out := schema.Count(n, t.Name)
	if where != "" {
		out += ", " + where
	}
	return out + orderWords(t, order)
}
