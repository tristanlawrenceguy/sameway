package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The states a page takes that the assistant's side shapes, for the crawl
// that reads every page (every_page_reads_test.go): CI skips the browser
// suites for a change to internal/chat, which decides what waits on Today,
// what a reply's receipt says and offers, the question a proposal puts,
// and what a tab offers to tidy. Each is here at its edge: more than one
// change under a reply, a long Today, a tab with a block that cannot be
// shown, a tab with a long name.

// seedEveryState puts the workspace seedLikeAPerson made into those states
// and returns the pages only they reach. /design, the style guide, is
// every component's examples side by side, names repeated on purpose; the
// component suites read it.
func seedEveryState(t *testing.T, a *app.App, h http.Handler) []string {
	t.Helper()
	for _, title := range []string{
		"Ring the council about the parking permit for the new flat before the end of the month and ask about the visitor passes",
		strings.Repeat("Supercalifragilistic", 3),
	} {
		postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": title})
	}
	// A long Today: late tasks enough to start with three and fold the rest.
	for i := 1; i <= 7; i++ {
		due := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": fmt.Sprintf("Late errand %d", i), "due": due})
	}
	var tab struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/canvas", map[string]any{"name": "Everything about the house move in March and boxes"}), &tab)
	postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "canvas": tab.ID, "props": map[string]any{"type": "task", "label": "Before the move"}})
	// A block the page cannot show, written as an older version did: the
	// tab offers to tidy it.
	if _, err := a.Store.Create("block", map[string]any{"component": "collection", "props": map[string]any{"type": "gone", "label": "Old list"}, "position": 9, "canvas": tab.ID, "created_by": "agent"}); err != nil {
		t.Fatal(err)
	}
	// A question put to the person, then the newest reply with several
	// changes, which is the one that offers Undo all.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("propose_change", map[string]any{"summary": "Take away the old list on the move tab?", "tool": "remove_component", "id": "will-be-fixed"}),
		{Text: "Asked."},
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Boxes to buy"}}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Movers to call"}}),
		{Text: "Made two notes for the move."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"tidy the move tab"}, "from": {"/chat"}})
	postForm(t, h, "/chat", url.Values{"message": {"two notes for the move"}, "from": {"/chat"}})

	blocks, _ := a.Store.List("block", store.ListOptions{})
	notes, _ := a.Store.List("note", store.ListOptions{})
	pages := []string{"/c/" + tab.ID, "/apps", "/bring", "/calendars", "/mail", "/review", "/share", "/when", "/workspaces/new", "/t/task/import"}
	for _, b := range blocks { // a block's own page, of one that can be shown
		if props, _ := b.Fields["props"].(map[string]any); b.Fields["component"] == "collection" && props["type"] == "task" {
			pages = append(pages, "/canvas/"+b.ID)
			break
		}
	}
	if len(notes) > 0 {
		pages = append(pages, "/t/note/"+notes[0].ID+"/whole")
	}
	return pages
}
