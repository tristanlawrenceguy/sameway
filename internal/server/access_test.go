package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// as makes a request as someone on another device, the way the tailnet
// marks one it has let in.
func as(t *testing.T, h http.Handler, v chat.Visitor, method, path string, body string, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req = req.WithContext(chat.WithVisitor(req.Context(), v))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Someone Tailscale says is bob@example.com gets in as the person with
// that email and the access the owner gave them; someone nobody let in is
// told they have asked, and the owner is asked, once.
func TestWhoGetsInIsWhoTheOwnerLetIn(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	if _, err := a.Store.Create("person", map[string]any{"name": "Bob", "email": "Bob@Example.com", "access": "edit"}); err != nil {
		t.Fatal(err)
	}

	ctx, _, ok := srv.Admit(context.Background(), "bob@example.com", "Robert", "pixel-7", false)
	v := chat.VisitorOf(ctx)
	if !ok || v.Access != chat.Edit || v.Name != "Bob" || v.Device != "pixel-7" {
		t.Errorf("Bob should get in to edit, under the name the owner knows him by: %+v %v", v, ok)
	}
	if ctx, _, ok := srv.Admit(context.Background(), "me@example.com", "Me", "laptop", true); !ok || !chat.VisitorOf(ctx).Owner() {
		t.Error("the owner's own device gets in as the owner")
	}

	for i := 0; i < 2; i++ {
		_, say, ok := srv.Admit(context.Background(), "carol@example.com", "Carol", "iphone", false)
		if ok || !strings.Contains(say, "once its owner says yes") {
			t.Fatalf("Carol is not let in and is told why: %q", say)
		}
	}
	asked := 0
	for _, p := range a.Chat.Proposals() {
		if summary, _ := p.Fields["summary"].(string); strings.Contains(summary, "carol@example.com") {
			asked++
		}
	}
	if asked != 1 {
		t.Errorf("the owner should be asked about Carol once, was asked %d times", asked)
	}
}

// Someone who may look can read but not change; someone who may edit can
// change content but not reach what is the owner's alone, nor give
// themselves more access through the API.
func TestWhatEachLevelMayDo(t *testing.T) {
	a, h := newApp(t)
	bob, err := a.Store.Create("person", map[string]any{"name": "Bob", "email": "bob@example.com", "access": "edit"})
	if err != nil {
		t.Fatal(err)
	}
	viewer := chat.Visitor{Name: "Vi", Login: "vi@example.com", Access: chat.View, Device: "tablet"}
	editor := chat.Visitor{Person: bob.ID, Name: "Bob", Login: "bob@example.com", Access: chat.Edit, Device: "pixel-7"}
	note := `{"title":"From a guest"}`

	if rec := as(t, h, viewer, http.MethodGet, "/t/note", "", ""); rec.Code != http.StatusOK {
		t.Errorf("a viewer reads: %d", rec.Code)
	}
	if rec := as(t, h, viewer, http.MethodPost, "/api/note", note, "application/json"); rec.Code != http.StatusForbidden {
		t.Errorf("a viewer cannot write: %d", rec.Code)
	}
	if rec := as(t, h, editor, http.MethodPost, "/api/note", note, "application/json"); rec.Code != http.StatusCreated {
		t.Errorf("an editor writes: %d %s", rec.Code, rec.Body.String())
	}
	for _, path := range []string{"/chat", "/workspaces", "/api/message", "/t/proposal"} {
		if rec := as(t, h, editor, http.MethodGet, path, "", ""); rec.Code != http.StatusForbidden {
			t.Errorf("%s is the owner's alone, an editor got %d", path, rec.Code)
		}
	}
	form := url.Values{"prop-access": {"edit"}}.Encode()
	if rec := as(t, h, editor, http.MethodPost, "/t/person/import", form, "application/x-www-form-urlencoded"); rec.Code != http.StatusForbidden {
		t.Errorf("an import could write access, so it is the owner's: %d", rec.Code)
	}
	carol, _ := a.Store.Create("person", map[string]any{"name": "Carol", "email": "carol@example.com"})
	if rec := as(t, h, editor, http.MethodPatch, "/api/person/"+carol.ID, `{"access":"edit"}`, "application/json"); rec.Code != http.StatusForbidden {
		t.Errorf("an editor cannot give access through the API: %d", rec.Code)
	}
	if got, _ := a.Store.Get("person", carol.ID); got.Fields["access"] == "edit" {
		t.Error("Carol's access must not have changed")
	}
	if rec := get(t, h, "/workspaces"); rec.Code == http.StatusForbidden {
		t.Error("the owner, at the machine itself, reaches everything")
	}
}

// What the owner said to the assistant stays theirs: a visitor sees the
// canvas without the conversation.
func TestAVisitorDoesNotSeeTheOwnersConversation(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Say("the owner's private words")
	if !strings.Contains(get(t, h, "/").Body.String(), "the owner&#39;s private words") {
		t.Skip("this canvas shows no conversation to hide")
	}
	body := as(t, h, chat.Visitor{Name: "Vi", Access: chat.View}, http.MethodGet, "/", "", "").Body.String()
	if strings.Contains(body, "private words") {
		t.Error("a visitor should not see the owner's conversation")
	}
	if !strings.Contains(body, "for the workspace's owner") {
		t.Error("a visitor should be told whose the assistant is")
	}
}

// A change someone else makes is theirs in the log, by name and device.
func TestTheLogSaysWhoElseMadeAChange(t *testing.T) {
	a, h := newApp(t)
	editor := chat.Visitor{Name: "Bob", Access: chat.Edit, Device: "pixel-7"}
	rec, _ := a.Store.Create("note", map[string]any{"title": "Shopping"})
	if r := as(t, h, editor, http.MethodPost, "/t/note/"+rec.ID+"/delete", "", "application/x-www-form-urlencoded"); r.Code >= 400 {
		t.Fatalf("Bob deletes a note: %d", r.Code)
	}
	log, _ := a.Store.List(chat.ActivityType, store.ListOptions{})
	found := false
	for _, e := range log {
		if s, _ := e.Fields["summary"].(string); strings.HasPrefix(s, "Bob deleted") && strings.HasSuffix(s, ", on pixel-7") {
			found = true
		}
	}
	if !found {
		t.Error("the log should say Bob deleted it, on pixel-7")
	}
}
