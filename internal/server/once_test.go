package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change sent again with the same Idempotency-Key is done once and
// answered as it was the first time; the same key for a different change
// is refused.
func TestAChangeSentTwiceIsDoneOnce(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	send := func(body, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/note", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		return res
	}
	first := send(`{"title":"Seeds"}`, "k1")
	again := send(`{"title":"Seeds"}`, "k1")
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 {
		t.Fatalf("sent twice, made once: %d notes", len(notes))
	}
	if again.Code != first.Code || again.Body.String() != first.Body.String() || again.Header().Get("Idempotent-Replayed") != "true" {
		t.Errorf("the repeat is answered as the first was: %d %s", again.Code, again.Body.String())
	}
	if res := send(`{"title":"Other"}`, "k1"); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("a key reused for another change is refused: %d", res.Code)
	}
	send(`{"title":"Other"}`, "k2")
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 2 {
		t.Errorf("a new key is a new change: %d notes", len(notes))
	}
	// A page's action takes the key too: the same form twice is one note.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/t/note/add", nil)
		req.Header.Set("Idempotency-Key", "add-1")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 3 {
		t.Errorf("the page's Add twice with one key is one note: %d notes", len(notes))
	}
}
