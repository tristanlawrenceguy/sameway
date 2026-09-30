package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What is the owner's is read by nobody else, on any page or route: two
// leaks were found by hand in two days (search showing a viewer the owner's
// conversation titles; a published page showing the internet the owner's
// log), each on a route nobody had thought to look at. This looks at all
// of them. The workspace holds marked words in each place that is the
// owner's alone, and every GET route, for every kind and a real record of
// it, is read as someone let in to look and as anyone on the internet; no
// marked word may come back.
func TestNothingOfTheOwnersIsReadByOthers(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)

	// The owner's alone, marked: a conversation and what was said in it,
	// a question the assistant asked, and a person the owner let in.
	convo, _ := a.Store.Create(chat.ConversationType, map[string]any{"title": "OWNERCONVO lawyer questions"})
	a.Store.Create(chat.MessageType, map[string]any{"role": "user", "content": "OWNERSAID keep this between us", "conversation": convo.ID})
	a.Store.Create(chat.ProposalType, map[string]any{"summary": "OWNERASKED delete the diary?", "action": map[string]any{"tool": "remove_component"}, "state": "pending"})
	a.Store.Create("person", map[string]any{"name": "Hana", "email": "hana-private@example.com", "access": "edit"})
	// The log, which names who changed what: the internet's never, and a
	// viewer's only as the pages let people see changes (not this entry,
	// which is about the owner's conversation).
	chat.Record(a.Store, "human", chat.Change{Action: "cleared", Component: "conversation", ID: convo.ID, Detail: "OWNERLOG private plans"})
	// The owner was away, so their "since you were last here" holds the
	// log's news: it must stay theirs.
	a.Store.SetMeta("last:owner", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano))
	get(t, h, "/")
	chat.Record(a.Store, "human", chat.Change{Action: "added", Component: "note", Detail: "OWNERNEWS Hana's plans", By: "Hana", ByLogin: "hana@example.com"})
	note, _ := a.Store.Create("note", map[string]any{"title": "Sourdough", "body": "Flour, water and salt."})
	a.Workspace.Config.Publish.Types = "note"
	a.Workspace.Config.Publish.Tabs = "Home"
	pub := srv.Public(nil)
	viewer := chat.Visitor{Name: "Vi", Login: "vi@example.com", Access: chat.View}

	ids := map[string]string{}
	for _, typ := range a.Types.Types {
		if recs, err := a.Store.List(typ.Name, store.ListOptions{}); err == nil && len(recs) > 0 {
			ids[typ.Name] = recs[0].ID
		}
	}
	ids["note"] = note.ID

	var paths []string
	files, _ := filepath.Glob("*.go")
	pattern := regexp.MustCompile(`HandleFunc\("GET (/[^"]*)"`)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, _ := os.ReadFile(f)
		for _, m := range pattern.FindAllStringSubmatch(string(src), -1) {
			route := strings.ReplaceAll(m[1], "{$}", "")
			if strings.Contains(route, "stream") || strings.HasSuffix(route, "/live") || route == "/events" || route == "/api/changes" {
				continue // streams that wait; their first answer is read in their own tests
			}
			if !strings.Contains(route, "{type}") && !strings.Contains(route, "{file}") {
				paths = append(paths, fill(route, "", ids))
				continue
			}
			for typ := range ids {
				p := strings.ReplaceAll(route, "{type}", typ)
				p = strings.ReplaceAll(p, "{file}", typ+".csv")
				paths = append(paths, fill(p, typ, ids))
			}
		}
	}
	paths = append(paths, "/search?q=lawyer", "/api/search?q=lawyer", "/api/search?q=between", "/search?q=diary", "/api/search?q=private",
		"/export/conversation.csv", "/export/message.csv", "/export/activity.csv", "/export/proposal.csv", "/export/person.vcf")

	marks := regexp.MustCompile(`OWNER[A-Z]+|hana-private@example\.com`)
	for _, p := range paths {
		if res := as(t, h, viewer, http.MethodGet, p, "", ""); res.Code < 400 {
			if m := marks.FindString(res.Body.String()); m != "" && !viewerMay(p, m) {
				t.Errorf("someone let in to look reads %s at %s", m, p)
			}
		}
		if res := public(t, pub, http.MethodGet, p, ""); res.Code < 400 {
			if m := marks.FindString(res.Body.String()); m != "" {
				t.Errorf("anyone on the internet reads %s at %s", m, p)
			}
		}
	}
}

// viewerMay is what someone let in may read of these: the people let in
// are shown to each other, email included, wherever people are found, and
// see what the others changed, as the pages of what changed say it.
func viewerMay(path, mark string) bool {
	return mark == "hana-private@example.com" || mark == "OWNERNEWS"
}

// fill puts a real id in a route's {id}, of its kind when it has one.
func fill(route, typ string, ids map[string]string) string {
	id := ids[typ]
	if id == "" {
		id = ids["note"]
	}
	return regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(route, id)
}
