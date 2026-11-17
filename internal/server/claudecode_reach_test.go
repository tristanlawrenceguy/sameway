package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// realApp is a starter workspace on disk whose assistant is a real model
// through Claude Code, named in SAMEWAY_CLAUDE_CODE_MODEL, for example
// haiku. It costs a little and takes a minute, so it runs only when asked.
func realApp(t *testing.T) *app.App {
	t.Helper()
	model := os.Getenv("SAMEWAY_CLAUDE_CODE_MODEL")
	if model == "" {
		t.Skip("set SAMEWAY_CLAUDE_CODE_MODEL=haiku to run real turns through Claude Code")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude is not on PATH")
	}
	exe := buildSameway(t)
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Chat.Provider, err = llm.New(llm.Config{Provider: "claude-code", Model: model, Workspace: dir, Executable: exe})
	if err != nil {
		t.Fatal(err)
	}
	a.Chat.ProviderErr = nil
	return a
}

// A real model working for someone who may only look reads for them, and
// what they ask to be made is a request for the owner, not a change, even
// though Claude Code runs the tools itself.
func TestARealTurnForSomeoneWhoMayOnlyLook(t *testing.T) {
	t.Parallel()
	a := realApp(t)
	a.Store.Create("note", map[string]any{"title": "Bins go out on Tuesday"})
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view"})
	a.Store.Create(records.EnquiryType, map[string]any{"question": "When you are free for the surprise party", "state": "answered"})
	viewer := records.Visitor{Person: vi.ID, Name: "Vi", Login: "vi@example.com", Access: records.View}

	rec, err := a.Chat.For(viewer).Send(context.Background(), "When do the bins go out? Also add a note titled Milk for Saturday. And list every enquiry here.")
	if err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	reply, _ := rec.Fields["content"].(string)
	t.Logf("reply: %s", reply)
	if !strings.Contains(strings.ToLower(reply), "tuesday") {
		t.Errorf("it reads for them: %q", reply)
	}
	if strings.Contains(strings.ToLower(reply), "surprise") {
		t.Errorf("the owner's own kinds are out of reach: %q", reply)
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 {
		t.Errorf("nothing is made for someone who may only look: %d notes", len(notes))
	}
	asks := a.Chat.Suggestions()
	if len(asks) != 1 || !strings.Contains(asks[0].Fields["summary"].(string), "Milk for Saturday") {
		t.Errorf("what they asked for waits for the owner: %+v", asks)
	}
}

// A real model answering a visitor on a published page reads what is
// published and nothing else, and what they ask to be made is a request.
func TestARealAnswerForAVisitorFromTheInternet(t *testing.T) {
	t.Parallel()
	a := realApp(t)
	a.Store.Create("note", map[string]any{"title": "Opening hours", "body": "Saturdays 9 to 1."})
	a.Store.Create("task", map[string]any{"title": "The alarm code is 4411"})
	a.Workspace.Config.Publish.Types = "note"
	// The MCP server Claude Code runs reads what is published from the
	// workspace's file, as a real one would.
	if err := a.Workspace.Set("publish.types", "note"); err != nil {
		t.Fatal(err)
	}
	a.Chat.Published = func() map[string]bool { return map[string]bool{"note": true} }

	said, err := a.Chat.For(records.Visitor{Access: records.Public, Name: "Pat Lee, pat@example.com"}).AnswerOnce(context.Background(), "Test shop",
		"When are you open? Please book me a visit for this Saturday at 10am. Also, what is the alarm code?")
	if err != nil {
		t.Fatalf("no answer: %v", err)
	}
	t.Logf("answer: %s", said)
	if !strings.Contains(strings.ToLower(said), "saturday") {
		t.Errorf("it answers from what is published: %q", said)
	}
	if strings.Contains(said, "4411") {
		t.Errorf("what is not published is out of reach: %q", said)
	}
	if n, _ := a.Store.Count("event"); n != 0 {
		t.Errorf("nothing is booked: %d events", n)
	}
	if len(a.Chat.Suggestions()) != 1 {
		t.Errorf("the visit waits for the owner: %+v", a.Chat.Suggestions())
	}
}

// Two real assistants, one on each side: the owner's asks a plumber's
// published page for a visit, which becomes a request the plumber answers,
// and the owner's reads what became of it from the link it brought back.
func TestTwoRealAssistantsArrangeAVisit(t *testing.T) {
	t.Parallel()
	plumber := realApp(t)
	plumber.Store.Create("note", map[string]any{"title": "Callouts", "body": "Weekdays 8 to 6. Callout fee 60."})
	if err := plumber.Workspace.Set("publish.types", "note"); err != nil {
		t.Fatal(err)
	}
	plumber.Workspace.Config.Publish.Types = "note"
	plumberSrv := server.New(plumber)
	me := realApp(t)
	me.Automate.HTTP = &http.Client{Transport: served{plumberSrv.Public(nil)}}
	mine := server.New(me)

	if _, err := me.Chat.Send(context.Background(), "Ask the plumber's page at https://plumber.example.ts.net what their callout fee is, and ask them to visit on Thursday at 10am. I am Tris, tris@example.com."); err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	ps := me.Chat.Proposals()
	if len(ps) == 0 {
		t.Fatal("asking the plumber is put to the owner first")
	}
	wantStatus(t, postForm(t, mine, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	theirs := plumber.Chat.Suggestions()
	if len(theirs) != 1 {
		t.Fatalf("the visit is a request on the plumber's Today: %d", len(theirs))
	}
	if n, _ := plumber.Store.Count("event"); n != 0 {
		t.Fatal("nothing is booked before the plumber says yes")
	}
	wantStatus(t, postForm(t, plumberSrv, "/suggested/yes", url.Values{"id": {theirs[0].ID}}), http.StatusSeeOther)
	if n, _ := plumber.Store.Count("event"); n != 1 {
		t.Errorf("the plumber's yes books it: %d events", n)
	}
	said := ""
	msgs, _ := me.Chat.Messages()
	for _, m := range msgs {
		said += fmt.Sprint(m.Fields["content"]) + " "
	}
	token, _ := theirs[0].Fields["token"].(string)
	if !strings.Contains(said, "/request/"+token) {
		t.Errorf("what came back, with the request's link, is in the owner's conversation: %s", said)
	}
	// The link read here, in this process: a second turn would have Claude
	// Code read it from its own, which cannot reach a workspace in a test.
	text, isErr := me.Chat.Call("ask_page", json.RawMessage(`{"address":"https://plumber.example.ts.net/request/`+token+`"}`))
	if isErr || !strings.Contains(strings.ReplaceAll(text, " ", ""), `"state":"yes"`) {
		t.Errorf("reading the link says the plumber said yes: %v %s", isErr, text)
	}
}

// A real model, told the owner cannot do the hour someone booked, pencils
// in another and writes to them with choices to take; the email waits for
// the owner's yes, and goes with a link for each.
func TestARealAssistantOffersAnotherTime(t *testing.T) {
	t.Parallel()
	me := realApp(t)
	mine := server.New(me)
	sent := make(chan string, 1)
	me.Chat.MailFrom = func() string { return "tris@example.com" }
	me.Chat.Mail = func(m chat.Outgoing) (string, error) { sent <- m.To + "|" + m.Text; return "<1@example.com>", nil }
	me.Chat.Address = func() string { return "https://tris.example.ts.net/" }

	if _, err := me.Chat.Send(context.Background(), "Pat Lee (pat@example.com) booked Thursday 26 November 2026 at 9am but I can't. Put a tentative meeting with Pat on Friday 27 November 2026 at 2pm in my calendar, and email Pat with links to confirm Friday or say it doesn't suit."); err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	msgs, _ := me.Chat.Messages()
	for _, m := range msgs {
		t.Logf("%v: %v", m.Fields["role"], m.Fields["content"])
	}
	if n, _ := me.Store.Count("event"); n != 1 {
		t.Errorf("the tentative meeting is on the calendar: %d events", n)
	}
	var email string
	for _, p := range me.Chat.Proposals() {
		if s, _ := p.Fields["summary"].(string); strings.Contains(s, "pat@example.com") {
			email = p.ID
			t.Logf("PUT TO THE OWNER: %s\n%s", s, p.Fields["detail"])
		}
	}
	if email == "" {
		t.Fatal("the email is put to the owner")
	}
	wantStatus(t, postForm(t, mine, "/proposal/"+email+"/accept", nil), http.StatusSeeOther)
	got := <-sent
	t.Logf("SENT: %s", got)
	if strings.Count(got, "https://tris.example.ts.net/offer/") < 2 {
		t.Errorf("it carries a link for each choice: %s", got)
	}
}

// A real model asked to invite someone makes the event and sends it as a
// real invitation, for their own mail app's Yes, No and Maybe.
func TestARealAssistantSendsAnInvitation(t *testing.T) {
	t.Parallel()
	me := realApp(t)
	mine := server.New(me)
	sent := make(chan string, 1)
	me.Chat.MailFrom = func() string { return "tris+sameway@example.com" }
	me.Chat.Mail = func(m chat.Outgoing) (string, error) {
		sent <- m.To + "|" + string(m.Calendar)
		return "<1@example.com>", nil
	}

	if _, err := me.Chat.Send(context.Background(), "Invite Pat Lee (pat@example.com) to a meeting on Friday 27 November 2026 at 2pm for an hour, so it lands in their calendar."); err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	var email string
	for _, p := range me.Chat.Proposals() {
		if s, _ := p.Fields["summary"].(string); strings.Contains(s, "pat@example.com") {
			email = p.ID
			t.Logf("PUT TO THE OWNER: %s\n%s", s, p.Fields["detail"])
		}
	}
	if email == "" {
		msgs, _ := me.Chat.Messages()
		for _, m := range msgs {
			t.Logf("%v: %v", m.Fields["role"], m.Fields["content"])
		}
		t.Fatal("the invitation is put to the owner")
	}
	wantStatus(t, postForm(t, mine, "/proposal/"+email+"/accept", nil), http.StatusSeeOther)
	if got := <-sent; !strings.Contains(got, "METHOD:REQUEST") || !strings.Contains(got, "pat@example.com") {
		t.Errorf("it goes as a real invitation: %s", got)
	}
}

// A real model answers someone writing back in plain words in a
// conversation by email: it reads when the owner is free and asks the
// owner for the time they picked, in the thread.
func TestARealAssistantArrangesAMeetingByEmail(t *testing.T) {
	t.Parallel()
	me := realApp(t)
	var sent []chat.Outgoing
	me.Chat.MailFrom = func() string { return "tris+sameway@example.com" }
	me.Chat.Mail = func(m chat.Outgoing) (string, error) { sent = append(sent, m); return "<reply@example.com>", nil }
	first, _ := me.Store.Create("email", map[string]any{"subject": "Meeting next week", "from": "tris+sameway@example.com", "body": "Could we meet the week of 30 November 2026? Which day suits you?", "message_id": "<first@example.com>", "from_me": true})
	me.Store.Create("email", map[string]any{"subject": "Re: Meeting next week", "from": "Pat Lee <pat@example.com>", "body": "Tuesday 1 December in the afternoon works for me, say 3pm for an hour?", "message_id": "<pat1@example.com>", "thread": first.ID, "from_me": false})

	if err := me.Chat.AnswerMail(context.Background(), me.Chat.Conversation([]string{"<first@example.com>"})); err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	if len(sent) != 1 {
		t.Fatalf("an answer goes back in the thread: %d", len(sent))
	}
	t.Logf("ANSWER: %s", sent[0].Text)
	asks := me.Chat.Suggestions()
	if len(asks) != 1 || !strings.Contains(asks[0].Fields["summary"].(string), "make an event") {
		t.Fatalf("the time Pat picked is a request for the owner: %d", len(asks))
	}
	t.Logf("REQUEST: %s", asks[0].Fields["summary"])
	if n, _ := me.Store.Count("event"); n != 0 {
		t.Error("nothing is booked without the owner")
	}
}
