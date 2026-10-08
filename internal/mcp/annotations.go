package mcp

import "github.com/tristanlawrenceguy/sameway/internal/chat"

// What each tool is like, as MCP's annotations say it, so a client can ask
// its person before what changes or reaches outside, and run what only
// reads without asking. The assistant's tools say it in their Op
// (chat.Traits); the three that are MCP's own say it here. Every tool
// listed has a title (TestEveryToolSaysWhatItIs), and every tool said to
// read only is called and the workspace checked unchanged.

type ownTool struct {
	title string
	chat.Traits
}

var ownTools = map[string]ownTool{
	"describe": {"Describe the workspace", chat.Traits{ReadOnly: true, Idempotent: true}},
	"look":     {"Read a page", chat.Traits{ReadOnly: true, Idempotent: true}},
	"try":      {"Try a change without making it", chat.Traits{ReadOnly: true, Idempotent: true}},
}

// traitsOf is a tool's title and traits, from MCP's own or the registry.
func traitsOf(name string) (string, chat.Traits, bool) {
	if t, ok := ownTools[name]; ok {
		return t.title, t.Traits, true
	}
	op, ok := chat.OpFor(name)
	return op.Title, op.Traits, ok
}

// readOnly says whether a tool only reads.
func readOnly(name string) bool {
	_, tr, _ := traitsOf(name)
	return tr.ReadOnly
}

// annotations are a tool's traits as MCP spells them, or nil for a tool
// it does not know.
func annotations(name string) map[string]any {
	title, tr, ok := traitsOf(name)
	if !ok {
		return nil
	}
	return map[string]any{"title": title, "readOnlyHint": tr.ReadOnly, "destructiveHint": tr.Destructive,
		"idempotentHint": tr.Idempotent, "openWorldHint": tr.OpenWorld}
}
