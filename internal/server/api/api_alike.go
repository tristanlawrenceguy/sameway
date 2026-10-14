package api

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record an agent makes that is already there. The assistant is told
// when it makes a task called as open ones are (chat's same_title.go); an
// agent writing through the API was not, and the workspaces agents work
// in filled with "Dentist appointment" twice and "Pick up dry cleaning"
// three times, each made by a run that could not see the last. The answer
// to a create now says the open records called the same, with what to do.

// alikeSaid is the open records of rec's type called as rec is.
type alikeSaid struct {
	Records []alikeRecord `json:"records"`
	Note    string        `json:"note"`
}

type alikeRecord struct {
	ID   string `json:"id"`
	Page string `json:"page"`
}

// alikeOf is what the answer says of records like rec, nil when none.
func (s *Service) alikeOf(rec *store.Record) *alikeSaid {
	t, ok := s.app.Store.Types().Get(rec.Type)
	if !ok {
		return nil
	}
	alike := records.Alike(s.app.Store, t, rec)
	if len(alike) == 0 {
		return nil
	}
	out := &alikeSaid{Note: fmt.Sprintf("%s called the same %s open already. If this is one of those, delete this one and use that; if they are one thing twice, POST /api/tools/keep_one makes them one.",
		schema.Count(len(alike), t.Name), map[bool]string{true: "was", false: "were"}[len(alike) == 1])}
	for _, r := range alike {
		out.Records = append(out.Records, alikeRecord{ID: r.ID, Page: relate.Page(t.Name, r.ID)})
	}
	return out
}
