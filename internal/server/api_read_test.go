package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// An agent reads a long list a page at a time and only the fields it
// needs, and is told the fields there are when it names one that is not.
func TestAnAgentReadsLess(t *testing.T) {
	_, h := newApp(t)
	for i := 1; i <= 7; i++ {
		postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": fmt.Sprintf("Note %d", i), "body": strings.Repeat("words ", 50)})
	}
	var first map[string]any
	decode(t, get(t, h, "/api/note?page=1&limit=3&fields=title"), &first)
	recs := first["records"].([]any)
	if len(recs) != 3 || first["total"] != float64(7) || first["pages"] != float64(3) || !strings.Contains(fmt.Sprint(first["next"]), "page=2") {
		t.Fatalf("three of seven, three pages, and the next: %v", first)
	}
	fields := recs[0].(map[string]any)["fields"].(map[string]any)
	if len(fields) != 1 || fields["title"] == nil {
		t.Errorf("only the title: %v", fields)
	}
	var last map[string]any
	decode(t, get(t, h, "/api/note?page=3&limit=3"), &last)
	if len(last["records"].([]any)) != 1 || last["next"] != nil {
		t.Errorf("the last page has the one left and no next: %v", last)
	}
	var all map[string]any
	decode(t, get(t, h, "/api/note"), &all)
	if len(all["records"].([]any)) != 7 {
		t.Errorf("without page, every one, as before")
	}
	res := get(t, h, "/api/note?fields=title,colour")
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "its fields are") {
		t.Errorf("a field it does not have is named, with the ones it has: %d %s", res.Code, res.Body.String())
	}
	id := recs[0].(map[string]any)["id"].(string)
	var one map[string]any
	decode(t, get(t, h, "/api/note/"+id+"?fields=title"), &one)
	if f := one["fields"].(map[string]any); len(f) != 1 {
		t.Errorf("one record, only the title: %v", f)
	}
}
