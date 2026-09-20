package server_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person undoes a change where they see it: under the newest reply, and
// on the activity page, and they stay where they were. What they deleted
// themselves can come back the same way.
func TestAPersonUndoesFromWhereTheyAre(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Plan"}}),
		{Text: "Added."},
	}}, nil
	get(t, h, "/")
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}, "from": {"/chat"}})

	cards := func() int {
		var blocks struct {
			Records []struct{ Fields map[string]any }
		}
		decode(t, get(t, h, "/api/block"), &blocks)
		n := 0
		for _, b := range blocks.Records {
			if b.Fields["component"] == "card" {
				n++
			}
		}
		return n
	}
	if cards() != 1 {
		t.Fatal("the assistant should have added a card")
	}

	// The receipt under the newest reply offers Undo, back to the chat.
	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `class="sw-message__undo"`) || !strings.Contains(page, `name="from" value="/chat"`) {
		t.Fatal("the receipt under the newest reply should offer Undo")
	}

	// The activity page offers it too; pressing it returns there.
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)"`)
	act := get(t, h, "/activity").Body.String()
	m := undo.FindStringSubmatch(act)
	if m == nil {
		t.Fatal("the activity page should offer Undo on the addition")
	}
	rec := postForm(t, h, m[1], url.Values{"from": {"/activity"}})
	wantStatus(t, rec, http.StatusSeeOther)
	if loc := rec.Header().Get("Location"); loc != "/activity" {
		t.Errorf("undo should return to the page it was pressed on, got %q", loc)
	}
	if cards() != 0 {
		t.Fatal("undoing the addition should remove the card")
	}
	act = get(t, h, "/activity").Body.String()
	if !logged(t, h, "You undid: Assistant added card Plan") || !strings.Contains(act, "undid:") {
		t.Error("the log should say the person undid the addition")
	}
	// The undo itself can be undone from the same page, and the old receipt
	// no longer offers a control for something that is gone.
	if !undo.MatchString(act) {
		t.Error("the undo entry should offer Undo in turn")
	}
	if strings.Contains(get(t, h, "/chat").Body.String(), `class="sw-message__undo"`) {
		t.Error("a receipt for something already gone should not offer Undo")
	}

	// A note the person deleted comes back the same way.
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Sunday."})
	wantStatus(t, created, http.StatusCreated)
	var note struct{ ID string }
	decode(t, created, &note)
	wantStatus(t, postForm(t, h, "/t/note/"+note.ID+"/delete", url.Values{}), http.StatusOK)
	act = get(t, h, "/activity").Body.String()
	if !logged(t, h, "You deleted note Water the plants") {
		t.Fatal("a deletion by the person should be in the log")
	}
	m = undo.FindStringSubmatch(act)
	wantStatus(t, postForm(t, h, m[1], url.Values{"from": {"/t/note"}}), http.StatusSeeOther)
	if rec := get(t, h, "/api/note/"+note.ID); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Sunday.") {
		t.Errorf("undoing the deletion should bring the note back whole, got %d %s", rec.Code, rec.Body.String())
	}
}

// logged says whether the activity log has an entry with that sentence.
func logged(t *testing.T, h http.Handler, summary string) bool {
	t.Helper()
	var log struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/activity"), &log)
	for _, r := range log.Records {
		if r.Fields["summary"] == summary {
			return true
		}
	}
	return false
}
