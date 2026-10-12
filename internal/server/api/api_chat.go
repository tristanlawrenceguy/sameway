package api

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// What pages show that /api did not: a chat turn with a file. What pages
// do is an agent's already, through the same forms (server/agents.go).

// apiChat is one turn of the conversation for an agent: the message, the
// tab it is about, and a file already added (POST /api/file/upload), whose
// text goes to the assistant with the message as it does for a person.
func (s *Service) apiChat(w http.ResponseWriter, r *http.Request) {
	body, err := web.ReadBody(r)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	problems := map[string]string{}
	for k := range body {
		if k != "message" && k != "canvas" && k != "file" {
			problems[k] = "not something chat takes: it takes message, canvas and file"
		}
	}
	text, ok := body["message"].(string)
	if body["message"] != nil && !ok {
		problems["message"] = "the words to send, as a string"
	}
	canvas, _ := body["canvas"].(string) // the tab to build on; absent is Home
	file, ok := body["file"].(string)
	if body["file"] != nil && !ok {
		problems["file"] = "the id of a file record, as a string"
	}
	if file != "" {
		if _, err := s.app.Store.Get(records.FileType, file); err != nil {
			problems["file"] = "there is no file " + file + "; add it first with POST /api/file/upload and send the id it answers with"
		}
	}
	if strings.TrimSpace(text) == "" && file == "" && len(problems) == 0 {
		problems["message"] = "empty: say something, or attach a file"
	}
	if len(problems) > 0 {
		web.WriteError(w, &schema.ValidationError{Problems: problems})
		return
	}
	rec, err := s.ChatFor(r).SendFile(r.Context(), canvas, text, file)
	if rec == nil {
		web.WriteError(w, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"reply": rec, "ok": err == nil})
}
