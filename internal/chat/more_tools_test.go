package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A turn that speaks of nothing is given the core and more_tools, which
// names the rest; asked for one, the tool is given from the next step on,
// and asked for a topic, its rules are the answer. A turn that speaks of
// a topic has its rules at the end of the prompt, and one that does not
// has none of them.
func TestTheRestIsFoundByName(t *testing.T) {
	t.Parallel()
	svc, m := withModel(t,
		call("more_tools", map[string]any{"name": "set_setting"}),
		call("more_tools", map[string]any{"name": "habits"}),
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
	if strings.Contains(first, ",set_setting,") || !strings.Contains(more, "set_setting (change a setting)") || !strings.Contains(more, "habits") {
		t.Errorf("first the core, and more_tools naming the rest: %s\n%s", first, more)
	}
	if strings.Contains(m.seen[0].System, "To keep something up or within") {
		t.Error("hi speaks of no habit, so its rules are not given")
	}
	if next, _ := offered(m.seen[1]); !strings.Contains(next, ",set_setting,") {
		t.Errorf("asked for, set_setting is given from the next step: %s", next)
	}
	if res := lastToolResult(m.seen[2]); !strings.Contains(res.Content, "To keep something up or within") && !strings.Contains(res.Content, "create_record a habit") {
		t.Errorf("asked for a topic, its rules are the answer: %s", res.Content)
	}

	svc2, m2 := withModel(t, &llm.Response{Text: "ok"})
	svc2.Send(context.Background(), "I want to drink 8 glasses of water every day")
	if !strings.Contains(m2.seen[0].System, "For what this turn is about:") || !strings.Contains(m2.seen[0].System, "create_record a habit") {
		t.Error("a turn about a habit has the habit rules")
	}
}
