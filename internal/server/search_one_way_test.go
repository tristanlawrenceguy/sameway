package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// A search is one thing on every surface: when nothing has every word,
// the page, the API and the assistant all give what has some of them and
// say so in the same words.
func TestSearchIsOneWayEverywhere(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("note", map[string]any{"title": "Boiler", "body": "The engineer comes on Thursday."})
	if page := get(t, h, "/search?q=boiler+visit").Body.String(); !strings.Contains(page, search.SomeWords) || !strings.Contains(page, "Boiler") {
		t.Errorf("the page gives some of the words and says so")
	}
	var api struct {
		Some  bool   `json:"some"`
		Said  string `json:"said"`
		Total int    `json:"total"`
	}
	json.Unmarshal(get(t, h, "/api/search?q=boiler+visit").Body.Bytes(), &api)
	if !api.Some || api.Total != 1 || !strings.HasPrefix(api.Said, search.SomeWords) {
		t.Errorf("the API the same: %+v", api)
	}
	if text, _ := searchCall(a, map[string]any{"query": "boiler visit"}); !strings.HasPrefix(text, search.SomeWords) || !strings.Contains(text, "Boiler") {
		t.Errorf("the assistant the same: %s", text)
	}
}
