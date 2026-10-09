package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A calendar link pasted on Calendars brings its events in and keeps them
// in step: what changed there changes here, what went there goes here, and
// an event the person made is never touched.
func TestACalendarLinkKeepsItsEventsInStep(t *testing.T) {
	t.Parallel()
	feed := ics("a1", "Standup", "20261012T090000Z") + ics("a2", "Review", "20261013T140000Z")
	cal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n"+feed+"END:VCALENDAR\r\n")
	}))
	defer cal.Close()
	a, h := newApp(t)
	if _, err := a.Store.Create("event", map[string]any{"title": "Dentist", "starts": "2026-10-14 10:00"}); err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/calendars").Body.String(); !strings.Contains(page, "Secret address in iCal format") {
		t.Fatalf("the page says where each calendar gives its link: %s", truncate(page))
	}
	page := after(t, h, postForm(t, h, "/calendars/add", url.Values{"url": {cal.URL + "/work.ics"}, "name": {"Work"}})).Body.String()
	if !strings.Contains(page, "Work is kept in step") || !strings.Contains(page, "2 events") {
		t.Fatalf("its events come in: %s", truncate(page))
	}
	titles := func() string {
		recs, _ := a.Store.List("event", store.ListOptions{OrderBy: "title"})
		var out []string
		for _, r := range recs {
			out = append(out, r.Fields["title"].(string))
		}
		return strings.Join(out, ",")
	}
	if got := titles(); got != "Dentist,Review,Standup" {
		t.Fatalf("both came, beside the person's own: %s", got)
	}
	feed = ics("a1", "Daily standup", "20261012T093000Z")
	postForm(t, h, "/calendars/add", url.Values{"url": {cal.URL + "/work.ics"}, "name": {"Again"}}) // a second link to the same calendar syncs it too
	if got := titles(); !strings.Contains(got, "Daily standup") || !strings.Contains(got, "Dentist") {
		t.Errorf("a change there is here, the person's own untouched: %s", got)
	}
}

func ics(uid, title, start string) string {
	return "BEGIN:VEVENT\r\nUID:" + uid + "\r\nSUMMARY:" + title + "\r\nDTSTART:" + start + "\r\nEND:VEVENT\r\n"
}
