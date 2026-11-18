package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// No, with another time offered, was two messages to whoever asked: the
// answer, then an email. Now the owner's words go with the answer, in one
// message where they asked and on the request's page, put to the owner
// first word for word, as any words sent in their name are.
func TestAnAnswerCarriesTheOwnersWords(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	a.Workspace.Config.Publish.Book = "on"
	var told []string
	a.Chat.Channel = func(_ records.Visitor, text string) bool { told = append(told, text); return true }
	token, err := a.Chat.Booking("Pat", "pat@example.com", "About fitting the kitchen", thursdayAt(9))
	if err != nil {
		t.Fatal(err)
	}
	p := a.Chat.RequestByToken(token)
	words := "Thursday is full, sorry. Friday at 10 works, if that suits?"
	raw, _ := json.Marshal(map[string]any{"id": p.ID, "yes": false, "say": words})
	a.Chat.Call("answer_request", raw)
	var asked string
	for _, q := range a.Chat.Proposals() {
		if strings.Contains(q.Fields["summary"].(string), "and say this?") {
			asked = q.ID
			if q.Fields["yes"] != "Say no, with these words" || !strings.Contains(q.Fields["detail"].(string), words) {
				t.Errorf("the words are put to the owner first: %+v", q.Fields)
			}
		}
	}
	if asked == "" || len(told) != 0 {
		t.Fatalf("nothing is told before the owner's press: %v", told)
	}
	wantStatus(t, postForm(t, h, "/proposal/"+asked+"/accept", nil), http.StatusSeeOther)
	if len(told) != 1 || !strings.Contains(told[0], "said no") || !strings.Contains(told[0], words) {
		t.Fatalf("one message: the answer and the words: %v", told)
	}
	if page := get(t, h, "/request/"+token).Body.String(); !strings.Contains(page, "Friday at 10 works") {
		t.Errorf("and the request's page says them: %s", truncate(page))
	}
}
