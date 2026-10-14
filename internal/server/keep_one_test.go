package server_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestAListOffersToKeepOneOfTwins: tasks called the same and due the same
// day are offered, under the list's count, as one to keep; the press makes
// them one, logged, and Undo puts them back. The same errand on another day
// is not offered, and a list with none says nothing. If they are really
// two, Rename one opens the oldest to be named apart.
func TestAListOffersToKeepOneOfTwins(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if page := get(t, h, "/t/task").Body.String(); strings.Contains(page, "Keep the newest") {
		t.Fatal("a list with nothing alike offers nothing")
	}
	first, _ := a.Store.Create("task", map[string]any{"title": "Buy coffee", "due": "2026-10-16T00:00:00Z"})
	a.Store.Create("task", map[string]any{"title": "Buy coffee", "due": "2026-10-16T08:00:00Z"})
	a.Store.Create("task", map[string]any{"title": "Buy coffee", "due": "2026-10-23T00:00:00Z"})

	page := get(t, h, "/t/task").Body.String()
	if !strings.Contains(page, "2 tasks called Buy coffee, and nothing tells them apart.") {
		t.Fatalf("the two due the same day are said, not the third:\n%s", truncate(page))
	}
	form := regexp.MustCompile(`action="(/t/task/[a-z0-9]+/keep-one)"`).FindStringSubmatch(page)
	if form == nil || !strings.Contains(page, "Keep the newest") {
		t.Fatalf("one press keeps the newest:\n%s", truncate(page))
	}
	if !strings.Contains(page, `href="/t/task/`+first.ID+`#edit"`) || !strings.Contains(page, "Rename one") {
		t.Errorf("if they are two, Rename one opens the oldest to be named apart:\n%s", truncate(page))
	}
	// On its own beside a button, a link takes the button look and its
	// 44px target, or a thumb misses it on a phone (2.5.5).
	if !regexp.MustCompile(`class="sw-link sw-link--button[^"]*"[^>]*href="/t/task/` + first.ID + `#edit"`).MatchString(page) {
		t.Errorf("Rename one is the size of a target:\n%s", truncate(page))
	}
	postForm(t, h, form[1], nil)
	if tasks, _ := a.Store.List("task", store.ListOptions{}); len(tasks) != 2 {
		t.Errorf("the twins are one, the third stays: %d tasks", len(tasks))
	}
	if strings.Contains(get(t, h, "/t/task").Body.String(), "Keep the newest") {
		t.Error("once they are one, nothing is offered")
	}
	postForm(t, h, "/activity/"+newestEntry(t, a.Store).ID+"/undo", nil)
	if tasks, _ := a.Store.List("task", store.ListOptions{}); len(tasks) != 3 {
		t.Errorf("Undo puts the other back: %d tasks", len(tasks))
	}
}
