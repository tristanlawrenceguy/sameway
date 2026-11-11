package automate

import (
	"fmt"
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
	b.WriteString(linkedText(st, rec))
	return clipEnds(b.String(), 4000)
}

// linkedText is the words of the files a record links to: a bill shared
// as a PDF is a note with only its link, and its amount and its day are
// in the file (pictures.go has a photo's).
func linkedText(st *store.Store, rec *store.Record) string {
	var b strings.Builder
	seen := map[string]bool{}
	for _, v := range rec.Fields {
		str, _ := v.(string)
		for _, m := range fileLink.FindAllStringSubmatch(str, -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			if f, err := st.Get(records.FileType, m[1]); err == nil {
				if text, _ := f.Fields["text"].(string); strings.TrimSpace(text) != "" {
					b.WriteString("\n\nThe file " + fmt.Sprint(f.Fields["title"]) + " it links to says:\n" + text)
				}
			}
		}
	}
	return b.String()
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
