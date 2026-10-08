package server_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// other opens the same workspace a second time, as `sameway mcp` does
// beside the server when chat goes through Claude Code.
func other(t *testing.T, a *app.App) *app.App {
	t.Helper()
	b, err := app.Load(a.Workspace.Dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	return b
}

// A type and a field made by another process on the same workspace are on
// the running server's pages at once, with no restart: the list page, a
// block that shows the type, describe. Before, /t/recipe stayed not found
// and the block said there was no such type until the server restarted,
// right after the assistant said it was done. Records the other process
// wrote were always there: both open the same data.db.
func TestATypeMadeByAnotherProcessIsServedWithoutARestart(t *testing.T) {
	a, h := newApp(t)
	b := other(t, a)
	if _, err := b.AddType(&schema.Type{Name: "recipe", Description: "A recipe.", Fields: []schema.Field{{Name: "title", Type: "string"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AddField("note", schema.Field{Name: "mood", Type: "string", Description: "How it felt."}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Store.Create("recipe", map[string]any{"title": "Lentil soup"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Store.Create(records.BlockType, b.Chat.BlockFields(map[string]any{"component": "collection", "props": map[string]any{"type": "recipe"}})); err != nil {
		t.Fatal(err)
	}

	get := func(path string) (int, string) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w.Code, w.Body.String()
	}
	if code, body := get("/"); !strings.Contains(body, "Lentil soup") || strings.Contains(body, "no content type") {
		t.Errorf("the block showing recipes lists them (%d):\n%s", code, body)
	}
	if code, body := get("/t/recipe"); code != http.StatusOK || !strings.Contains(body, "Lentil soup") {
		t.Errorf("/t/recipe is served with its record, got %d", code)
	}
	if _, body := get("/api/describe/types/note"); !strings.Contains(body, `"mood"`) {
		t.Errorf("the field the other process added is described:\n%s", body)
	}
	if _, err := a.Store.Create("note", map[string]any{"title": "Walk", "mood": "calm"}); err != nil {
		t.Errorf("the server writes the new field too: %v", err)
	}
}

// Open pages hear that the types changed, as they hear of a change from
// another computer, so a block saying a type is missing is drawn again.
// The server looks every second (serve starts WatchSchema); a request
// that comes first takes the change itself.
func TestOpenPagesFollowATypeMadeElsewhere(t *testing.T) {
	a, h := newApp(t)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.WatchSchema(ctx, 20*time.Millisecond, h.(*server.Server).Changed)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	lines := make(chan string)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	if l := <-lines; l != "event: hello" {
		t.Fatalf("the stream opens with hello, got %q", l)
	}
	if _, err := other(t, a).AddType(&schema.Type{Name: "wine", Fields: []schema.Field{{Name: "title", Type: "string"}}}); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatal("the stream ended before saying anything changed")
			}
			if l == "event: changed" {
				if _, ok := a.Types.Get("wine"); !ok {
					t.Error("the page is told only once the server has the type")
				}
				return
			}
		case <-deadline:
			t.Fatal("no changed event within 5s of the type being made elsewhere")
		}
	}
}
