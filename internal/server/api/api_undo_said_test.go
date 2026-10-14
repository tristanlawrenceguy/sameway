package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Every write answers with how to take it back: its log entry and the
// call. Made, a list made at once, and deleted are each taken back by the
// call their answer gave.
func TestAWriteSaysHowToTakeItBack(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	send := func(method, path, body string) map[string]any {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var out map[string]any
		json.Unmarshal(rec.Body.Bytes(), &out)
		if rec.Code >= 300 {
			t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body)
		}
		return out
	}
	undo := func(answer map[string]any) {
		u, _ := answer["undo"].(map[string]any)
		entry, _ := u["entry"].(string)
		how, _ := u["how"].(string)
		if entry == "" || !strings.Contains(how, `/api/tools/undo_change`) || !strings.Contains(how, entry) {
			t.Fatalf("the answer says how to take it back: %v", answer["undo"])
		}
		send(http.MethodPost, "/api/tools/undo_change", `{"id": "`+entry+`"}`)
	}
	count := func() int { n, _ := a.Store.List("note", store.ListOptions{}); return len(n) }

	undo(send(http.MethodPost, "/api/note", `{"title": "Trying"}`))
	if count() != 0 {
		t.Errorf("the note made is taken back")
	}
	undo(send(http.MethodPost, "/api/note", `[{"title": "One"}, {"title": "Two"}]`))
	if count() != 0 {
		t.Errorf("the two made at once are taken back at once")
	}
	kept := send(http.MethodPost, "/api/note", `{"title": "Keep me"}`)
	undo(send(http.MethodDelete, "/api/note/"+kept["id"].(string), ""))
	if count() != 1 {
		t.Errorf("the deleted note is put back")
	}
}
