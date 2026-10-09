package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Suggestions wait on the writing's page, grouped as an editor would and
// shown as the writing reads, never as Markdown; fixes can be accepted all
// at once as one change, and each answer says how many are left. Someone
// who may only look sees none of them.
func TestSuggestionsWaitOnTheWritingAsItReads(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	body := "Jobs\n\nWe recieve seeds in April and teh liner in May, so at the meeting it was decided by everyone that the garden opens in June when the beds are ready and the paths are laid and the water is on."
	note, err := a.Store.Create("note", map[string]any{"title": "Garden", "body": body})
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "recieve", "replacement": "receive", "why": "Spelling.", "kind": "fix"},
		map[string]any{"passage": "teh", "replacement": "the", "why": "Typo.", "kind": "fix"},
		map[string]any{"passage": "Jobs", "replacement": "## Jobs", "why": "A heading for the list.", "kind": "style"},
		map[string]any{"passage": "it was decided by everyone that", "replacement": "we decided", "why": "Shorter.", "kind": "clarity"},
	}})
	if text, isErr := a.Chat.Call("suggest_edits", args); isErr {
		t.Fatal(text)
	}
	page := get(t, h, "/t/note/"+note.ID).Body.String()
	for _, want := range []string{"4 suggested changes", "<h3>Fixes</h3>", "<h3>Formatting</h3>", "<h3>Clarity</h3>", "Accept all 2 fixes", "Make it a heading",
		`Change starts: </span>it was decided by everyone that<span class="sw-visually-hidden">, change ends,</span></span>`,
		// Who suggested it is said in words, not by colour alone.
		`1 of 4</span>, suggested by the assistant</p>`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page should have %q", want)
		}
	}
	if strings.Contains(page, "Help with the writing") {
		t.Error("help is a part, off until there is a reason")
	}
	if asked := get(t, h, "/t/note/"+note.ID+"?show=writing-help").Body.String(); !strings.Contains(asked, "Check spelling and typos") {
		t.Error("?show=writing-help opens it")
	}
	if strings.Contains(page, "## Jobs") {
		t.Error("a heading is shown as a heading, never as Markdown")
	}
	viewer := as(t, h, records.Visitor{Name: "Ana", Login: "ana@example.com", Access: records.View}, http.MethodGet, "/t/note/"+note.ID, "", "").Body.String()
	if strings.Contains(viewer, "suggested change") || strings.Contains(viewer, "Help with the writing") {
		t.Error("someone who may only look sees no suggestions and no help")
	}

	res := postForm(t, h, "/suggestions/accept-all", url.Values{"about": {"note/" + note.ID}, "kind": {"fix"}})
	if res.Code >= 400 {
		t.Fatalf("accept all: %d", res.Code)
	}
	now, _ := a.Store.Get("note", note.ID)
	if b := now.Fields["body"].(string); !strings.Contains(b, "We receive seeds in April and the liner") || !strings.Contains(b, "decided by everyone") {
		t.Errorf("both fixes went in, nothing else: %q", b)
	}
	left := chat.Suggestions(a.Store, "note", note.ID)
	if len(left) != 2 {
		t.Fatalf("two still wait: %d", len(left))
	}
	req := httptest.NewRequest(http.MethodPost, "/suggestions/"+left[0].ID+"/decline", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "1 left.") {
		t.Errorf("an answer says how many are left: %s", rec.Body.String())
	}
}

// Who suggested a change is said in words beside its coloured edge: an
// agent by the name the activity log has for it.
func TestASuggestionSaysWhoSuggestedIt(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	note, err := a.Store.Create("note", map[string]any{"title": "Garden", "body": "The pond liner comes in May."})
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]any{"type": "note", "id": note.ID, "edits": []any{
		map[string]any{"passage": "May", "replacement": "April", "why": "The plan says April.", "kind": "clarity"},
	}})
	agent := a.Chat.ByAgent(records.Agent{Name: "Claude Code", Through: records.ThroughMCP})
	if text, isErr := agent.Call("suggest_edits", args); isErr {
		t.Fatal(text)
	}
	page := get(t, h, "/t/note/"+note.ID).Body.String()
	if !strings.Contains(page, `data-actor="agent"`) || !strings.Contains(page, "1 of 1</span>, suggested by Claude Code, an agent</p>") {
		t.Error("an agent's suggestion names the agent in words")
	}
}
