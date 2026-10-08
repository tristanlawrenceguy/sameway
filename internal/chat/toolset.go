package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A small model on the person's own computer read every tool Sameway has
// before each answer, some thirty, 34 KB, when a message needs a few: the
// settings alone are 7 KB. It read slower, and chose among more than it
// could weigh. A model on this computer is now given the tools every turn
// needs, and the others only when the message speaks of what they do, or
// when the conversation has used them already. A model elsewhere is given
// them all, as before: it reads them in no time and weighs them well.
// Measured with Qwen 3.5 (two runs a request): the hard requests took 22
// seconds instead of 33, passing 10 of 26 against 11, and the newcomer's
// 17 instead of 19, passing 12 of 20 against 14, within what one run to
// the next varies by. Each Op says whether it is Core, given every turn,
// and the Words in a message that bring it along.

// smallHere says whether a provider is a model on this computer, which
// is given the tools a turn needs rather than all of them.
func smallHere(p llm.Provider) bool {
	o, ok := p.(*llm.OpenAI)
	return ok && llm.IsOllama(o.BaseURL)
}

// toolsFor is the tools a turn is given: all of them for a model
// elsewhere; for one here, the core, those the message's words bring,
// and those the conversation has called.
func (s *Service) toolsFor(all []llm.Tool, history []llm.Message) []llm.Tool {
	if !smallHere(s.Provider) {
		return all
	}
	used := map[string]bool{}
	said := ""
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		for _, c := range m.ToolCalls {
			used[c.Name] = true
		}
		if m.Role == llm.RoleUser && said == "" {
			said = strings.ToLower(m.Content)
		}
	}
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
	return append(core, more...)
}

func speaksOf(said string, words []string) bool {
	for _, w := range words {
		if strings.Contains(said, w) {
			return true
		}
	}
	return false
}
