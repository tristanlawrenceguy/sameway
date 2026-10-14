package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A small model on the person's own computer read every tool Sameway has
// before each answer, some thirty, 34 KB, when a message needs a few: the
// settings alone are 7 KB. It read slower, and chose among more than it
// could weigh. A model is now given the tools every turn needs, and the
// others only when the message speaks of what they do, or when the
// conversation has used them already. Measured with Qwen 3.5 (two runs a
// request): the hard requests took 22 seconds instead of 33, passing 10
// of 26 against 11, and the newcomer's 17 instead of 19, passing 12 of 20
// against 14, within what one run to the next varies by. A model
// elsewhere was given them all at first, as one that reads them in no
// time; it pays for them in tokens on every turn all the same, and is
// given the same now. Each Op says whether it is Core, given every turn,
// and the Words in a message that bring it along; the rules of a topic
// come the same way (rules.go).

// toolsFor is the tools a turn is given: the core, those the message's
// words bring, and those the conversation has called.
func (s *Service) toolsFor(all []llm.Tool, history []llm.Message) []llm.Tool {
	said, used := spoken(history)
	// The core comes first and in the same order every turn, so what a
	// model has already read of a turn before (warm.go) is read again
	// only from where this turn differs.
	var core, more []llm.Tool
	for _, t := range all {
		switch {
		case opNamed(t.Name).Core:
			core = append(core, t)
		case used[t.Name] || speaksOf(said, opNamed(t.Name).Words):
			more = append(more, t)
		}
	}
	given := append(core, more...)
	if t, ok := moreTool(all, given, history); ok { // more_tools.go: the rest, by name
		given = append(given, t)
	}
	return given
}

func speaksOf(said string, words []string) bool {
	for _, w := range words {
		if strings.Contains(said, w) {
			return true
		}
	}
	return false
}

// refresh is a turn's next step's prompt and tools: the page as it is now,
// and what more_tools brought (more_tools.go).
func (s *Service) refresh(req *llm.Request) {
	req.System = s.systemPrompt(req.Messages)
	req.Tools = s.toolsFor(s.Tools(), req.Messages)
}
