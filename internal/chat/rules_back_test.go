package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// Every topic's rules are given every turn, in whatever language the
// person writes: coming back, the week and the pile, when each dose
// reminds, offering a day and naming what still gives the old one.
func TestEveryTurnHasEveryRule(t *testing.T) {
	t.Parallel()
	svc, m := withModel(t, &llm.Response{Text: "ok"})
	svc.Send(context.Background(), "Ich bin zurück")
	for _, w := range []string{"keep them as tasks with no date", "When the person is back after time away", "were kept while they were away", "never a day already past", "look in notes too for an appointment", "with remind set to each time", "offer once to put it on its day", "still give the old one and name them", "call organise_writing"} {
		if !strings.Contains(m.seen[0].System, w) {
			t.Errorf("the prompt has %q", w)
		}
	}
}
