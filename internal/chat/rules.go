package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant was told every rule it has before each answer: how to
// keep a habit, run a webhook, read a file, export a spreadsheet, set
// Sameway up, chart numbers, as well as the few every turn needs. Most
// turns need none of the topics, and a model reads 15 KB of instructions
// slower and weighs them less. A rule of one topic now comes with a turn
// that speaks of it, by the same words that bring a tool along
// (toolset.go), or when the conversation has used the tool it is about.
// It comes after the page, so the start of the prompt a model has read
// before (warm.go) is the same from one turn to the next.

// rule is one topic's rule: its words, and the words or tools that bring
// it.
type rule struct {
	Topic string // what more_tools brings it by (more_tools.go)
	Text  string
	Words []string
	Tools []string
}

// spoken is what a turn speaks of: the person's latest words, lowered,
// and the tools the conversation has called.
func spoken(history []llm.Message) (said string, used map[string]bool) {
	used = map[string]bool{}
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		for _, c := range m.ToolCalls {
			used[c.Name] = true
			if c.Name == moreToolsName { // what was asked for is used from then on
				used[askedFor(c.Args)] = true
			}
		}
		if m.Role == llm.RoleUser && said == "" {
			said = strings.ToLower(m.Content)
		}
	}
	return said, used
}

// topicalRules are the rules a turn speaks of, as the end of the prompt.
func topicalRules(history []llm.Message) string {
	said, used := spoken(history)
	var b strings.Builder
	for _, r := range topical {
		bring := speaksOf(said, r.Words) || used[r.Topic]
		for _, t := range r.Tools {
			bring = bring || used[t]
		}
		if bring {
			b.WriteString("- " + r.Text + "\n")
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "\nFor what this turn is about:\n" + b.String()
}
