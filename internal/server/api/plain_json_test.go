package api_test

import (
	"strings"
	"testing"
)

// The API's answers are read as text by agents: <, > and & are as written,
// in its own instructions and in a record, never \u003c.
func TestTheAPIWritesAngleBracketsAsTheyAre(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("note", map[string]any{"title": "Tom & Jerry <draft>"})
	describe := get(t, h, "/api/describe").Body.String()
	if !strings.Contains(describe, "Bearer <key>") || strings.Contains(describe, `\u003c`) {
		t.Errorf("the description says Bearer <key> as written")
	}
	if notes := get(t, h, "/api/note").Body.String(); !strings.Contains(notes, "Tom & Jerry <draft>") {
		t.Errorf("a record's words are as written: %.300s", notes)
	}
}
