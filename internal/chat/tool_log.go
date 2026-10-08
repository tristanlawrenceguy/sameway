package chat

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// logCall writes each tool the conversation's model called, its arguments
// and the start of what it was told, to the file SAMEWAY_TOOL_LOG names,
// when it names one: the bench reads it to say what a model on this
// computer did, as SAMEWAY_MCP_LOG says it for one that runs its tools
// over MCP. A line is the time, ok or refused, the tool, its arguments,
// and after a tab what it was told.
func logCall(call llm.ToolCall, r toolResult) {
	path := os.Getenv("SAMEWAY_TOOL_LOG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	outcome := "ok"
	if r.isErr {
		outcome = "refused"
	}
	fmt.Fprintf(f, "%s %s %s %s\t%s\n", time.Now().Format("15:04:05"), outcome, call.Name, trim.Clip(string(call.Args), 400), trim.Clip(strings.ReplaceAll(r.text, "\n", " "), 400))
}
