package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestAPIErrorsSayWhatIsWrongInPlainWords: a bad body is answered with
// what is wrong in the caller's terms, never the decoder's (backlog 0320,
// 0342). Go type names and "invalid character" puzzles tell an agent or a
// person nothing they can act on.
func TestAPIErrorsSayWhatIsWrongInPlainWords(t *testing.T) {
	_, h := newApp(t)
	leaks := []string{"Go value", "Go struct", "interface {}", "json:", "invalid character", "unexpected end of JSON", "map[string]"}
	for _, c := range []struct{ method, path, body, want string }{
		{"POST", "/api/note", "{bad", "not valid JSON"},
		{"POST", "/api/note", "[1]", "an array, not an object"},
		{"POST", "/api/note", `{"title": "x"`, "not valid JSON"},
		{"POST", "/api/look", `{"path": 5}`, "path should be text, not number"},
		{"POST", "/api/look", `{"colour": "red"}`, `"colour" is not one of them`},
	} {
		rec := do(t, h, c.method, c.path, strings.NewReader(c.body), "application/json")
		var answer struct{ Error struct{ Message string } }
		decode(t, rec, &answer)
		msg := answer.Error.Message
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s %s %s: want 400, got %d: %s", c.method, c.path, c.body, rec.Code, msg)
		}
		if !strings.Contains(msg, c.want) {
			t.Errorf("%s %s %s: the error should say %q: %s", c.method, c.path, c.body, c.want, msg)
		}
		for _, leak := range leaks {
			if strings.Contains(msg, leak) {
				t.Errorf("%s %s %s: the error shows the decoder's words %q: %s", c.method, c.path, c.body, leak, msg)
			}
		}
	}
}
