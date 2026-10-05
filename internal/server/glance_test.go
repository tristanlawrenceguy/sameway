package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A record says the same at a glance wherever it is met: its row in its
// list, a list on the canvas, its own page. Three functions once said it
// three ways, and the same task read "Due Fri 9 Oct 2026, 14:00" on the
// canvas and "In 4 days at 2:00pm" in its list.
func TestARecordSaysTheSameEverywhere(t *testing.T) {
	a, h := newApp(t)
	ana, _ := a.Store.Create("person", map[string]any{"name": "Ana Silva"})
	due := time.Now().AddDate(0, 0, 3).Format("2006-01-02") + " 14:00"
	task, _ := a.Store.Create("task", map[string]any{"title": "Buy paint", "status": "doing", "due": due, "for": ana.ID})
	a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}}))
	day := when.Relative(task.Fields["due"].(string), time.Now())

	pages := map[string]string{
		"its row":    read(get(t, h, "/t/task").Body.String()),
		"the canvas": read(get(t, h, "/").Body.String()),
		"its page":   read(get(t, h, "/t/task/"+task.ID).Body.String()),
	}
	for where, page := range pages {
		for _, says := range []string{"Doing", day, "Ana Silva"} {
			if !strings.Contains(page, says) {
				t.Errorf("%s does not say %q", where, says)
			}
		}
		if strings.Contains(page, when.Text(task.Fields["due"].(string))) {
			t.Errorf("%s says the day as a machine writes it", where)
		}
	}
	// And a done one says nothing of its state but its tick.
	postJSON(t, h, http.MethodPatch, "/api/task/"+task.ID, map[string]any{"status": "done"})
	if page := get(t, h, "/t/task").Body.String(); strings.Contains(page, ">Done</span>") && strings.Contains(page, "checked") {
		t.Error("a ticked task's row says Done once, by its box")
	}
}

// read is what a person reads on a page (machine_words_test.go).
func read(page string) string { return strings.Join(peopleText(page), "\n") }
