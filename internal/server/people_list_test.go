package server_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A meeting's people are records: written as ids by a tool and as names
// by a person, read as names, refused when a name is nobody or two people,
// and a person's page connects to the meetings they are in.
func TestAMeetingsPeopleAreThePeople(t *testing.T) {
	a, h := newApp(t)
	ann, _ := a.Store.Create("person", map[string]any{"name": "Ann Lee", "email": "ann@example.com"})
	ben, _ := a.Store.Create("person", map[string]any{"name": "Ben Ortiz"})
	res := postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "Stand-up", "people": "Ann Lee, ben ortiz"})
	if res.Code != http.StatusCreated {
		t.Fatalf("names are matched to people: %d %s", res.Code, res.Body.String())
	}
	var made struct{ ID string }
	json.Unmarshal(res.Body.Bytes(), &made)
	ev, _ := a.Store.Get("event", made.ID)
	if got, _ := ev.Fields["people"].([]any); len(got) != 2 || got[0] != ann.ID || got[1] != ben.ID {
		t.Fatalf("stored as ids, in order: %v", ev.Fields["people"])
	}
	if res := postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "x", "people": []string{"ann@example.com", ben.ID}}); res.Code != http.StatusCreated {
		t.Errorf("an email or an id finds the person too: %d %s", res.Code, res.Body.String())
	}
	if res := postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "x", "people": "Anne Lea"}); res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "check the spelling") {
		t.Errorf("a name that is nobody is refused, nobody made up: %d %s", res.Code, res.Body.String())
	}
	a.Store.Create("person", map[string]any{"name": "Ben Ortiz", "email": "ben2@example.com"})
	if res := postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "x", "people": "Ben Ortiz"}); res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "2 person records") {
		t.Errorf("a name two people have is refused: %d %s", res.Code, res.Body.String())
	}

	page := get(t, h, "/t/event/"+made.ID).Body.String()
	if !strings.Contains(page, "Ann Lee, Ben Ortiz") || strings.Contains(page, ann.ID+",") {
		t.Errorf("the page names them: %.2500s", page)
	}
	var person struct {
		Related []struct{ Key string } `json:"related"`
	}
	json.Unmarshal(get(t, h, "/api/person/"+ann.ID).Body.Bytes(), &person)
	found := false
	for _, r := range person.Related {
		found = found || r.Key == "points-here:event.people"
	}
	if !found {
		t.Errorf("a person is connected to the meetings they are in: %+v", person.Related)
	}
}

// A calendar's invitations bring their people: found by email, or made.
func TestACalendarsPeopleComeWithIt(t *testing.T) {
	a, h := newApp(t)
	ann, _ := a.Store.Create("person", map[string]any{"name": "Ann Lee", "email": "ann@example.com"})
	ics := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:1@x\r\nSUMMARY:Planning\r\nDTSTART:20261005T090000Z\r\nORGANIZER;CN=Ann Lee:mailto:ann@example.com\r\nATTENDEE;CN=\"Cara Diaz\";ROLE=REQ-PARTICIPANT:mailto:cara@example.com\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	body, ct := multipartFile(t, "work.ics", ics, nil)
	id := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	postForm(t, h, "/t/event/import/"+id+"/run", url.Values{})
	events, _ := a.Store.List("event", store.ListOptions{})
	if len(events) != 1 {
		t.Fatalf("one event: %d", len(events))
	}
	people, _ := events[0].Fields["people"].([]any)
	if len(people) != 2 || people[0] != ann.ID {
		t.Fatalf("the organiser found by email, the attendee after: %v", people)
	}
	cara, err := a.Store.Get("person", people[1].(string))
	if err != nil || cara.Fields["name"] != "Cara Diaz" || cara.Fields["email"] != "cara@example.com" {
		t.Errorf("a new attendee is made, with name and email: %v", cara)
	}
}
