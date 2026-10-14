package query_test

import (
	"strings"
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

// A list by title is in the order a person reads: Émile with the E's, not
// after Zoë, and Item 2 before Item 10. Dates still sort as dates.
func TestAListByTitleIsInReadingOrder(t *testing.T) {
	t.Parallel()
	st, typ := tasks(t)
	for _, title := range []string{"Zoë", "Item 10", "Émile", "apple", "Item 2"} {
		st.Create("task", map[string]any{"title": title})
	}
	recs, err := query.Filter(st, typ, nil, "title", 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range recs {
		got = append(got, r.Fields["title"].(string))
	}
	if want := "apple Émile Item 2 Item 10 Zoë"; strings.Join(got, " ") != want {
		t.Errorf("by title: %v, want %s", got, want)
	}
}

// title= is one name in any case and in either form a keyboard stores é
// in: one letter (Windows) or e and an accent (a Mac's file names).
func TestAnEqualsConditionIsOneNameInAnyForm(t *testing.T) {
	t.Parallel()
	st, typ := tasks(t)
	st.Create("task", map[string]any{"title": "Cafe\u0301 with Ana"})
	st.Create("task", map[string]any{"title": "Cafe with Ana"})
	for where, want := range map[string]int{"title=caf\u00e9 with ana": 1, "title!=Café with Ana": 1, "title=STRASSE": 0} {
		recs, err := query.Filter(st, typ, []string{where}, "", 0, time.Now())
		if err != nil || len(recs) != want {
			t.Errorf("%s finds %d, want %d (%v)", where, len(recs), want, err)
		}
	}
}
