package server

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Coming back after a while, a person is told what the others changed
// meanwhile, each with its Undo, at the top of the first page they open,
// until they say they have seen it. Each computer notes this for the
// people who use it (Store.Meta): when each was last about, and where
// their catching up starts.

const away = 30 * time.Minute

// whoKey names the one asking, for these notes: their login, or the owner
// when the tailnet has not said who that is.
func (s *Server) whoKey(r *http.Request) string {
	if login, _ := s.whoAsks(r); login != "" {
		return login
	}
	return "owner"
}

// cameBack notes that someone is about. Back after being away, what the
// others did since they left is theirs to catch up on.
func (s *Server) cameBack(key string) {
	now := time.Now().UTC()
	s.present.mu.Lock()
	if s.present.last == nil {
		s.present.last = map[string]time.Time{}
	}
	last, known := s.present.last[key]
	if !known {
		last, _ = time.Parse(time.RFC3339Nano, s.app.Store.Meta("last:"+key))
	}
	s.present.last[key] = now
	s.present.mu.Unlock()
	if !last.IsZero() && now.Sub(last) > away && s.app.Store.Meta("since:"+key) == "" {
		s.app.Store.SetMeta("since:"+key, last.Format(time.RFC3339Nano))
	}
	// Written down now and then, so a restart still knows.
	if now.Sub(last) > time.Minute || !known {
		s.app.Store.SetMeta("last:"+key, now.Format(time.RFC3339Nano))
	}
}

// sinceNotice is what the others changed while the one asking was away,
// or nothing.
func (s *Server) sinceNotice(r *http.Request) template.HTML {
	key := s.whoKey(r)
	since, err := time.Parse(time.RFC3339Nano, s.app.Store.Meta("since:"+key))
	if err != nil {
		return ""
	}
	entries, err := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 60})
	if err != nil {
		return ""
	}
	from := r.URL.Path
	var items []string
	for _, e := range entries {
		if !e.CreatedAt.After(since) || len(items) >= 8 {
			continue
		}
		if !s.byOther(e, key) {
			continue
		}
		items = append(items, `<li>`+string(s.event(e, from, 0, true))+`</li>`)
	}
	if len(items) == 0 {
		return ""
	}
	return template.HTML(`<section class="sw-panel sw-stack" data-component="since" aria-labelledby="since-h"><h2 id="since-h">Since you were last here</h2><ul class="sw-plain sw-stack">` + strings.Join(items, "") +
		`</ul><form method="post" action="/since/seen"><input type="hidden" name="from" value="` + template.HTMLEscapeString(from) + `"><button type="submit" class="sw-button sw-button--secondary sw-pressable">Got it</button></form></section>`)
}

// sinceSeen is the person saying they have caught up.
func (s *Server) sinceSeen(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.app.Store.SetMeta("since:"+s.whoKey(r), "")
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

// byOther says whether a person other than the one asking made a change:
// someone by their login, or the owner of this computer's copy (written
// with no login when the tailnet had not said who they are).
func (s *Server) byOther(e *store.Record, key string) bool {
	if e.Fields["actor"] != "human" {
		return false
	}
	login, _ := e.Fields["by_login"].(string)
	by, _ := e.Fields["by"].(string)
	if login == "" && by == "" {
		login = s.app.Chat.Owner.Login
		if login == "" {
			login = "owner"
		}
	}
	return login != "" && !strings.EqualFold(login, key) || login == "" && by != ""
}
