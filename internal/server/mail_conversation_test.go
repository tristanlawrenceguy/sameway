package server_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// conversation is three emails: Joe asks, the person answers, Joe asks
// again, quoting what came before; the last says it is your turn.
func conversation(t *testing.T, st *store.Store) (first, mine, last *store.Record) {
	t.Helper()
	first, _ = st.Create("email", map[string]any{"subject": "Quote", "from": "Joe <joe@joe.example>", "received": "2026-10-05T09:00:00Z", "body": "Can you send the quote by Friday?", "message_id": "q1@joe.example"})
	mine, _ = st.Create("email", map[string]any{"subject": "Re: Quote", "from": "Me <me@example.com>", "received": "2026-10-05T10:00:00Z", "body": "Here it is, attached.", "thread": first.ID, "from_me": true})
	last, _ = st.Create("email", map[string]any{"subject": "Re: Quote", "from": "Joe <joe@joe.example>", "received": "2026-10-06T11:00:00Z", "turn": "yours",
		"body": "Thanks. Which day suits for the visit?\n\nOn Mon 5 Oct, Me wrote:\n> Here it is, attached.", "thread": first.ID})
	if first == nil || mine == nil || last == nil {
		t.Fatal("the conversation was made")
	}
	return first, mine, last
}

// An email's page is that email at rest; asked, it shows its whole
// conversation oldest first, each email under a heading of who and when,
// in its own words (not what it quotes), and this email only named.
func TestAnEmailPageShowsItsConversationWhenAsked(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	first, mine, last := conversation(t, a.Store)
	if page := get(t, h, "/t/email/"+last.ID).Body.String(); strings.Contains(page, `id="conversation"`) {
		t.Error("at rest the page is the email alone")
	}
	page := get(t, h, "/t/email/"+last.ID+"?show=conversation").Body.String()
	conv := page[strings.Index(page, `id="conversation"`):]
	joe, you, this := strings.Index(conv, ">Joe, "), strings.Index(conv, ">You, "), strings.Index(conv, "this email</h3>")
	if !strings.Contains(conv, "The conversation, 3 emails</h2>") || joe < 0 || you < joe || this < you {
		t.Fatalf("oldest first, by who and when, this one last: %s", truncate(conv))
	}
	if !strings.Contains(conv, `href="/t/email/`+first.ID+`"`) || !strings.Contains(conv, `href="/t/email/`+mine.ID+`"`) {
		t.Error("each other email leads to its own page")
	}
	if strings.Contains(conv, "Which day suits") {
		t.Error("this email's words are said once, at the top, not again in the conversation")
	}
	if strings.Count(conv, "Here it is, attached.") != 1 {
		t.Error("what a reply quotes is not said again")
	}
	if !strings.Contains(conv, "Fewer") {
		t.Error("the address opened it, so a link closes it")
	}
	var rec struct {
		WrittenBy string `json:"written_by"`
		Parts     []struct{ Key string }
	}
	json.Unmarshal(get(t, h, "/api/email/"+last.ID).Body.Bytes(), &rec)
	if rec.WrittenBy != "an email from Joe <joe@joe.example>" {
		t.Errorf("an agent is told the words are the sender's, not Sameway's: %q", rec.WrittenBy)
	}
	if page := get(t, h, "/t/email/"+first.ID).Body.String(); strings.Contains(page, "an action run by a schedule") {
		t.Error("an email is not said to be from an action")
	}
}

// Your turn to reply opens on the conversation, says only the email's own
// words, and No reply needed takes it off, logged and undone.
func TestNoReplyNeededTakesAConversationOffYourTurn(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	_, _, last := conversation(t, a.Store)
	today := get(t, h, "/today").Body.String()
	if !strings.Contains(today, `/t/email/`+last.ID+`?show=conversation#conversation`) {
		t.Errorf("Your turn opens on the conversation: %s", truncate(today))
	}
	if !strings.Contains(today, "Joe: Thanks. Which day suits for the visit?</span>") {
		t.Errorf("the line is who and their own words: %s", truncate(today))
	}
	if !strings.Contains(today, `action="/mail/answered"`) || !strings.Contains(today, "No reply needed") {
		t.Fatalf("each has No reply needed: %s", truncate(today))
	}
	res := postForm(t, h, "/mail/answered", url.Values{"id": {last.ID}})
	if after, _ := a.Store.Get("email", last.ID); after.Fields["turn"] != "" && after.Fields["turn"] != nil {
		t.Errorf("off your turn: %v", after.Fields["turn"])
	}
	if strings.Contains(get(t, h, "/today").Body.String(), "Your turn to reply") {
		t.Error("Today no longer lists it")
	}
	if loc := res.Header().Get("Location"); loc == "" {
		t.Errorf("the person is told, with undo: %d", res.Code)
	}
	if again := postForm(t, h, "/mail/answered", url.Values{"id": {last.ID}}); again.Code >= 500 {
		t.Errorf("twice is said plainly: %d", again.Code)
	}
}
