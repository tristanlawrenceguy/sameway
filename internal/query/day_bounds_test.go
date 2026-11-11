package query_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
)

// A small model asked for "everything on Thursday" wrote the day as both
// bounds, due>=Thursday 12 November and due<=Thursday 12 November, and
// found nothing: the second bound was read as Thursday's first instant.
// A whole day as a bound is that day, on either side, and ~ is any time
// on it; a moment as a bound is still that moment.
func TestADayAsABoundIsTheWholeDay(t *testing.T) {
	t.Parallel()
	st, typ := tasks(t)
	loc := time.FixedZone("CET", 3600)
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, loc)
	st.Create("task", map[string]any{"title": "Pay rent", "due": "2026-11-12T00:00:00Z"})  // a whole day
	st.Create("task", map[string]any{"title": "Call bank", "due": "2026-11-12T16:00:00Z"}) // 17:00 there
	st.Create("task", map[string]any{"title": "Water plants", "due": "2026-11-11T00:00:00Z"})
	titles := func(where ...string) string {
		t.Helper()
		recs, err := query.Filter(st, typ, where, "title", 0, now)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, r := range recs {
			out = append(out, r.Fields["title"].(string))
		}
		return strings.Join(out, "|")
	}
	for _, c := range []struct {
		where []string
		want  string
	}{
		{[]string{"due>=Thursday 12 November", "due<=Thursday 12 November"}, "Call bank|Pay rent"},
		{[]string{"due~2026-11-12"}, "Call bank|Pay rent"},
		{[]string{"due=~Thursday 12 November"}, "Call bank|Pay rent"},
		{[]string{"due<2026-11-12"}, "Water plants"},
		{[]string{"due>2026-11-11"}, "Call bank|Pay rent"},
		{[]string{"due<2026-11-12T12:00"}, "Pay rent|Water plants"},
	} {
		if got := titles(c.where...); got != c.want {
			t.Errorf("%v: got %q, want %q", c.where, got, c.want)
		}
	}
}
