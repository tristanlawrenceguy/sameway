package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// cleanActivityFields resolves raw schema identifiers in type-setting activity
// entries. When target is "type", detail holds the raw identifier (e.g.,
// "test_type") and summary may contain it too ("System added type test_type").
// Both are replaced with their display-form names ("Test Type") so agents
// reading the API list see human-readable values, not internal IDs.
func cleanActivityFields(fields map[string]any) {
	target, _ := fields["target"].(string)
	if target != "type" {
		return // nothing to clean for non-type-setting entries
	}
	detail, _ := fields["detail"].(string)
	summary, _ := fields["summary"].(string)

	cleanedDetail := schema.DisplayName(detail)
	fields["detail"] = cleanedDetail

	if strings.Contains(summary, " "+detail) {
		fields["summary"] = strings.Replace(summary, " "+detail, " "+cleanedDetail, 1)
	} else if strings.HasSuffix(summary, detail) && summary != cleanedDetail {
		fields["summary"] = summary[:len(summary)-len(detail)] + cleanedDetail
	}
}

// entriesApart tells apart log entries shown together that say the same,
// "created task Call plumber" twice, whose links and Undo buttons would
// share a name (apart.go): the way the record's own list tells it apart
// while it exists, then by their time, then by which was added first.
func (s *Server) entriesApart(recs []*store.Record) []string {
	names := make([]string, len(recs))
	for i, r := range recs {
		w := chat.Say(s.app.Store, r.Fields)
		names[i] = strings.Join([]string{w.Action, w.Target, w.Detail}, " ")
	}
	return apart(names, func(i int) []string {
		r := recs[i]
		ways := make([]string, 4)
		target, _ := r.Fields["target"].(string)
		id, _ := r.Fields["target_id"].(string)
		if t, ok := s.app.Types.Get(target); ok && id != "" {
			if rec, err := s.app.Store.Get(t.Name, id); err == nil {
				// Said as its row is, number and all: a time here once
				// named two notes made a second apart differently from
				// their list.
				return recordWays(t, rec)
			}
		}
		return append(ways, "at "+momentWords(r.CreatedAt), "at "+secondWords(r.CreatedAt))
	}, func(i int) string {
		// Numbered as the records they are about are, so an entry and
		// its record's row say the same.
		r := recs[i]
		target, _ := r.Fields["target"].(string)
		id, _ := r.Fields["target_id"].(string)
		if t, ok := s.app.Types.Get(target); ok && id != "" {
			if rec, err := s.app.Store.Get(t.Name, id); err == nil {
				return addedRank(rec)
			}
		}
		return addedRank(r)
	})
}
