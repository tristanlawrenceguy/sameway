package servertest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
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
	day := func(d int) string { return time.Now().AddDate(0, 0, d).UTC().Format(time.RFC3339) } // wall clock: records dated around the real today, for apps on the real clock
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

// Written writes a record as every way in does, through records and
// logged as the person's, so what waits on a change (an automation,
// sorting) hears it; the store itself tells nobody. action is created,
// updated or deleted.
func Written(t *testing.T, st *store.Store, action, typ, id string, fields map[string]any) *store.Record {
	t.Helper()
	rec, _, err := records.WriteAs(st, records.Who{Actor: "human"}, action, typ, id, fields)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// NewestEntry is the newest entry in the activity log.
func NewestEntry(t *testing.T, st *store.Store) *store.Record {
	t.Helper()
	recs, err := st.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if err != nil || len(recs) == 0 {
		t.Fatalf("the log should have an entry: %v", err)
	}
	return recs[0]
}
