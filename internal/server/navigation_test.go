package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// Getting from here to the thing: a detail page says which listing it
// belongs to, an activity entry leads to what it changed while that thing
// exists, a receipt under a reply leads to what the assistant made, and a
// list of activities reads as sentences rather than a column of verbs.
func TestEverythingLeadsSomewhere(t *testing.T) {
	a, h := newApp(t)

	// A detail page carries the way back to its listing.
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants"}), &note)
	detail := get(t, h, "/t/note/"+note.ID).Body.String()
	if !strings.Contains(detail, `aria-label="You are here"`) || !strings.Contains(detail, `href="/t/note">Notes</a>`) {
		t.Errorf("the detail page crumb should link to the listing\n%s", detail)
	}

	// The assistant makes a note and a card; the receipt under its reply
	// links to both, and the activity log links to both.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Call the dentist"}}),
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Monday"}}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"a note and a card"}, "from": {"/"}})
	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	card := ""
	for _, b := range blocks.Records {
		if b.Fields["component"] == "card" {
			card = b.ID
		}
	}
	var notes struct{ Records []struct{ ID string } }
	decode(t, get(t, h, "/api/note"), &notes)
	if card == "" || len(notes.Records) != 2 {
		t.Fatalf("expected a card block and two notes, got card=%q notes=%d", card, len(notes.Records))
	}
	dentist := notes.Records[0].ID
	if notes.Records[0].ID == note.ID {
		dentist = notes.Records[1].ID
	}

	chat := get(t, h, "/chat").Body.String()
	for _, want := range []string{`href="/t/note/` + dentist + `"`, `href="/canvas/` + card + `"`} {
		if !strings.Contains(chat, want) {
			t.Errorf("the receipt under the reply should link to what was made, want %s", want)
		}
	}
	activity := get(t, h, "/activity").Body.String()
	for _, want := range []string{`<a class="sw-event__link" href="/t/note/` + dentist + `"`, `<a class="sw-event__link" href="/canvas/` + card + `"`} {
		if !strings.Contains(activity, want) {
			t.Errorf("the activity log should lead to what was changed, want %s", want)
		}
	}

	// Once the card is gone, its entry says what happened but leads nowhere.
	wantStatus(t, postForm(t, h, "/canvas/"+card+"/delete", url.Values{"from": {"/"}}), http.StatusSeeOther)
	activity = get(t, h, "/activity").Body.String()
	if strings.Contains(activity, `href="/canvas/`+card+`"`) {
		t.Error("an entry about a removed block should not link to a page that is gone")
	}

	// The generic listing of activities reads as sentences.
	listing := get(t, h, "/t/activity").Body.String()
	if !strings.Contains(listing, "Assistant created note Call the dentist") || !strings.Contains(listing, "You removed card Monday") {
		t.Errorf("activity listings should read as sentences, not verbs:\n%s", truncate(listing))
	}
}
