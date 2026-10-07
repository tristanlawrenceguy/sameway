package chat

import "github.com/tristanlawrenceguy/sameway/internal/llm"

// Describe is describe, for tests outside the package.
var Describe = describe

// ToolsFor is toolsFor, for tests outside the package.
func (s *Service) ToolsFor(all []llm.Tool, history []llm.Message) []llm.Tool {
	return s.toolsFor(all, history)
}
