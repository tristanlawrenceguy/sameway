package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/automate"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// toolResult is what one tool call produced: text for the model, an error
// flag, and the change to record if any.
type toolResult struct {
	text   string
	isErr  bool
	change *records.Change
	// changes is for a tool that makes several, such as an arrangement:
	// each is logged and shown on its own.
	changes []records.Change
	// answer is what an action got back, bare: a webhook's reply, a
	// command's output; what the next action is given (internal/automate).
	answer string
}

func fail(format string, args ...any) toolResult {
	return toolResult{text: fmt.Sprintf(format, args...), isErr: true}
}

// ran is what an action did (internal/automate) as a tool's result.
func ran(r automate.Result) toolResult {
	return toolResult{text: r.Text, isErr: r.IsErr, change: r.Change, changes: r.Changes, answer: r.Answer}
}

// result is a tool's result as what an action did, for internal/automate.
func (r toolResult) result() automate.Result {
	return automate.Result{Text: r.text, IsErr: r.isErr, Change: r.change, Changes: r.changes, Answer: r.answer}
}
