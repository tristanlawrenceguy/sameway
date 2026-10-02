package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// noteCall writes each tool called, with its arguments and, when it was
// refused, the start of why, to the file SAMEWAY_MCP_LOG names, when it
// names one: how to watch what a model reached for, reads included, which
// the activity log does not keep. A line is the time, ok or refused, the
// tool, its arguments, and after a tab the refusal.
func noteCall(name string, args json.RawMessage, refused bool, text string) {
	path := os.Getenv("SAMEWAY_MCP_LOG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	outcome, why := "ok", ""
	if refused {
		outcome, why = "refused", "\t"+clip(strings.ReplaceAll(text, "\n", " "), 300)
	}
	fmt.Fprintf(f, "%s %s %s %s%s\n", time.Now().Format("15:04:05"), outcome, name, clip(string(args), 300), why)
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
