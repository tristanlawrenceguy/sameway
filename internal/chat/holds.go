package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// holdsAll says whether a record holds every word of a query, in its
// title or any of its text, matched as search matches them (case, accents
// and plurals alike): a person asks after the boiler engineer, and the
// note says the engineer comes to fix the boiler.
func holdsAll(title string, rec *store.Record, query string) bool {
	text := []string{title}
	for _, v := range rec.Fields {
		if s, ok := v.(string); ok {
			text = append(text, s)
		}
	}
	all := strings.Join(text, " ")
	for _, w := range search.Words(query) {
		if len(search.Spans(all, []string{w})) == 0 {
			return false
		}
	}
	return true
}
