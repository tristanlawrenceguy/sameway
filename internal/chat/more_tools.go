package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A turn is given the core tools and what its words speak of (toolset.go,
// rules.go), and a model cannot call a tool it was not given, nor follow a
// rule it never read. more_tools names the rest, a few words each, and
// hands over the one asked for: a tool is given from the next step on, for
// the rest of the conversation; a topic's rules are the answer. It is the
// conversation's own, not an op: MCP and the API list every tool already.

const moreToolsName = "more_tools"

// askedFor is the name a more_tools call asked for.
func askedFor(args json.RawMessage) string {
	var a struct{ Name string }
	json.Unmarshal(args, &a)
	return strings.TrimSpace(a.Name)
}

// moreTool is more_tools as this turn offers it, naming what it was not
// given, or false when it was given everything.
func moreTool(all, given []llm.Tool, history []llm.Message) (llm.Tool, bool) {
	have := map[string]bool{}
	for _, t := range given {
		have[t.Name] = true
	}
	var names, lines []string
	for _, t := range all {
		if !have[t.Name] {
			names = append(names, t.Name)
			lines = append(lines, t.Name+" ("+strings.ToLower(opNamed(t.Name).Title)+")")
		}
	}
	brought := topicalRules(history)
	var topics []string
	for _, r := range topical {
		if !strings.Contains(brought, r.Text) {
			names = append(names, r.Topic)
			topics = append(topics, r.Topic)
		}
	}
	if len(names) == 0 {
		return llm.Tool{}, false
	}
	sort.Strings(lines)
	desc := "What else you can do, beyond the tools this turn was given, and the rules of topics it did not speak of. Call with one's name: a tool is given to you from your next step on, a topic's rules are the answer. Tools: " + strings.Join(lines, ", ") + "."
	if len(topics) > 0 {
		desc += " Topics: " + strings.Join(topics, ", ") + "."
	}
	return llm.Tool{Name: moreToolsName, Description: desc, Schema: obj(map[string]any{
		"name": map[string]any{"type": "string", "enum": names},
	}, "name")}, true
}

// moreTools answers a more_tools call.
func (s *Service) moreTools(call llm.ToolCall) toolResult {
	name := askedFor(call.Args)
	for _, r := range topical {
		if r.Topic == name {
			return toolResult{text: r.Text}
		}
	}
	for _, t := range s.Tools() {
		if t.Name == name {
			return toolResult{text: fmt.Sprintf("%s is yours from your next step on: %s", name, t.Description)}
		}
	}
	return fail("there is no tool or topic called %q; more_tools' description lists them", name)
}

// MoreTool is more_tools for a turn given these of all the tools, as MCP
// lists it for the assistant whose model runs its tools there.
func MoreTool(all, given []llm.Tool) (llm.Tool, bool) {
	return moreTool(all, given, nil)
}
