package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person who cannot see the reply arrive hears what it says: the chat's
// status, which screen readers announce, carries the reply's first words.
// They are read out but not drawn, so the chip stays a few words long.
func TestTheStatusSaysWhatTheReplySays(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{Text: "Your garden list has three things on it."}}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"what is on my garden list?"}, "from": {"/chat"}})
	page := get(t, h, "/chat").Body.String()
	want := `<span class="sw-status__text">Assistant replied.</span><span class="sw-status__said sw-visually-hidden"> Your garden list has three things on it.</span>`
	if !strings.Contains(page, want) {
		t.Errorf("the chip says Assistant replied, and the reply's words are read out but not drawn; body: %s", truncate(page))
	}
}

// Answering one question sets the others aside, on the server as on the
// page, so none is left waiting where nobody can see it.
func TestAnsweringOneQuestionSetsTheOthersAside(t *testing.T) {
	a, h := newApp(t)
	one, _ := a.Store.Create(chat.ProposalType, map[string]any{"summary": "Remove the first?", "action": map[string]any{"tool": "remove_component", "id": "x"}, "state": "pending"})
	a.Store.Create(chat.ProposalType, map[string]any{"summary": "Remove the second?", "action": map[string]any{"tool": "remove_component", "id": "y"}, "state": "pending"})
	wantStatus(t, postForm(t, h, "/proposal/"+one.ID+"/dismiss", url.Values{"from": {"/chat"}}), http.StatusSeeOther)
	if waiting := a.Chat.Proposals(); len(waiting) != 0 {
		t.Errorf("no question is left waiting unseen, got %d", len(waiting))
	}
}

// The outcome of an action can take focus, and has an id a reopened
// field points at, so it is read when the page the person returns to opens.
func TestTheOutcomeCanTakeFocus(t *testing.T) {
	a, h := newApp(t)
	note, _ := a.Store.Create("note", map[string]any{"title": "Seeds"})
	r := postForm(t, h, "/t/note/"+note.ID+"/props", url.Values{"prop-title": {"Seeds and peas"}})
	if body := after(t, h, r).Body.String(); !strings.Contains(body, `id="outcome" tabindex="-1"`) {
		t.Errorf("the outcome takes focus; body: %s", truncate(body))
	}
}
