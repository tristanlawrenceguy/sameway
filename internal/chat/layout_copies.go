package chat

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A block that copies records as words. Asked for "a summary list of my
// draft notes", an agent wrote a list block of five note titles, each with
// its link: it read as done, and it stays as it is when a note is renamed,
// finished or deleted, so the page goes on saying what is no longer so. A
// collection shows the same notes and stays current. The rule (never copy a
// record's words into a card) was in the prompt; the copy was not said
// anywhere the model looks after it writes.

// recordLink is a link to one record's page, /t/note/<id>.
var recordLink = regexp.MustCompile(`/t/([a-z_]+)/([a-z0-9]{16})\b`)

// copiedMost is how many records of one type a block links to before it
// reads as a copy of them rather than a pointer or two in its words.
const copiedMost = 3

// copiesProblems are Layout now's lines for blocks that copy records.
func copiesProblems(recs []*store.Record) []string {
	var out []string
	for _, b := range recs {
		if b.Fields["component"] == "collection" || b.Fields["component"] == "record" {
			continue
		}
		byType := map[string]map[string]bool{}
		for _, s := range wordsIn(b.Fields["props"]) {
			for _, m := range recordLink.FindAllStringSubmatch(s, -1) {
				if byType[m[1]] == nil {
					byType[m[1]] = map[string]bool{}
				}
				byType[m[1]][m[2]] = true
			}
		}
		var types []string
		for t, ids := range byType {
			if len(ids) >= copiedMost {
				types = append(types, t)
			}
		}
		sort.Strings(types)
		for _, t := range types {
			out = append(out, fmt.Sprintf("block %s (%s) copies %d %s as words, which stay as they are when those change, are done or go: a collection block of %s shows them and stays current; make it one with update_component, unless the person asked for a list that does not change", b.ID, b.Fields["component"], len(byType[t]), schema.Plural(t), t))
		}
	}
	return out
}

// wordsIn is every string in a block's props, however deep.
func wordsIn(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, x := range v {
			out = append(out, wordsIn(x)...)
		}
		return out
	case map[string]any:
		var out []string
		for _, x := range v {
			out = append(out, wordsIn(x)...)
		}
		return out
	}
	return nil
}
