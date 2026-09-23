package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// A record's state is one press away wherever the record shows: a task in
// a list has Mark as done, pressing it works without JavaScript, comes
// back to the page it was pressed on, glows, and can be undone.
func TestARecordsStateIsOnePressAway(t *testing.T) {
	a, h := newApp(t)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost"}), &task)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "record", "props": map[string]any{"type": "task", "record": task.ID}}), http.StatusCreated)

	canvas := get(t, h, "/").Body.String()
	form := `<form class="sw-mark" method="post" action="/t/task/` + task.ID + `/props" data-component="mark" data-record-type="task" data-record-id="` + task.ID + `" data-field="done"><label class="sw-mark__label"><input class="sw-mark__input" type="checkbox" name="prop-done" value="true" aria-label="Mark done Order compost"> Done<span class="sw-visually-hidden"> Order compost</span></label><input type="hidden" name="prop-done" value="false">`
	if strings.Count(canvas, form) != 2 {
		t.Errorf("the collection item and the record block each offer the checkbox, named with the record: %.600s", canvas[strings.Index(canvas, "Tasks"):])
	}
	if page := get(t, h, "/t/task/"+task.ID).Body.String(); !strings.Contains(page, `<form class="sw-mark" method="post" action="/t/task/`+task.ID+`/props" data-component="mark"`) {
		t.Error("the record's own page offers the press beside Delete")
	}

	// Pressing it from the canvas marks the task done and comes back.
	req := httptest.NewRequest(http.MethodPost, "/t/task/"+task.ID+"/props", strings.NewReader(url.Values{"prop-done": {"true"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusSeeOther)
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/") {
		t.Errorf("the press should come back to the canvas, got %q", loc)
	}
	done, _ := a.Store.Get("task", task.ID)
	if done.Fields["done"] != true {
		t.Errorf("the task should be done, got %v", done.Fields["done"])
	}
	after := get(t, h, "/").Body.String()
	if !strings.Contains(after, `name="prop-done" value="true" aria-label=`) || !strings.Contains(after, `checked> Done`) {
		t.Error("the checkbox now shows the fact")
	}
	if !strings.Contains(after, `data-changed=`) {
		t.Error("the change glows like any other")
	}

	// It is in the log with what it was, so it can be undone.
	var log struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/activity"), &log)
	if len(log.Records) == 0 || log.Records[0].Fields["action"] != "updated" || log.Records[0].Fields["actor"] != "human" {
		t.Fatalf("a press is a human update in the log, got %+v", log.Records)
	}
	wantStatus(t, postForm(t, h, "/activity/"+log.Records[0].ID+"/undo", url.Values{"from": {"/"}}), http.StatusSeeOther)
	undone, _ := a.Store.Get("task", task.ID)
	if undone.Fields["done"] != false {
		t.Errorf("undo should put the task back to not done, got %v", undone.Fields["done"])
	}

	// A type with no yes-or-no field offers nothing.
	var project struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &project)
	if page := get(t, h, "/t/project/"+project.ID).Body.String(); strings.Contains(page, `data-component="mark"`) {
		t.Error("a project has nothing to press")
	}
}
