package query_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/query"
)

// Conditions that ask one field for two values can never all hold, and
// say so with a fix; ones that can hold together are left alone.
func TestConditionsThatCanNeverAllHoldAreSaid(t *testing.T) {
	_, typ := tasks(t)
	for _, c := range []struct {
		where []string
		want  string
	}{
		{[]string{"done=true", "done=false"}, "done=true and done=false can never both hold, since a task has one done, so it would never show anything; use one value, or one block per value"},
		{[]string{"title=Dig", "effort=1", "title=Weed"}, "title=Dig and title=Weed can never both hold"},
		{[]string{"done=true", "done=yes"}, ""},        // one value, said twice
		{[]string{"effort=3", "effort=3.0"}, ""},       // one number
		{[]string{"title=Dig", "title=dig"}, ""},       // words match without case
		{[]string{"tags=garden", "tags=shopping"}, ""}, // a task can have both tags
		{[]string{"due=today", "due=2026-09-17"}, ""},  // may be one day
		{[]string{"effort>=1", "effort=3"}, ""},        // not two equals
		{[]string{"owner=me", "owner=you"}, ""},        // a wrong field is Filter's to say
	} {
		got := query.Contradiction(typ, c.where)
		if c.want == "" && got != "" || c.want != "" && !strings.HasPrefix(got, c.want) {
			t.Errorf("%v: want %q, got %q", c.where, c.want, got)
		}
	}
}
