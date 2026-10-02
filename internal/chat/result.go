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
	// answer is what an action got back, bare: a webhook's reply, a
	// command's output; what the next action is given (automate.go).
	answer string
}

func fail(format string, args ...any) toolResult {
	return toolResult{text: fmt.Sprintf(format, args...), isErr: true}
}
