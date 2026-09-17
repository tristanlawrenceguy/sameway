package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A chart from records counts or sums them by a field or by a period of
// a date, in an order a person expects, with the numbers as a table too;
// a wrong field is said in words.
func TestAChartCountsRecords(t *testing.T) {
	_, h := newApp(t)
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	for _, task := range []map[string]any{
		{"title": "a", "due": "2026-08-03T00:00:00Z", "done": true, "project": garden.ID},
		{"title": "b", "due": "2026-08-20T00:00:00Z", "done": true, "project": garden.ID},
		{"title": "c", "due": "2026-09-02T00:00:00Z", "done": true},
		{"title": "d", "due": "2026-09-09T00:00:00Z"},
	} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", task), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "chart", "props": map[string]any{"type": "task", "by": "due", "period": "month", "where": []string{"done=true"}, "detail": "page", "id": "done-by-month"},
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "chart", "props": map[string]any{"type": "task", "by": "project", "caption": "Tasks by project"},
	}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "chart", "props": map[string]any{"type": "task", "by": "owner"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()
	for _, want := range []string{
		`<figcaption class="sw-chart__caption" id="done-by-month-caption">How many tasks by Due</figcaption>`,
		`<th scope="row">2026-08</th><td>2</td></tr><tr><th scope="row">2026-09</th><td>1</td>`,
		`role="img" aria-labelledby="done-by-month-caption"`, `<rect class="sw-chart__bar"`,
		`<th scope="row">Garden</th><td>2</td>`, `<th scope="row">(none)</th><td>2</td>`, `<summary class="sw-pressable">Numbers</summary>`,
		`task has no field &#34;owner&#34; to group by`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the canvas should carry %s", want)
		}
	}
	// Months come in order, and the date table is open at page detail.
	if strings.Index(page, `<th scope="row">2026-08</th>`) > strings.Index(page, `<th scope="row">2026-09</th>`) {
		t.Error("months should be in order")
	}
}
