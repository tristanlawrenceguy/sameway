package automate

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// recordWords is a record as the model reads it: its name, then each of
// its words, clipped.
func recordWords(st *store.Store, t *schema.Type, rec *store.Record) string {
	var b strings.Builder
	b.WriteString(records.Name(st, t, rec))
	for _, f := range t.Fields {
		if f.Name == "tags" || f.Name == t.Title || f.ReadOnly { // what Sameway keeps is not what the record says
			continue
		}
		if v, ok := rec.Fields[f.Name].(string); ok && strings.TrimSpace(v) != "" && (f.Type == "string" || f.Type == "text" || f.Type == "markdown" || f.Type == "datetime") {
			b.WriteString("\n" + schema.Words(f.Name) + ": " + v)
		}
	}
	return clipEnds(b.String(), 4000)
}

// clipEnds is s cut to about n runes, keeping its start and its end: a
// conversation brought from ChatGPT says what was decided last ("book the
// train for Friday"), and the start alone left it out.
func clipEnds(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	half := n / 2
	return string(r[:half]) + "\n…\n" + string(r[len(r)-half:])
}

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
