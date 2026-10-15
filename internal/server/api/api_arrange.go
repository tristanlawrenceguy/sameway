package api

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// apiArrange is arrange_canvas over REST: a whole tab laid out in one
// change, with the checks and the one log entry the assistant's has, since
// it goes through the same tool.
func (s *Service) apiArrange(w http.ResponseWriter, r *http.Request) {
	body, err := web.ReadBody(r)
	if err != nil {
		web.WriteError(w, err)
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
		web.WriteError(w, &schema.ValidationError{Problems: problems})
		return
	}
	blocks, _ := json.Marshal(body["blocks"])
	text, isErr := s.ChatFor(r).ByAgent(Agent(r)).Arrange(canvas, blocks)
	if isErr {
		web.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": web.APIError{Code: "invalid", Message: text}})
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"arranged": text, "layout": s.layoutBrief(canvas)})
}

// layoutBrief is how a tab reads with its first few things to look at,
// and where the rest are: GET /api/layout.
func (s *Service) layoutBrief(canvas string) string {
	return s.app.Chat.LayoutBrief(canvas, 3, "GET /api/layout?canvas="+url.QueryEscape(canvas))
}

// apiLayout is the whole of how a tab reads: its rows, every thing to look
// at, and the arrangement that would fix the order and widths.
func (s *Service) apiLayout(w http.ResponseWriter, r *http.Request) {
	canvas := r.URL.Query().Get("canvas")
	if !s.app.Chat.HasCanvas(canvas) {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": web.APIError{Code: "not_found", Message: "no tab with id " + canvas + `; GET /api/canvas lists them, and "" is Home`}})
		return
	}
	reads, problems, fix := s.app.Chat.LayoutParts(canvas)
	if problems == nil {
		problems = []string{}
	}
	answer := map[string]any{"canvas": canvas, "reads": reads, "to_look_at": problems}
	if fix != "" {
		answer["arrange"] = "POST /api/arrange with " + fix
	}
	web.WriteJSON(w, http.StatusOK, answer)
}

// layoutOf is how a block's tab reads after a write through the API, for
// the answer: the same line the assistant's tools end with.
func (s *Service) layoutOf(r *http.Request, rec *store.Record) string {
	if r.PathValue("type") != records.BlockType || rec == nil {
		return ""
	}
	canvas, _ := rec.Fields["canvas"].(string)
	return s.layoutBrief(canvas)
}
