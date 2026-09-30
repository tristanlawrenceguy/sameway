package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Writes through the API are changes like any other: logged with what
// each record was before, so each can be undone. Whatever calls the API
// is a program, a script or an agent, so its changes are logged as an
// agent's, by the name it gives (see apiAgent), and a block it places is
// marked as that agent's too: a record and a block written by the same
// caller say the same who. The activity log itself is not written here
// at all.

func (s *Server) apiCreate(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) {
		return
	}
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	shows, refused := s.blockWrite(w, r, fields, nil)
	if refused {
		return
	}
	agent := apiAgent(r)
	if r.PathValue("type") == chat.BlockType {
		byAgent(fields, agent, true)
	}
	rec, _, err := chat.WriteAs(s.app.Store, agent.As(), "created", r.PathValue("type"), "", fields)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+rec.Type+"/"+rec.ID)
	writeJSON(w, http.StatusCreated, shownRecord{s.titled(rec), shows, s.layoutOf(r, rec)})
}

func (s *Server) apiUpdate(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) {
		return
	}
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	was, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if s.staleFor(w, r, was) { // see versions.go
		return
	}
	shows, refused := s.blockWrite(w, r, fields, was)
	if refused {
		return
	}
	agent := apiAgent(r)
	if r.PathValue("type") == chat.BlockType {
		byAgent(fields, agent, false)
	}
	rec, _, err := chat.WriteAs(s.app.Store, agent.As(), "updated", r.PathValue("type"), r.PathValue("id"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shownRecord{s.titled(rec), shows, s.layoutOf(r, rec)})
}

func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) {
		return
	}
	if was, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id")); err == nil && s.staleFor(w, r, was) {
		return
	}
	// Logged with everything it had, so it can be put back.
	gone, _, err := chat.WriteAs(s.app.Store, apiAgent(r).As(), "deleted", r.PathValue("type"), r.PathValue("id"), nil)
	if err != nil {
		writeError(w, err)
		return
	}
	out := map[string]any{"deleted": r.PathValue("id")}
	if layout := s.layoutOf(r, gone); layout != "" {
		out["layout"] = layout
	}
	writeJSON(w, http.StatusOK, out)
}

// keptLog refuses a write to the activity log itself: it is what makes
// every other change reversible, so it is kept by Sameway, read by anyone,
// and changed by nobody. A change is taken back by undoing it.
func (s *Server) keptLog(w http.ResponseWriter, r *http.Request) bool {
	if r.PathValue("type") != chat.ActivityType {
		return false
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": apiError{Code: "kept",
		Message: chat.ErrKeptLog.Error()}})
	return true
}

// apiAgent is who is calling the API: the name its X-Sameway-Agent header
// gives, or the product its User-Agent names (curl, python-requests),
// or nobody in particular, "An agent". A browser's User-Agent names no
// one, since every browser's starts Mozilla.
func apiAgent(r *http.Request) chat.Agent {
	// An agent with a key is who its key says, whatever it calls itself.
	if v := chat.VisitorOf(r.Context()); v.Agent {
		return chat.Agent{Name: v.Name, Through: chat.ThroughAPI}
	}
	name := r.Header.Get("X-Sameway-Agent")
	if name == "" {
		product, _, _ := strings.Cut(strings.TrimSpace(r.UserAgent()), "/")
		if product, _, _ = strings.Cut(product, " "); product != "Mozilla" {
			name = product
		}
	}
	return chat.Agent{Name: chat.AgentName(name), Through: chat.ThroughAPI}
}

// byAgent marks a block's fields as the agent's: who changed it last,
// and on a new one who added it. What the body says of these is not
// taken: who made a change is the log's to say, not the caller's.
func byAgent(fields map[string]any, a chat.Agent, created bool) {
	fields["actor"], fields["agent"] = chat.ActorAgent, a.Name
	delete(fields, "created_by")
	if created {
		fields["created_by"] = chat.ActorAgent
	}
}
