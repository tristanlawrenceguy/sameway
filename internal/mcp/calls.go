package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// noteCall writes each tool called, with its arguments, to the file
// SAMEWAY_MCP_LOG names, when it names one: how to watch what a model
// reached for, reads included, which the activity log does not keep.
func noteCall(name string, args json.RawMessage) {
	path := os.Getenv("SAMEWAY_MCP_LOG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	a := string(args)
	if len(a) > 300 {
		a = a[:300] + "..."
	}
	fmt.Fprintf(f, "%s %s %s\n", time.Now().Format("15:04:05"), name, a)
}
