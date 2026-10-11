package chat_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// elsewhere is the MCP server as it really runs for a model that runs its
// tools itself: another process, with its own store on the same
// database. Nothing it writes is told to this process's listeners, so it
// waits, while still answering, for the change to be shown here.
type elsewhere struct {
	st    *store.Store
	shown chan struct{}
	live  bool
}

func (e *elsewhere) Name() string       { return "elsewhere" }
func (e *elsewhere) ToolsOutside() bool { return true }
func (e *elsewhere) Complete(_ context.Context, _ llm.Request) (*llm.Response, error) {
	rec, _ := e.st.Create("note", map[string]any{"title": "Made in another process"})
	records.Record(e.st, "assistant", records.Change{Action: "created", Component: "note", ID: rec.ID, Detail: "Made in another process"})
	select {
	case <-e.shown:
		e.live = true
	case <-time.After(5 * time.Second):
	}
	return &llm.Response{Text: "Made it."}, nil
}

// A change made in another process is shown while the turn still runs.
// It cannot be heard through records.Listen, which hears only this
// process's writes, so the turn reads the log as it goes (outside.go);
// a watch that only listened would show it only once the turn was done.
func TestChangesMadeInAnotherProcessAreShownAsTheyLand(t *testing.T) {
	t.Parallel()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(t.TempDir(), "sameway.db")
	here, err := store.Open(db, types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { here.Close() })
	there, err := store.Open(db, types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { there.Close() })
	reg, err := app.NewRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	other := &elsewhere{st: there, shown: make(chan struct{}, 1)}
	svc := &chat.Service{Book: &records.Book{Store: here}, Registry: reg, HistoryLimit: 10, Provider: other}
	_, err = svc.SendLive(context.Background(), "", "make a note", "", func(e chat.Event) {
		if e.Kind == "change" {
			select {
			case other.shown <- struct{}{}:
			default:
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !other.live {
		t.Error("the change made in another process was shown only once the turn was done, not as it landed")
	}
}
