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

// Two computers host one workspace: a note made on one is on the other
// after they keep in step, and a change to it there comes back.
func TestTwoHostsKeepOneWorkspace(t *testing.T) {
	a, ha := newApp(t)
	b, hb := newApp(t)
	sa, sb := httptest.NewServer(ha), httptest.NewServer(hb)
	defer sa.Close()
	defer sb.Close()
	peerOf := func(s *httptest.Server) string { return strings.TrimPrefix(s.URL, "http://") }

	n, err := a.Store.Create("note", map[string]any{"title": "Shared shopping"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := peers.With(context.Background(), http.DefaultClient, b.Store, peerOf(sa)); err != nil {
		t.Fatal(err)
	}
	got, err := b.Store.Get("note", n.ID)
	if err != nil || got.Fields["title"] != "Shared shopping" {
		t.Fatalf("the note should reach the second host: %v %v", got, err)
	}

	b.Store.Update("note", n.ID, map[string]any{"title": "Shared shopping, Saturday"})
	if _, err := peers.With(context.Background(), http.DefaultClient, a.Store, peerOf(sb)); err != nil {
		t.Fatal(err)
	}
	if back, _ := a.Store.Get("note", n.ID); back.Fields["title"] != "Shared shopping, Saturday" {
		t.Errorf("the change made on the second host comes back: %v", back.Fields)
	}
	if rec := get(t, hb, "/t/note/"+n.ID); !strings.Contains(rec.Body.String(), "Saturday") {
		t.Error("the second host's pages show the shared note")
	}
}

// Only the owner's own computers and people made hosts keep in step: an
// editor's device cannot pull the whole workspace or push into it.
func TestOnlyHostsMayKeepInStep(t *testing.T) {
	_, h := newApp(t)
	body := `{"seen":{},"stamps":[]}`
	if rec := as(t, h, chat.Visitor{Name: "Bob", Login: "bob@example.com", Access: chat.Edit}, http.MethodPost, "/sync", body, "application/json"); rec.Code != http.StatusForbidden {
		t.Errorf("an editor may not sync: %d", rec.Code)
	}
	if rec := as(t, h, chat.Visitor{Name: "Hana", Login: "hana@example.com", Access: chat.Host}, http.MethodPost, "/sync", body, "application/json"); rec.Code != http.StatusOK {
		t.Errorf("a host may: %d %s", rec.Code, rec.Body.String())
	}
}
