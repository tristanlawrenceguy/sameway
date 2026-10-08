package chat

import "github.com/tristanlawrenceguy/sameway/internal/records"

// An agent is a program outside Sameway that changes the workspace:
// Claude Code or ChatGPT over MCP, a script through the API. What it does
// is logged as the agent's, by the name it gave, so the person can tell
// it from their own changes and from the assistant in the app, and undo
// it the same way. The assistant in the app stays "Assistant", even when
// its model runs the tools over MCP in another program.

// ByAgent is this service acting for an agent: every tool it runs is
// logged, and every block it places is marked, as that agent's.
func (s *Service) ByAgent(a records.Agent) *Service {
	c := *s
	c.agent = &a
	return &c
}

// actor is who the log says ran this service's tools.
func (s *Service) actor() string {
	if s.agent != nil {
		return records.ActorAgent
	}
	return "assistant"
}

// byWho adds the agent's name and way in to a change it made.
func (s *Service) byWho(c records.Change) records.Change {
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
