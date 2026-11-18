package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// wordsStart is how a record's own words start, its first text, on one
// line: enough to tell which notes are about the garden without reading
// each in full with get_record.
func wordsStart(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Fields {
		if f.Type != "markdown" && f.Type != "text" || f.Name == t.Title {
			continue
		}
		if v, _ := rec.Fields[f.Name].(string); strings.TrimSpace(v) != "" {
			return trim.Line(oneLine(v), 100)
		}
	}
	return ""
}
