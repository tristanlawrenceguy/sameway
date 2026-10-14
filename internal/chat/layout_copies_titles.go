package chat

import (
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Records copied by their titles alone. A model asked to "put my three
// tasks for today on the page" wrote a text block of the three titles, one
// to a line, with no links: the link check (layout_copies.go) saw nothing,
// and the words stayed when the tasks were done. A line that is a record's
// title counts as a copy of it, as a link to it does.

// titledMost is how many records a type may hold for its titles to be
// read; a workspace of more is read by its links alone.
const titledMost = 5000

// recordRef is one record by its type and id.
type recordRef struct{ typ, id string }

// titleIndex is each record's title, lowercased, to the record; built only
// when some block has three lines or more that could be titles.
func (s *Service) titleIndex(recs []*store.Record) map[string]recordRef {
	if !anyLines(recs) {
		return nil
	}
	out := map[string]recordRef{}
	for _, t := range records.ContentTypes(s.Store) {
		if n, err := s.Store.Count(t.Name); err != nil || n > titledMost {
			continue
		}
		list, _ := s.Store.List(t.Name, store.ListOptions{})
		for _, r := range list {
			key := strings.ToLower(strings.TrimSpace(t.Called(r.ID, r.Fields)))
			if _, taken := out[key]; !taken && key != "" {
				out[key] = recordRef{t.Name, r.ID}
			}
		}
	}
	return out
}

// anyLines says whether a block that is not a record's own has at least
// copiedMost lines in its words.
func anyLines(recs []*store.Record) bool {
	for _, b := range recs {
		if b.Fields["component"] == "collection" || b.Fields["component"] == "record" {
			continue
		}
		if len(linesIn(b.Fields["props"])) >= copiedMost {
			return true
		}
	}
	return false
}

// linesIn is every line of every string in a block's props.
func linesIn(props any) []string {
	var out []string
	for _, s := range wordsIn(props) {
		out = append(out, strings.Split(s, "\n")...)
	}
	return out
}

// listMark is what starts a list item or a checkbox line.
var listMark = regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)])?\s*(?:\[[ xX]\]\s*)?`)

// markdownLink is [words](address).
var markdownLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)

// lineTitle is a line as the title it may be: no list mark, a link's words
// for the link, nothing after a dash and a page address; lowercased.
func lineTitle(line string) string {
	line = markdownLink.ReplaceAllString(listMark.ReplaceAllString(line, ""), "$1")
	if i := strings.Index(line, " /t/"); i > 0 {
		line = strings.TrimRight(line[:i], " —-–:")
	}
	return strings.ToLower(strings.TrimSpace(line))
}
