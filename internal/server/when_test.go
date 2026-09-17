package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A date on a record's page reads as a person would say it and is edited
// in the same words; what is stored stays what a machine reads. Words
// nobody can read are refused with the ways that work.
func TestADateIsWrittenAsPeopleSayIt(t *testing.T) {
	a, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost", "due": "2026-09-19T00:00:00Z"}), &task)
	page := get(t, h, "/t/task/"+task.ID).Body.String()
	if !strings.Contains(page, `<dd data-prop="due" data-kind="datetime" data-source="2026-09-19T00:00:00Z">Sat 19 Sep 2026</dd>`) {
		t.Error("the page shows the day as a person reads it, with the stored value under it")
	}

	res := postForm(t, h, "/t/task/"+task.ID+"/props", url.Values{"prop-due": {"next friday 2pm"}})
	wantStatus(t, res, http.StatusSeeOther)
	ts, day, _ := when.Parse("next friday 2pm", time.Now())
	rec, err := a.Store.Get("task", task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Fields["due"] != when.Store(ts, day) || day {
		t.Errorf("next friday 2pm is stored as a moment a machine reads, got %v", rec.Fields["due"])
	}
	if page := get(t, h, "/t/task/"+task.ID).Body.String(); !strings.Contains(page, ">"+when.Text(when.Store(ts, day))+"</dd>") {
		t.Error("the page shows the moment as a person reads it")
	}

	// The API takes the same words, so an agent need not compute a date.
	wantStatus(t, postJSON(t, h, http.MethodPatch, "/api/task/"+task.ID, map[string]any{"due": "tomorrow"}), http.StatusOK)
	rec, _ = a.Store.Get("task", task.ID)
	if rec.Fields["due"] != time.Now().AddDate(0, 0, 1).Format("2006-01-02")+"T00:00:00Z" {
		t.Errorf("tomorrow is stored as that day, got %v", rec.Fields["due"])
	}

	res = postForm(t, h, "/t/task/"+task.ID+"/props", url.Values{"prop-due": {"sometime soon"}})
	wantStatus(t, res, http.StatusUnprocessableEntity)
	if body := res.Body.String(); !strings.Contains(body, "could not read &#34;sometime soon&#34;") || !strings.Contains(body, "next Friday") {
		t.Error("words nobody can read are refused with the ways that work")
	}

	// Cleared with nothing.
	wantStatus(t, postForm(t, h, "/t/task/"+task.ID+"/props", url.Values{"prop-due": {""}}), http.StatusSeeOther)
	if rec, _ = a.Store.Get("task", task.ID); rec.Fields["due"] != nil && rec.Fields["due"] != "" {
		t.Errorf("an emptied date is gone, got %v", rec.Fields["due"])
	}
}
