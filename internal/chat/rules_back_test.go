package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// Back after time away, the assistant is told to catch the person up from
// what waits and what is late, and to offer Today's keeping the long late
// with no date; asked to track tablets, to set when each dose reminds.
func TestTheRulesForComingBackAndMedicines(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ said, want string }{
		{"I'm back now, need to catch up on everything", "keep them as tasks with no date"},
		{"Mum has morning and evening pills", "with remind set to each time"},
	} {
		svc, m := withModel(t, &llm.Response{Text: "ok"})
		svc.Send(context.Background(), c.said)
		if !strings.Contains(m.seen[0].System, c.want) {
			t.Errorf("%q brings %q", c.said, c.want)
		}
	}
}
