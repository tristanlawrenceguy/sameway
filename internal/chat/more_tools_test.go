package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A turn is given the core and more_tools, which names the rest; asked
// for one, the tool is given from the next step on.
func TestTheRestIsFoundByName(t *testing.T) {
	t.Parallel()
	svc, m := withModel(t,
		call("more_tools", map[string]any{"name": "set_setting"}),
		&llm.Response{Text: "ok"},
	)
	if _, err := svc.Send(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	offered := func(r llm.Request) (names string, more string) {
		for _, tl := range r.Tools {
			names += "," + tl.Name
			if tl.Name == "more_tools" {
				more = tl.Description
			}
		}
		return names + ",", more
	}
	first, more := offered(m.seen[0])
	if strings.Contains(first, ",set_setting,") || !strings.Contains(more, "set_setting (change a setting)") {
		t.Errorf("first the core, and more_tools naming the rest: %s\n%s", first, more)
	}
	if next, _ := offered(m.seen[1]); !strings.Contains(next, ",set_setting,") {
		t.Errorf("asked for, set_setting is given from the next step: %s", next)
	}
	// The answer names the tool's arguments: a model calls from it, and
	// guessed more_tools' own "name" for set_setting's "key".
	msgs := m.seen[1].Messages
	if said := msgs[len(msgs)-1].ToolResults[0].Content; !strings.Contains(said, "set_setting is yours from your next step on (needs key, value)") {
		t.Errorf("the answer names what the tool takes: %s", said)
	}
}
