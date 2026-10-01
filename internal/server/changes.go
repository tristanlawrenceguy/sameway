package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person sees a change arrive: the page follows along. An agent had
// only asking again and again, or reading the log, which is the owner's
// alone. GET /api/changes is the same news for an agent: what changed
// since the cursor it last had, oldest first, and ?wait= holds the
// question open until something does, so it need not ask blindly.
// Everyone reads the changes to what they may read; only the owner reads
// who made them and how to take them back, as only they read the log.

const changesMax = 200

func (s *Server) apiChanges(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := time.Time{}
	if c := q.Get("since"); c != "" {
		t, err := time.Parse(time.RFC3339Nano, c)
		if err != nil {
			writeError(w, &badRequest{"since is the cursor a previous answer gave, such as " + time.Now().UTC().Format(time.RFC3339Nano)})
			return
		}
		since = t
	}
	limit := 100
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = min(n, changesMax)
	}
	wait, _ := strconv.Atoi(q.Get("wait"))
	deadline := time.Now().Add(time.Duration(min(max(wait, 0), 60)) * time.Second)
	owner := chat.VisitorOf(r.Context()).Owner()
	seen := -1
	for {
		// Counting the log is cheap; reading it only when it grew is not
		// a scan every moment of a wait.
		if n, _ := s.app.Store.Count(chat.ActivityType); n != seen || !time.Now().Before(deadline) {
			seen = n
			out, cursor, more := s.changesSince(since, limit, owner, q.Get("since") == "")
			if len(out) > 0 || !time.Now().Before(deadline) {
				writeJSON(w, http.StatusOK, map[string]any{"changes": out, "cursor": cursor, "more": more,
					"next":      "/api/changes?since=" + cursor,
					"untrusted": "each change's title was written by its written_by: " + chat.Untrusted})
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// badRequest is a plain refusal in the API's own words.
type badRequest struct{ msg string }

func (b *badRequest) Error() string { return b.msg }

// changesSince lists the changes after since that the asker may read.
// With no cursor at all, the newest few, to start from.
func (s *Server) changesSince(since time.Time, limit int, owner, start bool) ([]map[string]any, string, bool) {
	// Newest first, back to the cursor, then turned round: a long log is
	// read only as far back as the cursor.
	n := 500
	if start {
		n = 20
	}
	all, _ := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: n})
	if !start && len(all) == n && all[n-1].CreatedAt.After(since) {
		all, _ = s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
	}
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	cursor := since.UTC().Format(time.RFC3339Nano)
	if since.IsZero() {
		cursor = time.Now().UTC().Format(time.RFC3339Nano)
	}
	var out []map[string]any
	writers := s.app.Chat.Writers()
	for _, e := range all {
		if !e.CreatedAt.After(since) {
			continue
		}
		if len(out) == limit {
			return out, cursor, true
		}
		cursor = e.CreatedAt.UTC().Format(time.RFC3339Nano)
		if c := s.change(e, owner, writers); c != nil {
			out = append(out, c)
		}
	}
	return out, cursor, false
}

// change is one entry as an agent reads it, or nil when it is not the
// asker's to read.
func (s *Server) change(e *store.Record, owner bool, writers *chat.Writers) map[string]any {
	target, _ := e.Fields["target"].(string)
	id, _ := e.Fields["target_id"].(string)
	t, isType := s.app.Types.Get(target)
	if !owner && (!isType || !t.Content() || t.Owners) {
		return nil // the canvas, settings and the owner's own records are the owner's news
	}
	c := map[string]any{"at": e.CreatedAt.UTC().Format(time.RFC3339Nano), "action": e.Fields["action"], "type": target}
	if id != "" {
		c["id"] = id
		if isType {
			if rec, err := s.app.Store.Get(target, id); err == nil {
				c["title"], c["version"], c["page"] = s.title(t, rec), chat.Version(rec), "/t/"+target+"/"+id
				c["written_by"] = writers.Of(target, rec).Words
			} else {
				c["gone"] = true
			}
		}
	}
	if owner {
		c["said"], c["entry"] = chat.Sentence(s.app.Store, e.Fields), e.ID
		if s.app.Chat.Undoable(e) {
			c["undo"] = "/activity/" + e.ID + "/undo"
		}
	}
	return c
}
