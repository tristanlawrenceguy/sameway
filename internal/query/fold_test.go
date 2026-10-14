package query_test

import (
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
)

// A condition on words reads them as search does: case and accents do not
// matter, so title~sao finds São Paulo and title~CAFE finds Café; words
// that are not there still find nothing.
func TestAWordsConditionIgnoresCaseAndAccents(t *testing.T) {
	t.Parallel()
	st, typ := tasks(t)
	st.Create("task", map[string]any{"title": "Fly to São Paulo"})
	st.Create("task", map[string]any{"title": "Café with Ana"})
	st.Create("task", map[string]any{"title": "Water the plants"})
	now := time.Now()
	for where, want := range map[string]int{"title~sao": 1, "title~CAFE": 1, "title~são": 1, "title~lisbon": 0} {
		recs, err := query.Filter(st, typ, []string{where}, "", 0, now)
		if err != nil || len(recs) != want {
			t.Errorf("%s finds %d, want %d (%v)", where, len(recs), want, err)
		}
	}
}

// Quotes and dashes match as any keyboard types them: a phone's curly
// apostrophe finds a laptop's straight one, and the other way round.
func TestAWordsConditionIgnoresCurlyQuotes(t *testing.T) {
	t.Parallel()
	st, typ := tasks(t)
	st.Create("task", map[string]any{"title": "Mum's birthday"})
	st.Create("task", map[string]any{"title": "Ana’s visit – Friday"})
	now := time.Now()
	for where, want := range map[string]int{"title~mum’s": 1, "title~ana's": 1, "title~visit - friday": 1} {
		recs, err := query.Filter(st, typ, []string{where}, "", 0, now)
		if err != nil || len(recs) != want {
			t.Errorf("%s finds %d, want %d (%v)", where, len(recs), want, err)
		}
	}
}
