package server_test

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What is on a list page goes out as a file in the formats it can come in
// with, and comes back in as it was: tasks as a spreadsheet, events as a
// calendar, people as contacts; the system's own records never go out.
func TestWhatComesInGoesOutAgain(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("task", map[string]any{"title": "Order compost", "due": "2026-10-05T09:30:00Z", "repeat": "FREQ=WEEKLY;BYDAY=MO", "tags": []any{"garden", "soil"}})
	a.Store.Create("task", map[string]any{"title": "Dig the pond", "done": true})
	a.Store.Create("event", map[string]any{"title": "Stand-up, daily", "starts": "2026-10-05T08:30:00Z", "ends": "2026-10-05T08:45:00Z", "repeat": "FREQ=WEEKLY;BYDAY=MO", "where": "Room 2; upstairs"})
	a.Store.Create("event", map[string]any{"title": "Hana's birthday", "starts": "2026-10-12T00:00:00Z"})
	a.Store.Create("person", map[string]any{"name": "Hana Sato", "email": "hana@example.com", "phone": "+44 20 7946 0000", "organisation": "Garden Co"})

	// The list page offers what fits, with its own query.
	page := get(t, h, "/t/task?where=done%3Dfalse").Body.String()
	if !strings.Contains(page, `href="/export/task.csv?where=done%3Dfalse" download data-format="csv">Spreadsheet (CSV, `) || !strings.Contains(page, "/export/task.ics") || strings.Contains(page, "/export/task.vcf") {
		t.Errorf("the task list offers a spreadsheet and a calendar of what it shows:\n%.1500s", page)
	}
	if people := get(t, h, "/t/person").Body.String(); !strings.Contains(people, "/export/person.vcf") {
		t.Error("people go out as contacts")
	}

	// A spreadsheet of what the page shows, which Excel reads with accents.
	csv := get(t, h, "/export/task.csv?where=done%3Dfalse")
	body := csv.Body.String()
	if csv.Header().Get("Content-Type") != "text/csv; charset=utf-8" || !strings.HasPrefix(body, "\xef\xbb\xbfTitle,") || !strings.Contains(csv.Header().Get("Content-Disposition"), `attachment; filename="Tasks `) {
		t.Fatalf("a CSV download: %q %q", csv.Header(), body)
	}
	if !strings.Contains(body, "Order compost") || strings.Contains(body, "Dig the pond") || !strings.Contains(body, `"garden, soil"`) || !strings.Contains(body, "every Monday") {
		t.Errorf("only the tasks the page shows, as a person reads them: %q", body)
	}

	// Brought into another workspace, the spreadsheet is the same tasks.
	b, hb := newApp(t)
	all := get(t, h, "/export/task.csv").Body.String()
	up, ct := multipartFile(t, "Tasks.csv", all, nil)
	fileID := strings.TrimPrefix(do(t, hb, http.MethodPost, "/t/task/import", up, ct).Header().Get("Location"), "/t/task/import?file=")
	wantStatus(t, postForm(t, hb, "/t/task/import/"+fileID+"/run", url.Values{}), http.StatusSeeOther)
	back, _ := b.Store.List("task", store.ListOptions{})
	got := map[string]map[string]any{}
	for _, r := range back {
		got[r.Fields["title"].(string)] = r.Fields
	}
	if c := got["Order compost"]; c == nil || !strings.Contains(c["repeat"].(string), "FREQ=WEEKLY") || len(c["tags"].([]any)) != 2 || c["due"] == nil {
		t.Errorf("the compost task comes back whole: %v", c)
	}
	if d := got["Dig the pond"]; d == nil || d["done"] != true {
		t.Errorf("done comes back done: %v", d)
	}

	// A calendar, which a calendar app can take or subscribe to, and which
	// comes back in with nothing added twice.
	ics := get(t, h, "/export/event.ics").Body.String()
	for _, want := range []string{"BEGIN:VCALENDAR\r\n", "SUMMARY:Stand-up\\, daily\r\n", "DTSTART:20261005T083000Z\r\n", "DTEND:20261005T084500Z\r\n", "RRULE:FREQ=WEEKLY;BYDAY=MO\r\n", "LOCATION:Room 2\\; upstairs\r\n", "DTSTART;VALUE=DATE:20261012\r\n"} {
		if !strings.Contains(ics, want) {
			t.Errorf("the calendar should have %q:\n%s", want, ics)
		}
	}
	up, ct = multipartFile(t, "Events.ics", ics, nil)
	fileID = strings.TrimPrefix(do(t, hb, http.MethodPost, "/t/event/import", up, ct).Header().Get("Location"), "/t/event/import?file=")
	postForm(t, hb, "/t/event/import/"+fileID+"/run", url.Values{})
	postForm(t, hb, "/t/event/import/"+fileID+"/run", url.Values{})
	events, _ := b.Store.List("event", store.ListOptions{})
	if len(events) != 2 {
		t.Errorf("two events come back, once each, got %d", len(events))
	}

	// Contacts every address book reads.
	vcf := get(t, h, "/export/person.vcf").Body.String()
	for _, want := range []string{"BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Hana Sato\r\nN:Sato;Hana;;;\r\n", "EMAIL;TYPE=INTERNET:hana@example.com\r\n", "TEL:+44 20 7946 0000\r\n", "ORG:Garden Co\r\n"} {
		if !strings.Contains(vcf, want) {
			t.Errorf("the contacts should have %q:\n%s", want, vcf)
		}
	}

	// An Excel workbook, its header first.
	x, err := excelize.OpenReader(bytes.NewReader(get(t, h, "/export/task.xlsx").Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := x.GetRows("Task")
	if len(rows) != 3 || rows[0][0] != "Title" {
		t.Errorf("a sheet with a header and two tasks: %v", rows)
	}

	// The system's own records, and a format that does not fit, do not go out.
	if res := get(t, h, "/export/message.csv"); res.Code != http.StatusNotFound {
		t.Errorf("the conversation is not taken out: %d", res.Code)
	}
	if res := get(t, h, "/export/task.vcf"); res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "it can be .csv, .xlsx, .ics") {
		t.Errorf("a format that does not fit says which do: %q", res.Body.String())
	}
}

// A recording's words go out as subtitles and as text, from its text.
func TestATranscriptGoesOutAsSubtitlesAndText(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	body, ct := multipartFile(t, "Talk.m4a", "not really audio", nil)
	id := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	f, _ := a.Store.Get("file", id)
	stored := f.Fields["path"].(string)
	os.WriteFile(filepath.Join(a.Workspace.FilesDir(), strings.TrimSuffix(stored, filepath.Ext(stored))+".vtt"), []byte("WEBVTT\n\n00:00:01.000 --> 00:00:03.000\n<v Hana>Welcome.\n"), 0o644)
	if page := get(t, h, "/t/file/"+id).Body.String(); !strings.Contains(page, `href="/files/`+id+`/transcript.srt" download data-format="srt">Subtitles (SRT`) {
		t.Error("the page offers the transcript to take away")
	}
	srt := get(t, h, "/files/"+id+"/transcript.srt").Body.String()
	if srt != "1\n00:00:01,000 --> 00:00:03,000\nHana: Welcome.\n\n" {
		t.Errorf("subtitles: %q", srt)
	}
	if txt := get(t, h, "/files/"+id+"/transcript.txt").Body.String(); txt != "[0:01] **Hana:** Welcome.\n" {
		t.Errorf("text: %q", txt)
	}
}
