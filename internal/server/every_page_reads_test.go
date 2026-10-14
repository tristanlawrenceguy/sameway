package server_test

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Every page reads without a problem, in a workspace with real-looking
// records and the edges of what a person or a model may give it: a title
// as long as a sentence, one long word, a tab with a long name. CI runs
// the browser suites only for a change that can alter what a page is drawn
// from (ci.yml); a change to what the assistant is told, which can still
// change what a page says (what waits on Today, a template, a tab's name),
// is read here instead, on every run: no empty link or heading, no two
// controls of one name, headings in order, what look finds a screen
// reader user would be stuck on.
func TestEveryPageReadsWithoutAProblem(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLikeAPerson(t, a, h)         // machine_seed_test.go
	extra := seedEveryState(t, a, h) // every_state_seed_test.go
	paths := append(extra, "/", "/today", "/chat", "/templates", "/activity", "/calendar", "/help", "/workspaces", "/search?q=paint")
	for _, name := range a.Types.Names() {
		ty, _ := a.Types.Get(name)
		if !ty.Content() {
			continue
		}
		paths = append(paths, "/t/"+name)
		recs, _ := a.Store.List(name, store.ListOptions{})
		for _, r := range recs {
			paths = append(paths, "/t/"+name+"/"+r.ID)
		}
	}
	sort.Strings(paths)

	found := map[string][]string{}
	read := 0
	for _, p := range paths {
		res := get(t, h, p)
		if res.Code != http.StatusOK || !strings.Contains(res.Header().Get("Content-Type"), "html") {
			continue
		}
		read++
		o, err := look.Page(res.Body.String())
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		for _, why := range o.Problems {
			if strings.Contains(why, "there is no content type gone") {
				continue // the block seedEveryState set up wrong, said so on purpose
			}
			found[why] = append(found[why], p)
		}
	}
	// The states were reached, or the crawl read pages without them.
	for path, says := range map[string]string{
		"/chat":     "sw-message__undo-all",
		"/today":    "Start with these",
		extra[0]:    "Tidy this page",
		"/chat#ask": "Take away the old list on the move tab?",
	} {
		if !strings.Contains(get(t, h, strings.TrimSuffix(path, "#ask")).Body.String(), says) {
			t.Errorf("%s should show %q: the state the crawl is for was not reached", path, says)
		}
	}
	if read < 20 {
		t.Fatalf("only %d pages read; the crawl lost its pages", read)
	}
	var lines []string
	for why, where := range found {
		if len(where) > 4 {
			where = append(where[:4], fmt.Sprintf("and %d more", len(where)-4))
		}
		lines = append(lines, why+"\n      on "+strings.Join(where, ", "))
	}
	sort.Strings(lines)
	if len(lines) > 0 {
		t.Errorf("%d problems a screen reader user would be stuck on:\n  %s", len(lines), strings.Join(lines, "\n  "))
	}
}
