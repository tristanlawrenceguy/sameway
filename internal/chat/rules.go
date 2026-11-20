package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The rules of each topic (how to keep a habit, run a webhook, read a
// file, set Sameway up) came with a turn whose message held one of the
// topic's words, to spare a model 10 KB of instructions most turns do not
// need. Words work in one language only. Brought instead by what the
// assistant did, a rule came too late: it made the habit and answered
// before the habit's rule arrived. Given every turn, the rules cost no
// more calls or seconds with Claude Haiku (two runs of each request:
// 58 of 60 passed, against 57 by words and 51 by what was done), and they
// are the same text every turn, so they come before the page, in the part
// a model has read before (warm.go).

// rule is one topic's rule.
type rule struct {
	Topic string
	Text  string
}

// topicRules are every topic's rules, as the prompt gives them.
func topicRules() string {
	var b strings.Builder
	b.WriteString("\n\nBy topic (a tool named here that you were not given, more_tools gives you):\n")
	for _, r := range topical {
		b.WriteString("- " + r.Text + "\n")
	}
	return b.String()
}

// calledSo is the tools the conversation has called, and those more_tools
// brought.
func calledSo(history []llm.Message) map[string]bool {
	did := map[string]bool{}
	for _, m := range history {
		for _, c := range m.ToolCalls {
			did[c.Name] = true
			if c.Name == moreToolsName { // what was asked for is used from then on
				did[askedFor(c.Args)] = true
			}
		}
	}
	return did
}
