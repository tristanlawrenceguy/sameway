package server

import (
	"encoding/json"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// apiArrange is arrange_canvas over REST: a whole tab laid out in one
// change, with the checks and the one log entry the assistant's has, since
// it goes through the same tool.
func (s *Server) apiArrange(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	problems := map[string]string{}
	for k := range body {
		if k != "blocks" && k != "canvas" {
			problems[k] = "not something arrange takes: it takes blocks and canvas"
		}
	}
	canvas, ok := body["canvas"].(string)
	if body["canvas"] != nil && !ok {
		problems["canvas"] = `the tab, as a canvas id; "" or left out is Home`
	}
	if _, ok := body["blocks"].([]any); !ok {
		problems["blocks"] = `every block on the tab in reading order, as [{"id", "span", "region", "frame", "size"}]`
	}
	if len(problems) > 0 {
		writeError(w, &schema.ValidationError{Problems: problems})
		return
	}
	blocks, _ := json.Marshal(body["blocks"])
	text, isErr := s.chatFor(r).ByAgent(apiAgent(r)).Arrange(canvas, blocks)
	if isErr {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: text}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"arranged": text, "layout": s.app.Chat.LayoutNow(canvas)})
}

// layoutOf is how a block's tab reads after a write through the API, for
// the answer: the same line the assistant's tools end with.
func (s *Server) layoutOf(r *http.Request, rec *store.Record) string {
	if r.PathValue("type") != chat.BlockType || rec == nil {
		return ""
	}
	canvas, _ := rec.Fields["canvas"].(string)
	return s.app.Chat.LayoutNow(canvas)
}
