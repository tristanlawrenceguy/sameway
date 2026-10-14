package agenda_test

import (
	"strings"
	"testing"
)

// A task that matters and is not due yet is among the three with Done
// alone: Tomorrow would bring its day forward, or give it one, and leave
// it among the three. The line above the three says why one not due is
// there, and an agent reading the task type learns the same from its tags.
func TestWhatMattersIsSaidAndNotMovedForward(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	for i, title := range []string{"Buy flowers", "Buy fresh produce", "Return library books", "Pick up dry cleaning", "Water the plants", "Clean the fridge"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-3 + i/2)})
	}
	a.Store.Create("task", map[string]any{"title": "Move the dentist appointment", "due": day(2), "tags": []any{"important"}})
	page := get(t, h, "/today").Body.String()
	if !strings.Contains(page, "Start with these three, what is marked important first; the rest wait below.") {
		t.Fatalf("the line says why: %s", truncate(page))
	}
	at := strings.Index(page, "Move the dentist appointment")
	row := page[at : at+strings.Index(page[at:], "</li>")]
	if !strings.Contains(row, "Done") || strings.Contains(row, "Tomorrow") {
		t.Errorf("a task not due yet is done, not moved to tomorrow: %s", row)
	}
	late := strings.Index(page, "Buy flowers")
	if !strings.Contains(page[late:late+strings.Index(page[late:], "</li>")], "Tomorrow") {
		t.Error("a late task still moves")
	}
	typ, _ := a.Types.Get("task")
	said := false
	for _, f := range typ.Fields {
		said = said || f.Name == "tags" && strings.Contains(f.Description, "important") && strings.Contains(f.Description, "Today")
	}
	if !said {
		t.Error("an agent reading the task type learns what marking important does")
	}
}
