package chat

import "fmt"

// toolResult is what one tool call produced: text for the model, an error
// flag, and the change to record if any.
type toolResult struct {
	text   string
	isErr  bool
	change *Change
	// changes is for a tool that makes several, such as an arrangement:
	// each is logged and shown on its own.
	changes []Change
}

func fail(format string, args ...any) toolResult {
	return toolResult{text: fmt.Sprintf(format, args...), isErr: true}
}
