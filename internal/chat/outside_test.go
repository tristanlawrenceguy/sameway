package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// outside stands in for a program that runs the tools itself: while
// answering, it makes a note through the store and logs it, the way the
// MCP server would in another process.
type outside struct{ svc *chat.Service }

func (o *outside) Name() string       { return "outside" }
func (o *outside) ToolsOutside() bool { return true }
func (o *outside) Complete(_ context.Context, _ llm.Request) (*llm.Response, error) {
	rec, _ := o.svc.Store.Create("note", map[string]any{"title": "Made elsewhere"})
	chat.Record(o.svc.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: rec.ID, Detail: "Made elsewhere"})
	return &llm.Response{Text: "I made the note at /t/note/" + rec.ID + "."}, nil
}

// A reply from a provider that ran its tools elsewhere still carries the
// receipt: what the log says the assistant did during the turn, each
// entry undoable, and nothing from before the turn.
func TestAReplyFromOutsideToolsCarriesTheReceiptFromTheLog(t *testing.T) {
	svc := newFullService(t)
	chat.Record(svc.Store, "assistant", chat.Change{Action: "created", Component: "note", Detail: "earlier"})
	svc.Provider = &outside{svc: svc}
	reply, err := svc.Send(context.Background(), "make a note")
	if err != nil {
		t.Fatal(err)
	}
	changes, _ := reply.Fields["changes"].([]any)
	if len(changes) != 1 {
		t.Fatalf("the receipt should carry the one change made during the turn, got %v", reply.Fields["changes"])
	}
	c, _ := changes[0].(map[string]any)
	href, _ := c["href"].(string)
	if c["action"] != "created" || c["component"] != "note" || !strings.HasPrefix(href, "/t/note/") || c["activity"] == "" {
		t.Errorf("the change leads to the note and can be undone: %v", c)
	}
}

// While the tools run in another program, the changes they make are told
// as they land in the log, each as a step and a change, before the turn
// is done, so a page can show that turn as it happens too.
func TestChangesMadeOutsideAreToldAsTheyLand(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &outside{svc: svc}
	var events []string
	_, err := svc.SendLive(context.Background(), "", "make a note", "", func(e chat.Event) {
		events = append(events, e.Kind+":"+e.Label)
	})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(events, " ")
	if !strings.Contains(got, "tool:Created a note change: ") || !strings.HasSuffix(got, "done:") {
		t.Errorf("the change is told as a step and a change before done, got %v", events)
	}
}
