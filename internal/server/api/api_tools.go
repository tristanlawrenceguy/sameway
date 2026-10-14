package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// POST /api/tools/{name} is any of the assistant's tools for an agent over
// REST, as MCP has them: the same op from the registry (chat/op.go), run
// through the same chat service, so the same access (the tools the one
// asking may have), the same questions before what cannot be taken back,
// the same checks when a block is written and the same entry in the log,
// named for the agent. The body is the tool's arguments, as its schema in
// GET /api/describe/tools/{name} says.

func (s *Service) apiTool(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	svc := s.ChatFor(r)
	if !offers(svc, name) {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": web.APIError{Code: "not_found",
			Message: "there is no tool " + name + " you may call here; GET /api/describe/tools lists them", Describe: "/api/describe/tools"}})
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		web.WriteError(w, err)
		return
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		raw = []byte("{}")
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": web.APIError{Code: "bad_request",
			Message: "the body is the tool's arguments as a JSON object: " + web.JSONTrouble(err)}})
		return
	}
	text, isErr := svc.ByAgent(Agent(r)).Call(name, raw)
	if isErr {
		web.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": web.APIError{Code: "invalid", Message: text}})
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "tool": name, "result": text})
}

// offers says whether the one this service speaks for may call a tool.
func offers(svc *chat.Service, name string) bool {
	for _, t := range svc.Tools() {
		if t.Name == name {
			return true
		}
	}
	return false
}

// ToolOutward says whether a path calls a tool that reaches beyond the
// workspace, which a dry run cannot try (server/dry_run.go).
func ToolOutward(path string) bool {
	name, ok := strings.CutPrefix(path, "/api/tools/")
	if !ok {
		return false
	}
	op, _ := chat.OpFor(name)
	return op.OpenWorld
}
