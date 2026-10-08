package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// What is in a record is a glance fact like the rest (glance_count.go),
// worked out from the schema and said the same on every surface: its row,
// its page's lede, a list on the canvas, the API's glance and look's text.
// It was once added to project rows alone, by the type's name, and said
// nowhere else.
func TestWhatIsInARecordIsSaidTheSameEverywhere(t *testing.T) {
	a, h := newApp(t)
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	for _, task := range []map[string]any{
		{"title": "Dig the pond", "project": garden.ID, "done": true},
		{"title": "Plant garlic", "project": garden.ID},
		{"title": "Mend the fence", "project": garden.ID},
	} {
		postJSON(t, h, http.MethodPost, "/api/task", task)
	}
	a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{"component": "collection", "props": map[string]any{"type": "project", "label": "Projects"}}))
	const says = "3 tasks, 1 done"

	// Its row, as short words at the row's right.
	if row, _ := findRow(get(t, h, "/t/project").Body.String(), garden.ID); !strings.Contains(row, `<span class="sw-row__note">`+says+`</span>`) {
		t.Errorf("its row does not say %q:\n%s", says, row)
	}
	// Its page, once, as a chip in the lede and nowhere else as a count.
	page := get(t, h, "/t/project/"+garden.ID).Body.String()
	doc, err := htmltest.Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	ledes := doc.WithAttr("class", "sw-lede")
	if len(ledes) != 1 || !strings.Contains(htmltest.VisibleText(ledes[0]), says) {
		t.Errorf("its page's lede does not say %q", says)
	}
	if n := strings.Count(read(page), "3 tasks"); n != 1 {
		t.Errorf("its page says the count %d times, want once:\n%s", n, read(page))
	}
	// A list on the canvas.
	if canvas := read(get(t, h, "/").Body.String()); !strings.Contains(canvas, says) {
		t.Errorf("the canvas's list does not say %q:\n%s", says, canvas)
	}
	// The API's glance, which get_record gives too.
	var view struct{ Glance string }
	json.Unmarshal(get(t, h, "/api/project/"+garden.ID).Body.Bytes(), &view)
	if view.Glance != says {
		t.Errorf("the API's glance is %q, want %q", view.Glance, says)
	}
	// look's text, which reads the page as a person does.
	var look struct {
		Outline struct {
			Text []struct{ Text string } `json:"text"`
		} `json:"outline"`
	}
	json.Unmarshal(get(t, h, "/api/look?only=text&path=/t/project/"+garden.ID).Body.Bytes(), &look)
	found := false
	for _, p := range look.Outline.Text {
		found = found || strings.Contains(p.Text, says)
	}
	if !found {
		t.Errorf("look's text does not say %q: %+v", says, look.Outline.Text)
	}
}

// Which records are in which is read from the schema, by no type's name:
// a ref marked listed, the same one a page lists by (backrefs.go). A
// task's For names who it is for, not what it is in, so a person counts
// their interactions and not the tasks for them; a meeting counts the
// tasks that came up at it; a habit's entries are its tracker's to say.
func TestWhatIsInARecordIsReadFromTheSchema(t *testing.T) {
	_, h := newApp(t)
	var ana, standup struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/person", map[string]any{"name": "Ana Silva"}), &ana)
	decode(t, postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "Standup", "starts": "2026-10-01T09:00:00Z"}), &standup)
	postJSON(t, h, http.MethodPost, "/api/interaction", map[string]any{"summary": "Called about the fence", "person": ana.ID})
	postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Send the quote", "for": ana.ID, "event": standup.ID})
	glance := func(path string) string {
		var v struct{ Glance string }
		json.Unmarshal(get(t, h, path).Body.Bytes(), &v)
		return v.Glance
	}
	if got := glance("/api/person/" + ana.ID); got != "1 interaction" {
		t.Errorf("a person says %q, want %q (a task for them is not in them)", got, "1 interaction")
	}
	if got := glance("/api/event/" + standup.ID); !strings.HasSuffix(got, " · 1 task") {
		t.Errorf("a meeting says %q, want the task that came up at it last", got)
	}
	var water struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/habit", map[string]any{"name": "Drink water", "unit": "glasses", "target": 8}), &water)
	postJSON(t, h, http.MethodPost, "/api/entry", map[string]any{"habit": water.ID, "amount": 2})
	if got := glance("/api/habit/" + water.ID); strings.Contains(got, "entr") {
		t.Errorf("a habit says %q; its entries are not listed, so not counted", got)
	}
	// Its row in its list says the same, counted with the rest of the list.
	if body := get(t, h, "/t/person").Body.String(); !strings.Contains(body, `<span class="sw-row__note">1 interaction</span>`) {
		t.Errorf("a person's row does not say what is in it:\n%s", around(body, ana.ID))
	}
}
