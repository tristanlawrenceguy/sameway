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
func (s *Service) copiesProblems(recs []*store.Record) []string {
	titles := s.titleIndex(recs) // layout_copies_titles.go
	var out []string
	for _, b := range recs {
		for _, said := range copiesSaid(b, copiedBy(b, titles), true) {
			out = append(out, fmt.Sprintf("block %s (%s) %s", b.ID, placedOf(b).called(), said))
		}
	}
	return out
}

// copiedBy is how many different records of each type a block names in its
// own words, by their links or titles; nothing for a block that shows
// records itself.
func copiedBy(b *store.Record, titles map[string]recordRef) map[string]int {
	if b.Fields["component"] == "collection" || b.Fields["component"] == "record" {
		return nil
	}
	byType := map[string]map[string]bool{}
	add := func(typ, id string) {
		if byType[typ] == nil {
			byType[typ] = map[string]bool{}
		}
		byType[typ][id] = true
	}
	for _, w := range wordsIn(b.Fields["props"]) {
		for _, m := range recordLink.FindAllStringSubmatch(w, -1) {
			add(m[1], m[2])
		}
	}
	for _, line := range linesIn(b.Fields["props"]) {
		if ref, ok := titles[lineTitle(line)]; ok {
			add(ref.typ, ref.id)
		}
	}
	out := map[string]int{}
	for t, ids := range byType {
		out[t] = len(ids)
	}
	return out
}

// copiesSaid is what a block copying records is told, consequence first.
// Told it "copies 95 notes as words", an agent that had sent 106 titles
// read the number as items lost and filed it as a bug: the count is of
// different notes named, and what matters is that the words go stale.
func copiesSaid(b *store.Record, by map[string]int, short bool) []string {
	var types []string
	for t, n := range by {
		if n >= copiedMost {
			types = append(types, t)
		}
	}
	sort.Strings(types)
	var out []string
	for _, t := range types {
		if short { // Layout now's list; the change's own note says it whole
			out = append(out, fmt.Sprintf("will go stale: it names %d different %s by their titles or links as its own words; a collection block of %s stays current", by[t], schema.Plural(t), t))
			continue
		}
		out = append(out, fmt.Sprintf("will go stale: it names %d different %s by their titles or links, written into it as its own words, which stay as they are when those %s change, are done or go. A collection block of %s shows them and stays current (with where, it shows just the ones meant, as status=draft); use that instead with update_component, unless the person asked for a list that does not change",
			by[t], schema.Plural(t), schema.Plural(t), t))
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
