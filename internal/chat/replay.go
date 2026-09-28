package chat

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// replayChars caps a tool result as it is replayed in history: enough to
// show what came back, not the whole page it came from.
const replayChars = 600

// used records one tool call on the message it helped answer: what was
// called, with what, and what came back.
func used(call llm.ToolCall, r toolResult) map[string]any {
	var args any
	if err := json.Unmarshal(call.Args, &args); err != nil || args == nil {
		args = map[string]any{}
	}
	return map[string]any{"id": call.ID, "name": call.Name, "args": args, "result": truncate(r.text, replayChars), "error": r.isErr}
}

// replay turns a message's tools back into the turn the model made: the
// calls, then their results, ahead of the words it said. The history then
// shows tools being used rather than replies that mention them, and a
// model that copies its earlier turns copies the tool use. Without this a
// local model, after one reply that said "I created the note, it's at
// /t/note/…", said that again instead of calling anything.
func replay(v any) []llm.Message {
	items, _ := v.([]any)
	calls := llm.Message{Role: llm.RoleAssistant}
	results := llm.Message{Role: llm.RoleTool}
	for _, it := range items {
		m, _ := it.(map[string]any)
		id, _ := m["id"].(string)
		name, _ := m["name"].(string)
		if id == "" || name == "" {
			continue
		}
		raw, err := json.Marshal(m["args"])
		if err != nil {
			raw = []byte("{}")
		}
		result, _ := m["result"].(string)
		isErr, _ := m["error"].(bool)
		calls.ToolCalls = append(calls.ToolCalls, llm.ToolCall{ID: id, Name: name, Args: raw})
		results.ToolResults = append(results.ToolResults, llm.ToolResult{CallID: id, Content: result, IsError: isErr})
	}
	if len(calls.ToolCalls) == 0 {
		return nil
	}
	return []llm.Message{calls, results}
}
