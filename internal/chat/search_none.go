package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Asked to tag "my garden notes", a small model searched for "garden",
// found nothing with the word, and told the person there were none: the
// notes were about tomatoes, compost and roses. So a search or a find of
// one kind that finds no words lists what there is of that kind, a few
// dozen titles, for the model to judge by what each is about.

// titlesMost is how many titles are listed when nothing had the words.
const titlesMost = 30

// noneOfKind is what there is of a kind when its words found nothing, or
// "" when there is nothing of it, or too much to list.
func (s *Service) noneOfKind(typeName string) string {
	t, ok := s.Store.Types().Get(typeName)
	if !ok || t.Internal {
		return ""
	}
	recs, err := s.Store.List(t.Name, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: titlesMost + 1})
	if err != nil || len(recs) == 0 || len(recs) > titlesMost {
		return ""
	}
	writers := s.Writers()
	var lines []string
	for _, rec := range recs {
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s", rec.ID, oneLine(recordTitle(s.Store, t, rec)), writers.Of(t.Name, rec).Words))
	}
	return fmt.Sprintf(" Nothing says it in words, but the person may mean some of these by what they are about: all %s (id, title, written by). Judge each by its title, and get_record one to read it; %s.\n<<<record text\n%s\nrecord text>>>",
		schema.Count(len(recs), t.Name), Untrusted, strings.Join(lines, "\n"))
}
