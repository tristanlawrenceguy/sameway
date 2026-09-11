package server_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// canvasWithACalendar puts a calendar on the canvas the way the assistant
// would, at the small size it would use beside a conversation.
func canvasWithACalendar(t *testing.T) (http.Handler, string) {
	t.Helper()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "calendar", "props": map[string]any{
			"month": "2026-09", "today": "2026-09-11", "detail": "brief", "caption": "September 2026",
			"events": []any{map[string]any{"date": "2026-09-14", "label": "Rent due"}},
		}}),
		{Text: "Put your calendar alongside."},
	}}, nil
	get(t, h, "/") // opening the canvas is what puts a conversation on it
	postForm(t, h, "/chat", url.Values{"message": {"show my calendar"}})
	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	for _, b := range blocks.Records {
		if b.Fields["component"] == "calendar" {
			return h, b.ID
		}
	}
	t.Fatalf("no calendar block was added: %v", blocks.Records)
	return nil, ""
}

// TestBlockCanBePoppedOut: every block on the canvas offers a way to take
// over the middle of the page, at its own URL, so a person can bookmark it
// and an agent can ask for exactly that one thing.
func TestBlockCanBePoppedOut(t *testing.T) {
	h, id := canvasWithACalendar(t)
	canvas := parse(t, get(t, h, "/"))

	links := canvas.WithAttr("href", "/canvas/"+id)
	if len(links) != 1 {
		t.Fatalf("expected one way to expand the block, got %d", len(links))
	}
	// A bare "Expand" is uselessly ambiguous when a page has several blocks,
	// so the accessible name has to say what is being expanded.
	if name := canvas.AccessibleName(links[0]); name != "Expand calendar" {
		t.Errorf("expand link reads as %q", name)
	}

	rec := get(t, h, "/canvas/"+id)
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)
	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("the expanded view should have exactly one h1, got %d", n)
	}
	// The page is named after the block, not after the software.
	if got := htmltest.Text(doc.Elements("h1")[0]); got != "September 2026" {
		t.Errorf("expanded page heading = %q", got)
	}
	// A view that replaces the canvas has to say how to get back.
	if len(doc.WithAttr("href", "/")) == 0 {
		t.Errorf("the expanded view needs a way back to the canvas")
	}
	assertAllComponentsKnown(t, doc)
}

// TestExpandingAsksAComponentForItsFullestForm: the block was added at a
// glanceable size, and the same block expanded is the whole month with room
// for actions. Nothing is copied or moved to do it.
func TestExpandingAsksAComponentForItsFullestForm(t *testing.T) {
	h, id := canvasWithACalendar(t)
	if len(parse(t, get(t, h, "/")).WithAttr("data-detail", "brief")) != 1 {
		t.Errorf("on the canvas the calendar should still be the size it was asked for")
	}
	doc := parse(t, get(t, h, "/canvas/"+id))
	if len(doc.WithAttr("data-detail", "page")) != 1 {
		t.Errorf("expanded, the calendar should be at page size")
	}
	// The block is still on the canvas, untouched.
	back := parse(t, get(t, h, "/"))
	if len(back.WithAttr("data-block-id", id)) != 1 {
		t.Errorf("expanding must not move or remove the block")
	}
	if len(back.WithAttr("data-detail", "page")) != 0 {
		t.Errorf("expanding must not change the block's stored size")
	}
}

// TestExpandingTheConversationKeepsItLive: the chat is a block like any
// other, so it expands too, and it has to bring the actual conversation
// with it rather than an empty shell.
func TestExpandingTheConversationKeepsItLive(t *testing.T) {
	h, _ := canvasWithACalendar(t)
	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	var chatID string
	for _, b := range blocks.Records {
		if b.Fields["component"] == "chat" {
			chatID = b.ID
		}
	}
	if chatID == "" {
		t.Fatal("no conversation block on the canvas")
	}
	doc := parse(t, get(t, h, "/canvas/"+chatID))
	if len(doc.WithAttr("data-component", "message")) == 0 {
		t.Errorf("the expanded conversation should carry its messages")
	}
	if len(doc.Elements("form")) == 0 {
		t.Errorf("the expanded conversation should still be able to send")
	}
}

// TestExpandingSomethingThatIsNotThere is the plain 404, so a stale link
// from a bookmark or an agent says so instead of showing an empty page.
func TestExpandingSomethingThatIsNotThere(t *testing.T) {
	h, _ := canvasWithACalendar(t)
	wantStatus(t, get(t, h, "/canvas/nope"), http.StatusNotFound)
}
