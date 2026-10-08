package app

import (
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant's tools, as an agent reads them: each from the registry
// (chat/op.go), the same list the model is given and MCP's tools/list
// carries, with what a person calls it, what it is like, who may call it
// and where.

// DescribedTool is one tool the assistant can call, with its argument
// schema and how to call it over REST.
type DescribedTool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
	// Who is who may call it: view (whoever may look), edit (whoever may
	// change the workspace) or owner.
	Who    string      `json:"who"`
	Traits chat.Traits `json:"traits"`
	Call   string      `json:"call"`
}

// toolsRoute says how to call a tool over REST.
const toolsRoute = `POST /api/tools/{name} with the tool's arguments as a JSON object calls one of the assistant's tools, the ones GET /api/describe/tools lists with their schema (who says who may call it: view, edit or owner), the same as MCP's tools/call: the same checks, the same questions put to the person before what cannot be taken back, the same log entry, named for you. The answer is {"ok": true, "tool", "result": what the tool said}, or 422 {"error": {"code": "invalid", "message": what the tool said was wrong}}; a tool that is not there, or not yours to call, is 404`

var who = map[chat.Access]string{chat.ForViewers: "view", chat.ForEditors: "edit", chat.ForOwner: "owner"}

func describeTools(tools []llm.Tool) []DescribedTool {
	var out []DescribedTool
	for _, t := range tools {
		op, _ := chat.OpFor(t.Name)
		out = append(out, DescribedTool{Name: t.Name, Title: op.Title, Description: t.Description, Schema: t.Schema,
			Who: who[op.Access], Traits: op.Traits, Call: "POST /api/tools/" + t.Name})
	}
	return out
}
