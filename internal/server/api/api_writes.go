package api

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Writes through the API are changes like any other: logged with what
// each record was before, so each can be undone. Whatever calls the API
// is a program, a script or an agent, so its changes are logged as an
// agent's, by the name it gives (see Agent), and a block it places is
// marked as that agent's too: a record and a block written by the same
// caller say the same who. The activity log itself is not written here
// at all.

func (s *Service) apiCreate(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) || !s.knownType(w, r) {
		return
	}
	if list, err := batchRead(r); err != nil || list != nil { // api_batch.go: several at once
		if err != nil {
			web.WriteError(w, err)
			return
		}
		writes := make([]records.Writing, len(list))
		for i, f := range list {
			writes[i] = records.Writing{Fields: f}
		}
		s.batchWrite(w, r, "created", writes)
		return
	}
	fields, err := web.ReadBody(r)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	shows, refused := s.blockWrite(w, r, fields, nil)
	if refused {
		return
	}
	agent := Agent(r)
	if r.PathValue("type") == records.BlockType {
		byAgent(fields, agent, true)
		// A tab no one has opened yet is still empty: give it its
		// conversation first, as opening it would, or the first block an
		// agent adds leaves it with none.
		if c, _ := fields["canvas"].(string); s.app.Chat.HasCanvas(c) {
			s.SeedChat(c)
		}
	}
	rec, notes, entry, err := records.WriteNoted(s.app.Store, agent.As(), "created", r.PathValue("type"), "", fields)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+rec.Type+"/"+rec.ID)
	web.WriteJSON(w, http.StatusCreated, shownRecord{s.titled(rec), shows, s.layoutOf(r, rec), notes, undoOf(entry)})
}

func (s *Service) apiUpdate(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) || !s.knownType(w, r) {
		return
	}
	fields, err := web.ReadBody(r)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	was, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeErr(w, r, err) // api_refusals.go
		return
	}
	if s.staleFor(w, r, was) { // below
		return
	}
	shows, refused := s.blockWrite(w, r, fields, was)
	if refused {
		return
	}
	agent := Agent(r)
	if r.PathValue("type") == records.BlockType {
		byAgent(fields, agent, false)
	}
	rec, notes, entry, err := records.WriteNoted(s.app.Store, agent.As(), "updated", r.PathValue("type"), r.PathValue("id"), fields)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, shownRecord{s.titled(rec), shows, s.layoutOf(r, rec), notes, undoOf(entry)})
}

func (s *Service) apiDelete(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) || !s.knownType(w, r) {
		return
	}
	if was, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id")); err == nil && s.staleFor(w, r, was) {
		return
	}
	// Logged with everything it had, so it can be put back.
	gone, entry, err := records.WriteAs(s.app.Store, Agent(r).As(), "deleted", r.PathValue("type"), r.PathValue("id"), nil)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := map[string]any{"deleted": r.PathValue("id")}
	if u := undoOf(entry); u != nil {
		out["undo"] = u // api_undo_said.go: it can be put back
	}
	if layout := s.layoutOf(r, gone); layout != "" {
		out["layout"] = layout
	}
	web.WriteJSON(w, http.StatusOK, out)
}

// keptLog refuses a write to the activity log itself: it is what makes
// every other change reversible, so it is kept by Sameway, read by anyone,
// and changed by nobody. A change is taken back by undoing it.
func (s *Service) keptLog(w http.ResponseWriter, r *http.Request) bool {
	if r.PathValue("type") != records.ActivityType {
		return false
	}
	w.Header().Set("Allow", "GET") // 405 says what the path does take (RFC 9110, 15.5.6)
	web.WriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": web.APIError{Code: "kept",
		Message: records.ErrKeptLog.Error()}})
	return true
}

// Agent is who is calling the API: the name its X-Sameway-Agent header
// gives, or nobody in particular, "An agent". The User-Agent is not a
// name: it names the library that sent the request (Go-http-client,
// python-requests), which reached people as "Added by Go-http-client".
func Agent(r *http.Request) records.Agent {
	// An agent with a key is who its key says, whatever it calls itself.
	if v := records.VisitorOf(r.Context()); v.Agent {
		return records.Agent{Name: v.Name, Through: records.ThroughAPI}
	}
	return records.Agent{Name: records.AgentName(r.Header.Get("X-Sameway-Agent")), Through: records.ThroughAPI}
}

// byAgent marks a block's fields as the agent's: who changed it last,
// and on a new one who added it. What the body says of these is not
// taken: who made a change is the log's to say, not the caller's.
func byAgent(fields map[string]any, a records.Agent, created bool) {
	fields["actor"], fields["agent"] = records.ActorAgent, a.Name
	delete(fields, "created_by")
	if created {
		fields["created_by"] = records.ActorAgent
	}
}

// staleFor refuses an agent's change made from an older version than the
// record's, when it says which one it read with If-Match, and answers
// with the record as it is, to change again.
func (s *Service) staleFor(w http.ResponseWriter, r *http.Request, rec *store.Record) bool {
	want := strings.TrimSpace(r.Header.Get("If-Match"))
	if want == "" || want == "*" || records.SameVersion(rec, want) {
		return false
	}
	web.WriteJSON(w, http.StatusPreconditionFailed, map[string]any{
		"error":   web.APIError{Code: "stale", Message: "it has changed since the version you read (If-Match " + want + "): nothing was written. current is the record as it is now; make your change to it and send it again with If-Match set to its updated_at"},
		"current": s.titled(rec),
	})
	return true
}
