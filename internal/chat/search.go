package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/search"
)

var searchTool = llm.Tool{
	Name:        "search",
	Description: "Find anything the person has by the words in it: every record of every content type and every block on the canvas, with where each is. Use it before saying something does not exist, and to find the id of a thing they mention.",
	Schema: map[string]any{"type": "object", "properties": map[string]any{
		"query": map[string]any{"type": "string", "description": "Words that must all appear."},
	}, "required": []string{"query"}, "additionalProperties": false},
}

// search is one search over everything, the same one the page and the API use.
func (s *Service) search(query string) toolResult {
	hits := search.Find(s.Store, s.Store.Types(), query)
	if len(hits) == 0 {
		return toolResult{text: fmt.Sprintf("nothing has %q in it", strings.TrimSpace(query))}
	}
	var lines []string
	for _, h := range hits {
		line := fmt.Sprintf("%s %s\t%s\t%s", h.Type, h.ID, h.Title, h.Href)
		if h.Snippet != "" {
			line += "\t" + h.Snippet
		}
		lines = append(lines, line)
	}
	return toolResult{text: fmt.Sprintf("%d found (type id, title, page, words around the match):\n%s", len(hits), strings.Join(lines, "\n"))}
}
