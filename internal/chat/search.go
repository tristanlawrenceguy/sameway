package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// searchTool names the kinds the search can be narrowed to as they are
// now, so a kind added while the workspace runs is offered at once.
func (s *Service) searchTool() llm.Tool {
	kinds := strings.Join(search.Kinds(s.Store.Types()), ", ")
	return llm.Tool{
		Name: "search",
		Description: "Find anything the person has by the words in it: every record of every content type and every block on the canvas, with where each is. Use it before saying something does not exist, and to find the id of a thing they mention. " +
			"Search everything first: the answer begins with how many were found of each kind, 12 found: 7 notes, 3 tasks, 2 blocks. Then, if the counts show where it is, search again with type to see only that kind. " +
			fmt.Sprintf("At most %d come back at a time; the answer says when there are more, and page asks for them.", search.Limit),
		Schema: map[string]any{"type": "object", "properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Words that must all appear."},
			"type":  map[string]any{"type": "string", "description": "Only this kind of thing, one of: " + kinds + ". Leave it out to search everything."},
			"page":  map[string]any{"type": "integer", "minimum": 1, "description": "Which page of results, from 1; only when the answer says there are more."},
		}, "required": []string{"query"}, "additionalProperties": false},
	}
}

// search is one search over everything, the same one the page and the API
// use: counted by kind, narrowed when a type is named, a page at a time.
func (s *Service) search(query, only string, page int) toolResult {
	types := s.Store.Types()
	only = strings.TrimSpace(only)
	if _, ok := search.Searchable(types, only); only != "" && !ok {
		return fail("%s", search.Refusal(only, search.Kinds(types)))
	}
	hits, some := search.Matches(s.Store, types, query)
	res := search.Narrow(hits, query, only, page)
	res.Some = some
	if res.Total == 0 {
		return toolResult{text: fmt.Sprintf("nothing has %q in it", res.Query)}
	}
	if len(res.Hits) == 0 {
		return toolResult{text: res.Said() + ". Leave out type to see them."}
	}
	writers := s.Writers()
	var lines []string
	for _, h := range res.Hits {
		line := fmt.Sprintf("%s %s\t%s\t%s\t%s", h.Type, h.ID, oneLine(h.Title), h.Href, writers.OfID(h.Type, h.ID).Words)
		if h.Snippet != "" {
			line += "\t" + oneLine(h.Snippet)
		}
		lines = append(lines, line)
	}
	// Titles and words are fenced, each line saying who wrote it; see records/provenance.go.
	return toolResult{text: fmt.Sprintf("%s (type id, title, page, written by, words around the match). Each line's words were written by the one on it; %s.\n<<<record text\n%s\nrecord text>>>", res.Said(), Untrusted, strings.Join(lines, "\n"))}
}

// oneLine keeps a record's words to the one line a listing gives them,
// so they cannot start a line of their own.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
