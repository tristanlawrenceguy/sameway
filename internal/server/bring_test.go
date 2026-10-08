package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person brings their tasks from Todoist with the file the app made: no
// question about columns, the tasks come in with their words and days,
// it says how many from where, and Undo takes them away again.
func TestThingsAreBroughtFromAnotherApp(t *testing.T) {
	a, h := newApp(t)
	if page := get(t, h, "/bring").Body.String(); !strings.Contains(page, "Todoist") || !strings.Contains(page, `name="file"`) {
		t.Fatalf("the page says how to get each export and takes the file: %s", truncate(page))
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "Inbox.csv")
	fw.Write([]byte("TYPE,CONTENT,DESCRIPTION,PRIORITY,INDENT,AUTHOR,RESPONSIBLE,DATE,DATE_LANG,TIMEZONE\ntask,Call the bank,About the card,4,1,Me,,2026-10-12,en,UTC\ntask,Water plants,,1,1,Me,,,en,UTC\n"))
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/bring", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Origin", "http://example.com")
	r.Host = "example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	page := after(t, h, w).Body.String()
	if !strings.Contains(page, "Brought in from Todoist") || !strings.Contains(page, "2 tasks") {
		t.Fatalf("it says how many from where: %d %s", w.Code, truncate(page))
	}
	tasks, _ := a.Store.List("task", store.ListOptions{})
	if len(tasks) != 2 {
		t.Fatalf("the tasks came in: %d", len(tasks))
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(log) != 1 || !strings.Contains(records.Sentence(a.Store, log[0].Fields), "2 tasks from Todoist") {
		t.Fatalf("as one change: %v", log)
	}
	postForm(t, h, "/activity/"+log[0].ID+"/undo", nil)
	if left, _ := a.Store.List("task", store.ListOptions{}); len(left) != 0 {
		t.Errorf("Undo takes them away again: %d left", len(left))
	}
}
