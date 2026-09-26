package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/peers"
)

// Who else is in the workspace just now, and where, so two people do not
// set about the same thing at once. Someone is here while they have a page
// open (its live connection says so every second) or moved about in the
// last half minute; people on the other computers that host the workspace
// are here by what those computers say at each exchange. The page says it
// only when someone else is: "Also here: Hana, on Shopping list".

const presentFor = 30 * time.Second

type presence struct {
	mu   sync.Mutex
	here map[string]seenAt    // by login, on this computer
	away map[string]seenAt    // by login, on the others
	last map[string]time.Time // when each was last about; see since.go
}

type seenAt struct {
	peers.Presence
	at time.Time
}

// seen notes the person making a request as here, on the page it is for.
func (s *Server) seen(r *http.Request, path string) {
	s.cameBack(s.whoKey(r))
	login, name := s.whoAsks(r)
	if login == "" {
		return
	}
	s.present.mu.Lock()
	defer s.present.mu.Unlock()
	if s.present.here == nil {
		s.present.here = map[string]seenAt{}
	}
	s.present.here[login] = seenAt{peers.Presence{Login: login, Name: name, Place: s.placeName(path)}, time.Now()}
}

// whoAsks is the login and name of who makes a request: a visitor, or the
// owner of this computer's copy, once the tailnet says who that is.
func (s *Server) whoAsks(r *http.Request) (string, string) {
	v := chat.VisitorOf(r.Context())
	if v.Login != "" {
		name := v.Who()
		if v.Owner() {
			name = s.app.Chat.Owner.Name
		}
		return strings.ToLower(v.Login), name
	}
	return strings.ToLower(s.app.Chat.Owner.Login), s.app.Chat.Owner.Name
}

// PresentHere is who is here on this computer just now, for the others.
func (s *Server) PresentHere() []peers.Presence {
	s.present.mu.Lock()
	defer s.present.mu.Unlock()
	var out []peers.Presence
	for _, p := range s.present.here {
		if time.Since(p.at) < presentFor {
			out = append(out, p.Presence)
		}
	}
	return out
}

// HearPresence takes who is on another computer just now.
func (s *Server) HearPresence(ps []peers.Presence) {
	s.present.mu.Lock()
	defer s.present.mu.Unlock()
	if s.present.away == nil {
		s.present.away = map[string]seenAt{}
	}
	for _, p := range ps {
		s.present.away[strings.ToLower(p.Login)] = seenAt{p, time.Now()}
	}
}

// presentFor says who else is here, for the one asking; empty when nobody.
func (s *Server) presentFor(r *http.Request) template.HTML {
	me, _ := s.whoAsks(r)
	s.present.mu.Lock()
	all := map[string]seenAt{}
	for _, m := range []map[string]seenAt{s.present.away, s.present.here} {
		for k, p := range m {
			if k != me && k != "" && time.Since(p.at) < presentFor {
				all[k] = p
			}
		}
	}
	s.present.mu.Unlock()
	if len(all) == 0 {
		return ""
	}
	var names []string
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	var parts []string
	for _, k := range names {
		p := all[k]
		name := p.Name
		if name == "" {
			name, _, _ = strings.Cut(p.Login, "@")
		}
		one := fmt.Sprintf(`<span class="sw-present__who" data-person="%d"><span class="sw-present__dot" aria-hidden="true"></span>%s`, chat.PersonColour(p.Login), template.HTMLEscapeString(name))
		if p.Place != "" {
			one += ", on " + template.HTMLEscapeString(p.Place)
		}
		parts = append(parts, one+`</span>`)
	}
	return template.HTML("Also here: " + strings.Join(parts, " "))
}

// placeName is what a page is, as a person would say where someone is.
func (s *Server) placeName(path string) string {
	switch {
	case path == "/" || path == "":
		return "Home"
	case strings.HasPrefix(path, "/c/"):
		if c, err := s.app.Store.Get(chat.CanvasType, strings.TrimPrefix(path, "/c/")); err == nil {
			if title, _ := c.Fields["title"].(string); title != "" {
				return title
			}
		}
	case strings.HasPrefix(path, "/t/"):
		if title := s.linkTitle(path); title != "" {
			return title
		}
		parts := strings.Split(strings.TrimPrefix(path, "/t/"), "/")
		if len(parts) == 1 && parts[0] != "" {
			p := plural(parts[0])
			return strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return ""
}

// refererPath is the page a request was made from, for the live connection
// every open page keeps.
func refererPath(r *http.Request) string {
	if u, err := url.Parse(r.Referer()); err == nil && u.Path != "" {
		return u.Path
	}
	return ""
}

// isPage says whether a request is for a page a person looks at, rather
// than the scripts, styles, streams and data behind one.
func isPage(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	for _, p := range []string{"/design/", "/api/", "/events", "/clock/", "/chat/live", "/favicon"} {
		if strings.HasPrefix(r.URL.Path, p) {
			return false
		}
	}
	return true
}
