package chat

import (
	"strings"
	"unicode"
)

// An agent is a program outside Sameway that changes the workspace:
// Claude Code or ChatGPT over MCP, a script through the API. What it does
// is logged as the agent's, by the name it gave, so the person can tell
// it from their own changes and from the assistant in the app, and undo
// it the same way. The assistant in the app stays "Assistant", even when
// its model runs the tools over MCP in another program.

// ActorAgent is the log's actor for an agent; the others are human,
// assistant and system.
const ActorAgent = "agent"

// ThroughMCP says an agent came in over the Model Context Protocol, as
// ThroughAPI says it came through the HTTP API.
const ThroughMCP = "through MCP"

// Agent is who an outside change came from and how it came in.
type Agent struct {
	// Name is what the agent calls itself: the client's name on MCP, the
	// X-Sameway-Agent header or the User-Agent's product on the API. ""
	// when it said nothing, which reads as "An agent".
	Name string
	// Through is ThroughMCP or ThroughAPI.
	Through string
}

// Who is how the log names the agent: "Claude Code (through MCP)",
// "An agent (through the API)".
func (a Agent) Who() string {
	return AgentWho(a.Name, a.Through)
}

// machineNames are libraries' names that entries written before agents
// stopped being named by their User-Agent still hold: said as "An agent".
var machineNames = map[string]bool{
	"Go-http-client": true,
}

// MachineName says whether a stored name is a library's, said as "An agent".
func MachineName(name string) bool { return machineNames[name] }

// AgentWho is Agent.Who from a log entry's by and via.
func AgentWho(name, through string) string {
	if name == "" || machineNames[name] {
		name = "An agent"
	}
	if strings.HasPrefix(through, "through ") {
		return name + " (" + through + ")"
	}
	return name
}

// knownClients are the names MCP hosts give themselves in clientInfo,
// said the way a person knows them.
var knownClients = map[string]string{
	"claude-code":      "Claude Code",
	"claude-ai":        "Claude",
	"openai-mcp":       "ChatGPT",
	"codex-mcp-client": "Codex",
}

// AgentName cleans a name an agent gave for itself into one the log can
// show: a known client by its product's name, anything else kept as given
// but on one line, without control characters, and short. The name is
// the agent's own claim, so it is shown, never trusted.
func AgentName(name string) string {
	if known, ok := knownClients[strings.ToLower(strings.TrimSpace(name))]; ok {
		return known
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, name)
	name = strings.Join(strings.Fields(name), " ")
	if r := []rune(name); len(r) > 60 {
		name = string(r[:59]) + "…"
	}
	return name
}

// ByAgent is this service acting for an agent: every tool it runs is
// logged, and every block it places is marked, as that agent's.
func (s *Service) ByAgent(a Agent) *Service {
	c := *s
	c.agent = &a
	return &c
}

// actor is who the log says ran this service's tools.
func (s *Service) actor() string {
	if s.agent != nil {
		return ActorAgent
	}
	return "assistant"
}

// byWho adds the agent's name and way in to a change it made.
func (s *Service) byWho(c Change) Change {
	if s.agent != nil {
		c.By, c.Via = s.agent.Name, s.agent.Through
	}
	return c
}

// marked is a block's fields with who changed it: the actor, and the
// agent's name when an agent did.
func (s *Service) marked(fields map[string]any) map[string]any {
	fields["actor"] = s.actor()
	if s.agent != nil {
		fields["agent"] = s.agent.Name
	}
	return fields
}

// As is who an agent is in the log: by the name it gave, through the way
// it came in.
func (a Agent) As() Who { return Who{Actor: ActorAgent, By: a.Name, Via: a.Through} }
