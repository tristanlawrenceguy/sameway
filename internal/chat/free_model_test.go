package chat_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

func ask(t *testing.T, svc *chat.Service, tool string, args map[string]any) (string, bool) {
	t.Helper()
	raw, _ := json.Marshal(args)
	return svc.Call(tool, raw)
}

// What a small model on this computer got wrong, measured on the bench's
// hard set, and what Sameway now tells it: a day searched for finds what
// falls on it; words that match nothing list what there is to judge; a
// task naming a person here says whose to set; a change the person's words
// fit no better than another's is stopped once, to ask; a record's days
// are said in this computer's time.
func TestWhatASmallModelIsToldNow(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	now := time.Now()
	thursday := now.AddDate(0, 0, (int(time.Thursday)-int(now.Weekday())+7)%7)
	if thursday.Format("2006-01-02") == now.Format("2006-01-02") {
		thursday = thursday.AddDate(0, 0, 7)
	}
	day := thursday.Format("2006-01-02")
	bank, _ := svc.Store.Create("task", map[string]any{"title": "Call the bank", "due": day})
	svc.Store.Create("event", map[string]any{"title": "Team lunch", "starts": day + "T12:30"})
	for _, n := range []string{"Tomato seedlings", "Compost bins"} {
		svc.Store.Create("note", map[string]any{"title": n})
	}

	if text, _ := ask(t, svc, "search", map[string]any{"query": "Thursday"}); !strings.Contains(text, "Call the bank") || !strings.Contains(text, "Team lunch") || !strings.Contains(text, "12:30") {
		t.Errorf("a day searched for finds what falls on it: %s", text)
	}
	if text, _ := ask(t, svc, "search", map[string]any{"query": "garden", "type": "note"}); !strings.Contains(text, "Tomato seedlings") || !strings.Contains(text, "Compost bins") {
		t.Errorf("no words found lists the notes to judge: %s", text)
	}
	if text, _ := ask(t, svc, "find_records", map[string]any{"type": "task", "query": "Thursday"}); !strings.Contains(text, "Call the bank") || !strings.Contains(text, "due "+thursday.Format("Monday 2 January")) || strings.Contains(text, "02:00") || strings.Contains(text, "00:00") {
		t.Errorf("find_records says each day with its weekday: %s", text)
	}
	if text, _ := ask(t, svc, "get_record", map[string]any{"type": "task", "id": bank.ID}); !strings.Contains(text, `"days_here"`) || !strings.Contains(text, "without Z") {
		t.Errorf("get_record says its days here: %s", text)
	}

	ana, _ := svc.Store.Create("person", map[string]any{"name": "Ana Silva"})
	if text, _ := ask(t, svc, "create_record", map[string]any{"type": "task", "fields": map[string]any{"title": "Ana Silva: send the price list"}}); !strings.Contains(text, ana.ID) {
		t.Errorf("a task naming a person here says whose to set: %s", text)
	}

	a, _ := svc.Store.Create("task", map[string]any{"title": "Send invoice to Ana"})
	joe, _ := svc.Store.Create("task", map[string]any{"title": "Send invoice to Joe"})
	svc.Store.Create(records.MessageType, map[string]any{"role": "user", "content": "Mark the invoice to Joe done"})
	if text, isErr := ask(t, svc, "update_record", map[string]any{"type": "task", "id": joe.ID, "fields": map[string]any{"done": true}}); isErr {
		t.Errorf("words that single one out are not stopped: %s", text)
	}
	svc.Store.Create(records.MessageType, map[string]any{"role": "user", "content": "Mark the invoice task done"})
	text, isErr := ask(t, svc, "update_record", map[string]any{"type": "task", "id": a.ID, "fields": map[string]any{"done": true}})
	if !isErr || !strings.Contains(text, "Send invoice to Joe") || !strings.Contains(text, "Ask them which") {
		t.Fatalf("a change the words do not single out is stopped, to ask: %s", text)
	}
	if got, _ := svc.Store.Get("task", a.ID); got.Fields["done"] == true {
		t.Error("and nothing changed")
	}
	if _, isErr := ask(t, svc, "update_record", map[string]any{"type": "task", "id": a.ID, "fields": map[string]any{"done": true}}); isErr {
		t.Error("sent again, it goes through")
	}
}

// An action made after the person said when it runs, without when, says
// how; one sending a name the record does not have says what it has.
func TestAnActionSaysWhenItShouldRun(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Store.Create(records.MessageType, map[string]any{"role": "user", "content": "When a task tagged client is marked done, send its title to https://example.com/hook"})
	text, _ := ask(t, svc, "create_record", map[string]any{"type": "action", "fields": map[string]any{"title": "Tell the hook", "kind": "webhook", "url": "https://example.com/hook", "body": "{{name}} for {{for}}"}})
	if !strings.Contains(text, "set when") || !strings.Contains(text, "nothing called {{name}}") || !strings.Contains(text, "{{title}}") {
		t.Errorf("%s", text)
	}
}
