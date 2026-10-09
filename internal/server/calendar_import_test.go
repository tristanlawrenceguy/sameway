package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const workCalendar = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
	"BEGIN:VEVENT\r\nUID:standup@example.com\r\nDTSTART:20261005T083000Z\r\nDTEND:20261005T084500Z\r\nRRULE:FREQ=WEEKLY;BYDAY=MO\r\nSUMMARY:Stand-up\r\nLOCATION:Room 2\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:bday\r\nDTSTART;VALUE=DATE:20261012\r\nSUMMARY:Hana's birthday\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

// A calendar brought in becomes events, each at its time and repeating as
// it did, and on the calendar; the file itself reads as its events and
// leads to bringing them in; bringing it in again adds nothing twice.
func TestACalendarBecomesEvents(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	body, ct := multipartFile(t, "Work.ics", workCalendar, nil)
	res := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	id := strings.TrimPrefix(res.Header().Get("Location"), "/t/file/")
	page := get(t, h, "/t/file/"+id).Body.String()
	if !strings.Contains(page, "Stand-up") || !strings.Contains(page, `href="/t/event/import?file=`+id+`"`) || !strings.Contains(page, "Add these events to the calendar") {
		t.Errorf("the calendar file reads as its events and leads to bringing them in:\n%.2000s", page)
	}

	res = postForm(t, h, "/t/event/import/"+id+"/run", url.Values{})
	wantStatus(t, res, http.StatusSeeOther)
	events, _ := a.Store.List("event", store.ListOptions{})
	if len(events) != 2 {
		t.Fatalf("two events, got %d", len(events))
	}
	byTitle := map[string]map[string]any{}
	for _, e := range events {
		byTitle[e.Fields["title"].(string)] = e.Fields
	}
	s := byTitle["Stand-up"]
	if s == nil || !strings.HasPrefix(s["starts"].(string), "2026-10-05T08:30:00") || s["where"] != "Room 2" || !strings.Contains(s["repeat"].(string), "FREQ=WEEKLY") || s["uid"] != "standup@example.com" {
		t.Errorf("the stand-up at its time, where, repeating, with its calendar id: %v", s)
	}
	if b := byTitle["Hana's birthday"]; b == nil || !strings.HasPrefix(b["starts"].(string), "2026-10-12") {
		t.Errorf("the birthday on its day: %v", b)
	}
	postForm(t, h, "/t/event/import/"+id+"/run", url.Values{})
	if again, _ := a.Store.List("event", store.ListOptions{}); len(again) != 2 {
		t.Errorf("bringing the same calendar in again adds nothing twice, got %d", len(again))
	}
}

// Subtitles added beside a video of the same name, with no words yet,
// become its words and its captions.
func TestSubtitlesBecomeAVideosCaptions(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	body, ct := multipartFile(t, "Garden talk.mp4", "not really a video", nil)
	video := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	body, ct = multipartFile(t, "Garden talk.srt", "1\n00:00:01,000 --> 00:00:03,000\nWelcome to the garden.\n", nil)
	do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	v, _ := a.Store.Get("file", video)
	if v.Fields["text"] != "[0:01] Welcome to the garden." || !strings.Contains(v.Fields["note"].(string), "Garden talk.srt") {
		t.Errorf("the video has the subtitles' words: %v", v.Fields)
	}
	if page := get(t, h, "/t/file/"+video).Body.String(); !strings.Contains(page, `<track kind="captions"`) {
		t.Error("and shows them as captions")
	}
}
