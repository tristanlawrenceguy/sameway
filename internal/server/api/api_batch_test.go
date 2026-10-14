package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// POST /api/{type} with a list writes every record or none, as one change
// in the log; one refused names its place and writes nothing; PATCH
// /api/{type} makes the same change to several; blocks stay one at a time.
func TestSeveralRecordsInOneRequest(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	send := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	before, _ := a.Store.List("activity", store.ListOptions{})
	res := send(http.MethodPost, "/api/task", `[{"title": "Email Sarah"}, {"title": "Book the car"}, {"title": "Call mum"}]`)
	var made struct {
		Count   int `json:"count"`
		Records []struct {
			ID, Title, Page string
		} `json:"records"`
	}
	json.Unmarshal(res.Body.Bytes(), &made)
	if res.Code != http.StatusCreated || made.Count != 3 || made.Records[1].Title != "Book the car" || !strings.HasPrefix(made.Records[0].Page, "/t/task/") {
		t.Fatalf("three made, each with its page: %d %s", res.Code, res.Body)
	}
	after, _ := a.Store.List("activity", store.ListOptions{})
	if len(after)-len(before) != 1 {
		t.Errorf("one change in the log, got %d", len(after)-len(before))
	}

	res = send(http.MethodPost, "/api/task", `[{"title": "Fine"}, {"title": "Wrong", "status": "someday"}]`)
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), `"[1].status"`) {
		t.Errorf("the one refused is named by its place: %d %s", res.Code, res.Body)
	}
	if tasks, _ := a.Store.List("task", store.ListOptions{}); len(tasks) != 3 {
		t.Errorf("nothing of a refused list is written: %d tasks", len(tasks))
	}

	ids, _ := json.Marshal([]string{made.Records[0].ID, made.Records[2].ID})
	res = send(http.MethodPatch, "/api/task", `{"ids": `+string(ids)+`, "fields": {"due": "2026-10-19"}}`)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"count": 2`) {
		t.Fatalf("the same change to two: %d %s", res.Code, res.Body)
	}
	if rec, _ := a.Store.Get("task", made.Records[2].ID); !strings.HasPrefix(rec.Fields["due"].(string), "2026-10-19") {
		t.Errorf("Call mum moved: %v", rec.Fields["due"])
	}

	if res := send(http.MethodPost, "/api/block", `[{"component": "heading", "props": {"text": "A"}}]`); res.Code < 400 || !strings.Contains(res.Body.String(), "one at a time") {
		t.Errorf("blocks are written one at a time: %d %s", res.Code, res.Body)
	}
}
