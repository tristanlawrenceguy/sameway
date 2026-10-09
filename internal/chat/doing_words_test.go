package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// Every tool the assistant has is said in a person's words while a turn
// runs: "Create canvas" went by on the page, a tool's own name with the
// word canvas the assistant is told never to use with people.
func TestEveryToolIsSaidInWords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	for _, tool := range svc.Tools() {
		got := chat.Describe(llm.ToolCall{Name: tool.Name, Args: []byte("{}")})
		raw := strings.ToUpper(tool.Name[:1]) + strings.ReplaceAll(tool.Name[1:], "_", " ")
		if got == raw || strings.Contains(strings.ToLower(got), "canvas") {
			t.Errorf("%s goes by as %q", tool.Name, got)
		}
	}
}
