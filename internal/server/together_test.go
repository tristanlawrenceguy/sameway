package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/peers"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Two versions written at once: the record's page offers the other one,
// and using it puts it in place; keeping the page's clears the offer.
func TestTheOtherVersionIsOfferedOnThePage(t *testing.T) {
	a, h := newApp(t)
	n, _ := a.Store.Create("note", map[string]any{"title": "Plan", "body": "the later words"})
	now := time.Now().UTC()
	a.Store.Put(store.ClashType, "c1", map[string]any{"target": "note", "target_id": n.ID, "field": "body", "text": "the earlier words", "origin": "elsewhere", "state": "open"}, now, now)

	page := get(t, h, "/t/note/"+n.ID).Body.String()
	if !strings.Contains(page, "Another version of") || !strings.Contains(page, "the earlier words") || !strings.Contains(page, "Use this version") {
		t.Fatalf("the page offers the other version:\n%s", truncate(page))
	}
	if r := postForm(t, h, "/clash/c1/use", url.Values{"from": {"/t/note/" + n.ID}}); r.Code >= 400 {
		t.Fatalf("using it: %d", r.Code)
	}
	if got, _ := a.Store.Get("note", n.ID); got.Fields["body"] != "the earlier words" {
		t.Errorf("using the other version puts it in place: %v", got.Fields["body"])
	}
	if page := get(t, h, "/t/note/"+n.ID).Body.String(); strings.Contains(page, "Another version of") {
		t.Error("once chosen, it is no longer offered")
	}
}

// Someone else here is said, with where they are; oneself never is.
func TestWhoElseIsHere(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Owner = chat.Visitor{Access: chat.Owner, Login: "me@example.com", Name: "Me"}
	hana := chat.Visitor{Name: "Hana", Login: "hana@example.com", Access: chat.Edit}
	as(t, h, hana, http.MethodGet, "/t/note", "", "")
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, "Also here:") || !strings.Contains(page, "Hana</span>") && !strings.Contains(page, "Hana, on") {
		t.Errorf("the owner sees Hana is here:\n%s", truncate(page))
	}
	if !strings.Contains(page, "on Notes") {
		i := strings.Index(page, "Also here:")
		t.Errorf("and where she is: %s", page[i:i+300])
	}
	if theirs := as(t, h, hana, http.MethodGet, "/", "", "").Body.String(); strings.Contains(theirs, ">Hana") && strings.Contains(theirs, "Also here:") && !strings.Contains(theirs, "Me") {
		t.Error("Hana is not told she is here")
	}
	srv := h.(*server.Server)
	srv.HearPresence([]peers.Presence{{Login: "bob@example.com", Name: "Bob", Place: "Plan"}})
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "Bob, on Plan") {
		t.Error("someone on another computer is here too")
	}
}

// A task made out for this computer's owner on another computer tells
// them, once, and their assistant knows it is for them.
func TestSomethingForYouReachesYou(t *testing.T) {
	mine, hMine := newApp(t)
	hana, hHana := newApp(t)
	mine.Chat.Owner = chat.Visitor{Access: chat.Owner, Login: "me@example.com", Name: "Me"}
	rang := make(chan string, 4)
	hMine.(*server.Server).OnRing(func(title, text, url string) { rang <- title })

	me, _ := hana.Store.Create("person", map[string]any{"name": "Me", "email": "me@example.com"})
	task, _ := hana.Store.Create("task", map[string]any{"title": "Order compost", "for": me.ID})
	srv := httptest.NewServer(hHana)
	defer srv.Close()
	peer := strings.TrimPrefix(srv.URL, "http://")
	peers.With(context.Background(), http.DefaultClient, mine.Store, peer)
	select {
	case title := <-rang:
		if title != "For you: Order compost" {
			t.Errorf("the ring says what is for me: %q", title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the owner should hear the task is for them")
	}
	hana.Store.Update("task", task.ID, map[string]any{"notes": "the big bag"})
	peers.With(context.Background(), http.DefaultClient, mine.Store, peer)
	select {
	case again := <-rang:
		t.Errorf("once is enough: %q", again)
	case <-time.After(300 * time.Millisecond):
	}
	if page := get(t, hMine, "/t/task/"+task.ID).Body.String(); !strings.Contains(page, `class="sw-person"`) {
		t.Error("the task shows who it is for, in their colour")
	}
}

// Back after a while, what the others changed meanwhile is waiting, with
// its Undo, until Got it.
func TestSinceYouWereLastHere(t *testing.T) {
	a, h := newApp(t)
	a.Store.SetMeta("last:owner", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano))
	chat.Record(a.Store, "human", chat.Change{Action: "deleted", Component: "note", Detail: "Shopping", By: "Hana", ByLogin: "hana@example.com"})
	chat.Record(a.Store, "human", chat.Change{Action: "added", Component: "note", Detail: "Mine"})

	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, "Since you were last here") || !strings.Contains(page, "Shopping") {
		t.Fatalf("what Hana did while I was away is waiting:\n%s", truncate(page))
	}
	notice := page[strings.Index(page, "Since you were last here"):]
	notice = notice[:strings.Index(notice, "</section>")]
	if strings.Contains(notice, "Mine") {
		t.Error("my own change is not news to me")
	}
	postForm(t, h, "/since/seen", url.Values{"from": {"/"}})
	if page := get(t, h, "/").Body.String(); strings.Contains(page, "Since you were last here") {
		t.Error("Got it puts it away")
	}
}
