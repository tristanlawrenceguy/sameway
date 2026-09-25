package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/peers"
)

// A change made on another computer that hosts the workspace comes here in
// the log as that person's, by name and in their colour, and glows in
// their colour on the canvas; what was said to the assistant there does
// not come at all.
func TestAChangeFromAnotherHostIsTheirs(t *testing.T) {
	mine, hMine := newApp(t)
	hana, hHana := newApp(t)
	mine.Chat.Owner = chat.Visitor{Access: chat.Owner, Login: "tristan@example.com", Name: "Tristan"}
	hana.Chat.Owner = chat.Visitor{Access: chat.Owner, Login: "hana@example.com", Name: "Hana"}
	srv := httptest.NewServer(hHana)
	defer srv.Close()

	note, _ := hana.Store.Create("note", map[string]any{"title": "Shopping"})
	if r := postForm(t, hHana, "/t/note/"+note.ID+"/delete", nil); r.Code >= 400 {
		t.Fatalf("Hana deletes the note on her computer: %d", r.Code)
	}
	chat.Record(hana.Store, "human", chat.Change{Action: "said", Detail: "something private"})

	if _, err := peers.With(context.Background(), http.DefaultClient, mine.Store, strings.TrimPrefix(srv.URL, "http://")); err != nil {
		t.Fatal(err)
	}
	page := get(t, hMine, "/activity").Body.String()
	if !strings.Contains(page, `data-person="`) || !strings.Contains(page, ">hana</span>") && !strings.Contains(page, ">Hana</span>") {
		t.Errorf("Hana's deletion should read as hers, in her colour:\n%s", truncate(page))
	}
	if strings.Contains(page, "something private") {
		t.Error("what was said to the assistant stays on the computer it was said on")
	}
	if strings.Contains(page, "<span class=\"sw-event__actor\">You</span> <span class=\"sw-event__action\">deleted") {
		t.Error("Hana's deletion must not read as mine")
	}
}

// Someone else's latest change to a block makes it glow in their colour.
func TestABlockGlowsInTheColourOfWhoChangedIt(t *testing.T) {
	a, h := newApp(t)
	blk, err := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "heading", "props": map[string]any{"text": "Plan"}, "actor": "human"}))
	if err != nil {
		t.Fatal(err)
	}
	chat.Record(a.Store, "human", chat.Change{Action: "updated", Component: "heading", ID: blk.ID, By: "Bob", ByLogin: "bob@example.com"})
	body := get(t, h, "/").Body.String()
	i := strings.Index(body, `data-block-id="`+blk.ID+`"`)
	if i < 0 {
		t.Skip("the block is not on the home canvas")
	}
	tag := body[i : strings.Index(body[i:], ">")+i]
	if strings.Contains(tag, "data-changed") && !strings.Contains(tag, `data-person="`) {
		t.Errorf("Bob's change should glow in his colour: %s", tag)
	}
}

func TestEveryoneHasOneColourEverywhere(t *testing.T) {
	c := chat.PersonColour("Bob@Example.com")
	if c < 1 || c > 6 || c != chat.PersonColour("bob@example.com") {
		t.Errorf("a person's colour is one of six and does not depend on case: %d", c)
	}
	if chat.PersonColour("") != 0 {
		t.Error("nobody has no colour")
	}
}
