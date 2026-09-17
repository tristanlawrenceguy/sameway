package llm_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestHelperCLI is the program under test when SAMEWAY_FAKE_CLI is set:
// it stands in for Claude Code, prints JSON with a result field, and
// shows it was handed the prompt, the system prompt and an MCP
// configuration.
func TestHelperCLI(t *testing.T) {
	if os.Getenv("SAMEWAY_FAKE_CLI") == "" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	var prompt, mcp, system string
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "-p":
			prompt = args[i+1]
		case "--mcp-config":
			mcp = args[i+1]
		case "--append-system-prompt":
			system = args[i+1]
		}
	}
	raw, _ := os.ReadFile(mcp)
	stdin, _ := io.ReadAll(os.Stdin)
	fmt.Printf("{\"result\": %q}", "echo: "+prompt+" | system: "+system+" | stdin: "+string(stdin)+" | mcp: "+string(raw))
	os.Exit(0)
}

// A command provider runs the program the person is signed in to with
// the conversation as its prompt and this workspace's MCP server as its
// tools, and takes the reply from the JSON field named.
func TestACommandProviderRunsTheSignedInProgram(t *testing.T) {
	c := &llm.Command{
		Template:   os.Args[0] + " -test.run=TestHelperCLI -- -p {prompt} --append-system-prompt {system} --mcp-config {mcp} --output-format json",
		Field:      "result",
		Workspace:  t.TempDir(),
		Executable: "/bin/sameway",
	}
	os.Setenv("SAMEWAY_FAKE_CLI", "1")
	defer os.Unsetenv("SAMEWAY_FAKE_CLI")
	resp, err := c.Complete(context.Background(), llm.Request{System: "Be brief.", Messages: []llm.Message{
		{Role: llm.RoleUser, Content: "make a note"},
		{Role: llm.RoleAssistant, Content: "Done.", ToolCalls: []llm.ToolCall{{ID: "1", Name: "create_record"}}},
		{Role: llm.RoleUser, Content: "and a task"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"echo: Person: make a note", "You: Done.", "You used create_record", "Now the person says:\nand a task", "system: Be brief.", "\"command\":\"/bin/sameway\"", "\"--workspace\",", "\"mcp\"]"} {
		if !strings.Contains(resp.Text, want) {
			t.Errorf("the program should get the conversation, the system prompt and the MCP config; missing %q in:\n%s", want, resp.Text)
		}
	}
	if !c.ToolsOutside() {
		t.Error("a command provider runs its tools outside")
	}

	// Without the placeholders, the system prompt and the conversation go
	// on stdin, which is what claude -p reads and what a command line on
	// Windows cannot carry.
	c.Template = os.Args[0] + " -test.run=TestHelperCLI -- --mcp-config {mcp} --output-format json"
	resp, err = c.Complete(context.Background(), llm.Request{System: "Be brief.", Messages: []llm.Message{{Role: llm.RoleUser, Content: "make a note"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text, "echo:  | system:  | stdin: Be brief.\n\n---\n\nmake a note") {
		t.Errorf("stdin should carry the system prompt and the conversation: %s", resp.Text)
	}

	// The preset fills the template in, and a missing template is an error.
	p, err := llm.New(llm.Config{Provider: "claude-code", Model: "sonnet", Workspace: "/w"})
	if err != nil || p.Name() != "Claude Code" {
		t.Errorf("the claude-code preset should be a provider called Claude Code: %v %v", p, err)
	}
	if _, err := llm.New(llm.Config{Provider: "command"}); err == nil || !strings.Contains(err.Error(), "llm.command") {
		t.Errorf("provider command needs a command line: %v", err)
	}
}
