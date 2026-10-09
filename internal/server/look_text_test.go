package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An agent reads what a page says through look, not only its headings and
// controls: a record's lede with its badge and its day, and a message's
// words as the person reads them. The crew's checking roles asked for this
// fourteen times, one surface at a time.
func TestLookReadsWhatAPageSays(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLikeAPerson(t, a, h) // machine_seed_test.go
	paint := ""
	recs, _ := a.Store.List("task", store.ListOptions{})
	for _, r := range recs {
		if r.Fields["title"] == "Buy paint" {
			paint = r.ID
		}
	}
	says := func(path string) string {
		var o struct {
			Outline struct {
				Text []struct{ Under, Text string } `json:"text"`
			} `json:"outline"`
		}
		raw := get(t, h, "/api/look?only=text&path="+path).Body.Bytes()
		json.Unmarshal(raw, &o)
		var lines []string
		for _, p := range o.Outline.Text {
			lines = append(lines, p.Text)
		}
		return strings.Join(lines, "\n")
	}
	if task := says("/t/task/" + paint); !strings.Contains(task, "Done") || !strings.Contains(task, "at 2pm For Ana Silva") {
		t.Errorf("a record's lede is read with its day and its state:\n%s", task)
	}
	if chat := says("/chat"); !strings.Contains(chat, "Invalid API key") || strings.Contains(chat, "exit status") {
		t.Errorf("a message is read as the person reads it:\n%s", chat)
	}
}
