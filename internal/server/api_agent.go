package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// What pages show that /api did not: a chat turn with a file, and the
// workspaces on this machine. What pages do is an agent's already, through
// the same forms (agents.go).

func (s *Server) agentRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/workspaces", s.apiWorkspaces)
}

// apiChat is one turn of the conversation for an agent: the message, the
// tab it is about, and a file already added (POST /api/file/upload), whose
// text goes to the assistant with the message as it does for a person.
func (s *Server) apiChat(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	if err != nil {
		writeError(w, err)
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
		if _, err := s.app.Store.Get(FileType, file); err != nil {
			problems["file"] = "there is no file " + file + "; add it first with POST /api/file/upload and send the id it answers with"
		}
	}
	if strings.TrimSpace(text) == "" && file == "" && len(problems) == 0 {
		problems["message"] = "empty: say something, or attach a file"
	}
	if len(problems) > 0 {
		writeError(w, &schema.ValidationError{Problems: problems})
		return
	}
	rec, err := s.chatFor(r).SendFile(r.Context(), canvas, text, file)
	if rec == nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reply": rec, "ok": err == nil})
}

// workspaceOut is a workspace as the workspaces page lists it.
type workspaceOut struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	URL     string `json:"url,omitempty"`
	Running bool   `json:"running"`
	This    bool   `json:"this,omitempty"`
}

func (s *Server) apiWorkspaces(w http.ResponseWriter, r *http.Request) {
	cur := s.app.Workspace
	this := workspaceOut{Name: cur.Config.Name, Dir: cur.Dir, Running: true, This: true}
	for _, k := range workspace.KnownWorkspaces() {
		if sameDir(k.Dir, cur.Dir) && k.Addr != "" {
			this.URL = "http://" + k.Addr + "/"
		}
	}
	list := []workspaceOut{this}
	for _, o := range s.others() {
		list = append(list, workspaceOut{Name: o.Name, Dir: o.Dir, URL: o.URL, Running: o.Running})
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": list})
}
