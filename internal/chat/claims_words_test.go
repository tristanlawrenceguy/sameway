package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A reply naming one of the tools is asked once to say the same in the
// person's words, and the person gets that; named again, the reply stands
// with nothing said of changes it never claimed; a reply in words passes.
func TestARepliesNamingToolsIsAskedForThePersonsWords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		{Text: "I couldn't use add_component for this, because it places blocks on the page."},
		{Text: "I couldn't put that on the page. Tell me which page you meant."},
	}}
	svc.Provider = m
	msg, err := svc.Send(context.Background(), "put my tasks somewhere")
	if err != nil {
		t.Fatal(err)
	}
	if got := msg.Fields["content"].(string); got != "I couldn't put that on the page. Tell me which page you meant." {
		t.Errorf("the person gets the reply in their words: %q", got)
	}
	if len(m.seen) != 2 || !strings.Contains(m.seen[1].Messages[len(m.seen[1].Messages)-1].Content, "your reply names add_component") {
		t.Fatalf("the model is asked once, naming the tool")
	}

	stubborn := &scripted{steps: []*llm.Response{{Text: "Use find_records."}, {Text: "Use find_records, really."}}}
	svc.Provider = stubborn
	msg, _ = svc.Send(context.Background(), "how do I look things up")
	if got := msg.Fields["content"].(string); got != "Use find_records, really." {
		t.Errorf("named again, the reply stands as it is, with nothing added: %q", got)
	}

	plain := &scripted{steps: []*llm.Response{{Text: "Your tasks are on the Work tab."}}}
	svc.Provider = plain
	svc.Send(context.Background(), "where are my tasks")
	if len(plain.seen) != 1 {
		t.Errorf("a reply in words is not asked again")
	}
}
