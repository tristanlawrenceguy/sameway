package agenda_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// A freelancer's project due today is under Deadlines, a call's follow-up
// due today under Follow up, and either one past its day with what is
// overdue; a done project and a follow-up dealt with leave the page. The
// person's page counts the projects for them, and an agent reading
// /api/today gets the same lists.
func TestTodayHasDeadlinesAndFollowUps(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	client, _ := a.Store.Create("person", map[string]any{"name": "Acme Ltd"})
	a.Store.Create("project", map[string]any{"title": "Acme website", "due": day(0), "for": client.ID})
	a.Store.Create("project", map[string]any{"title": "Acme logo", "due": day(-2), "for": client.ID})
	a.Store.Create("project", map[string]any{"title": "Old brochure", "due": day(-5), "status": "done"})
	call, _ := a.Store.Create("interaction", map[string]any{"summary": "Call with Priya about the quote", "kind": "call", "follow_up": day(0)})
	a.Store.Create("interaction", map[string]any{"summary": "Email from the landlord", "kind": "email", "follow_up": day(-1)})

	page := get(t, h, "/today").Body.String()
	section := func(h2 string) string {
		i := strings.Index(page, "<h2>"+h2+"</h2>")
		if i < 0 {
			t.Fatalf("no %s on Today: %s", h2, truncate(page))
		}
		rest := page[i:]
		return rest[:strings.Index(rest, "</ul>")]
	}
	if s := section("Deadlines"); !strings.Contains(s, "Acme website") {
		t.Errorf("a project due today is a deadline: %s", s)
	}
	follow := section("Follow up")
	if !strings.Contains(follow, "Call with Priya") || !strings.Contains(follow, "Followed up") {
		t.Errorf("a follow-up due today is under Follow up, with its press: %s", follow)
	}
	late := section("Overdue")
	for _, want := range []string{"Acme logo", "Email from the landlord"} {
		if !strings.Contains(late, want) {
			t.Errorf("%s is past its day, so overdue: %s", want, late)
		}
	}
	if strings.Contains(page, "Old brochure") {
		t.Error("a done project is not on Today")
	}

	var api struct {
		Deadlines []struct{ Title string } `json:"deadlines"`
		FollowUps []struct{ Title string } `json:"follow_ups"`
	}
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &api)
	if len(api.Deadlines) != 1 || len(api.FollowUps) != 1 {
		t.Errorf("an agent reads the same lists: %+v", api)
	}

	postForm(t, h, "/today/followed-up", url.Values{"id": {call.ID}})
	if got, _ := a.Store.Get("interaction", call.ID); got.Fields["follow_up"] != nil && got.Fields["follow_up"] != "" {
		t.Errorf("followed up, the day is cleared: %v", got.Fields["follow_up"])
	}
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Call with Priya") {
		t.Error("a follow-up dealt with leaves Today")
	}
	if p := get(t, h, "/t/person/"+client.ID).Body.String(); !strings.Contains(p, "2 projects") {
		t.Errorf("the client's page counts their projects: %s", truncate(p))
	}
}
