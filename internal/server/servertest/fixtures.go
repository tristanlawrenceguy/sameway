package servertest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// Public makes a request as the internet makes it, to the handler the
// server answers the internet with (server.Public).
func Public(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// SeedTasks makes eight tasks: six not done, due from two days ago to a
// month on, and two done.
func SeedTasks(t *testing.T, a *app.App) {
	t.Helper()
	day := func(d int) string { return time.Now().AddDate(0, 0, d).UTC().Format(time.RFC3339) }
	for _, task := range []map[string]any{
		{"title": "Dig the pond", "due": day(-2)},
		{"title": "Order compost", "due": day(2)},
		{"title": "Plant garlic", "due": day(5)},
		{"title": "Buy seeds", "due": day(30)},
		{"title": "Empty the water butt"},
		{"title": "Mend the fence", "due": day(-1)},
		{"title": "Call the dentist", "due": day(-3), "done": true},
		{"title": "Wash the car", "due": day(3), "done": true},
	} {
		if _, err := a.Store.Create("task", task); err != nil {
			t.Fatal(err)
		}
	}
}

// AddCollection puts a collection on the canvas and says its block's id.
func AddCollection(t *testing.T, h http.Handler, props map[string]any) string {
	t.Helper()
	rec := PostJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": props})
	WantStatus(t, rec, http.StatusCreated)
	var out map[string]any
	Decode(t, rec, &out)
	id, _ := out["id"].(string)
	if id == "" {
		t.Fatalf("no block id: %s", rec.Body.String())
	}
	return id
}
