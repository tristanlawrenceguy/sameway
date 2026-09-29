package server_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A spreadsheet is RFC 4180 with a byte-order mark, a text that looks
// like a formula never runs, and the same text comes back through Import.
func TestASpreadsheetNeverRunsWhatItHolds(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("task", map[string]any{"title": `=HYPERLINK("http://evil.example","Click")`, "notes": "one, \"two\"\nthree"})
	a.Store.Create("task", map[string]any{"title": "@SUM(A1:A2)"})
	a.Store.Create("task", map[string]any{"title": "-2+3"})
	res := get(t, h, "/export/task.csv")
	body := res.Body.String()
	for _, want := range []string{
		"\xef\xbb\xbfTitle,",
		`"'=HYPERLINK(""http://evil.example"",""Click"")"`, // quoted, quotes doubled, made inert
		"'@SUM(A1:A2)", "'-2+3",
		"\"one, \"\"two\"\"\r\nthree\"", // a comma, a quote and a new line in one cell
		"\r\n",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the spreadsheet should hold %q:\n%q", want, body)
		}
	}
	if strings.Contains(body, ",=") || strings.Contains(body, "\n=") {
		t.Errorf("no cell starts with =: %q", body)
	}
	if cd := res.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, `attachment; filename="Tasks `) || !strings.HasSuffix(cd, `.csv"`) {
		t.Errorf("a file to keep, by its name: %q", cd)
	}

	// Brought in again, the words are as they were written.
	b, hb := newApp(t)
	up, ct := multipartFile(t, "Tasks.csv", body, nil)
	fileID := strings.TrimPrefix(do(t, hb, http.MethodPost, "/t/task/import", up, ct).Header().Get("Location"), "/t/task/import?file=")
	wantStatus(t, postForm(t, hb, "/t/task/import/"+fileID+"/run", url.Values{}), http.StatusSeeOther)
	back, _ := b.Store.List("task", store.ListOptions{})
	titles := map[string]bool{}
	for _, r := range back {
		titles[r.Fields["title"].(string)] = true
	}
	if !titles[`=HYPERLINK("http://evil.example","Click")`] || !titles["@SUM(A1:A2)"] || !titles["-2+3"] {
		t.Errorf("import takes the ' off again: %v", titles)
	}
}

// A calendar says all day by the day, with the end the day after the
// last, and a repeat's last day in the kind of value its start is; read
// back, it is the same days.
func TestACalendarSaysDaysAsCalendarsDo(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("event", map[string]any{"title": "Harvest fair", "starts": "2026-10-12T00:00:00Z", "ends": "2026-10-13T00:00:00Z", "repeat": "FREQ=YEARLY;UNTIL=20300101"})
	a.Store.Create("event", map[string]any{"title": "Birthday", "starts": "2026-11-02T00:00:00Z"})
	a.Store.Create("event", map[string]any{"title": "Choir", "starts": "2026-10-06T18:00:00Z", "repeat": "FREQ=WEEKLY;BYDAY=TU;UNTIL=20261201"})
	ics := get(t, h, "/export/event.ics").Body.String()
	for _, want := range []string{"DTSTART;VALUE=DATE:20261012\r\nDTEND;VALUE=DATE:20261014\r\n", "RRULE:FREQ=YEARLY;UNTIL=20300101\r\n", "DTSTART:20261006T180000Z\r\n", "UNTIL=20261201T235959Z"} {
		if !strings.Contains(ics, want) {
			t.Errorf("the calendar should say %q:\n%s", want, ics)
		}
	}
	got := map[string]convert.Event{}
	for _, e := range convert.ParseICS([]byte(ics), nil) {
		got[e.Title] = e
	}
	if e := got["Harvest fair"]; e.Starts != "2026-10-12" || e.Ends != "2026-10-13" {
		t.Errorf("two days come back as two days: %+v", e)
	}
	if e := got["Birthday"]; e.Starts != "2026-11-02" || e.Ends != "" {
		t.Errorf("one day comes back as one day: %+v", e)
	}
}

// A record's page offers the record as files that fit it, after the
// record, each link saying the kind of file, its format and its size
// where making it is cheap.
func TestARecordGoesOutOnItsOwn(t *testing.T) {
	a, h := newApp(t)
	task, _ := a.Store.Create("task", map[string]any{"title": "Dig the pond", "due": "2026-10-01T00:00:00Z", "notes": "Mind the roots."})
	undated, _ := a.Store.Create("task", map[string]any{"title": "Someday"})
	hana, _ := a.Store.Create("person", map[string]any{"name": "Hana Sato", "email": "hana@example.com"})

	page := get(t, h, "/t/task/"+task.ID).Body.String()
	sized := regexp.MustCompile(`href="/export/task/` + task.ID + `\.(md|docx|ics)" download data-format="(md|docx|ics)">(Text \(Markdown|Word \(DOCX|Calendar \(ICS), \d+ (bytes|KB)\)<span class="sw-visually-hidden">, this task</span>`)
	if n := len(sized.FindAllString(page, -1)); n != 3 {
		t.Errorf("the task's page offers it as Markdown, Word and a calendar entry, each with its size; found %d\n%s", n, page)
	}
	if !strings.Contains(page, `data-format="pdf">Printable (PDF)<`) || !strings.Contains(page, `data-format="html">Web page (HTML)<`) {
		t.Error("the web page and the PDF, made by rendering, say no size rather than guess")
	}
	if strings.Index(page, `data-component="export"`) < strings.Index(page, "Mind the roots.") {
		t.Error("the way out comes after the record")
	}
	if strings.Contains(get(t, h, "/t/task/"+undated.ID).Body.String(), ".ics\"") {
		t.Error("a task with no day is not offered as a calendar entry")
	}
	if !strings.Contains(get(t, h, "/t/person/"+hana.ID).Body.String(), `/export/person/`+hana.ID+`.vcf`) {
		t.Error("a person is offered as a contact")
	}

	md := get(t, h, "/export/task/"+task.ID+".md")
	if md.Header().Get("Content-Type") != "text/markdown; charset=utf-8" || md.Header().Get("Content-Disposition") != `attachment; filename="Dig the pond.md"` {
		t.Errorf("a Markdown file by the record's name: %q", md.Header())
	}
	if ics := get(t, h, "/export/task/"+task.ID+".ics").Body.String(); !strings.Contains(ics, "SUMMARY:Dig the pond\r\nDTSTART;VALUE=DATE:20261001\r\n") {
		t.Errorf("a calendar entry on its day:\n%s", ics)
	}
	if res := get(t, h, "/export/task/"+undated.ID+".ics"); res.Code != http.StatusNotFound {
		t.Errorf("no calendar entry without a day: %d", res.Code)
	}
	if res := get(t, h, "/export/task/"+task.ID+".vcf"); res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), ".md, .html, .docx, .pdf, .ics") {
		t.Errorf("a format that does not fit says which do: %q", res.Body.String())
	}
	if res := get(t, h, "/export/message/x.md"); res.Code != http.StatusNotFound {
		t.Errorf("the conversation does not go out: %d", res.Code)
	}

	// A name beyond ASCII is kept whole for the browsers that read it.
	cafe, _ := a.Store.Create("note", map[string]any{"title": "Café plans"})
	if cd := get(t, h, "/export/note/"+cafe.ID+".md").Header().Get("Content-Disposition"); cd != `attachment; filename="Caf_ plans.md"; filename*=UTF-8''Caf%C3%A9%20plans.md` {
		t.Errorf("filename and filename*: %q", cd)
	}
}

// The internet takes away what is published, and only that: a published
// list and record as files, without the fields kept out of sight.
func TestTheInternetTakesAwayOnlyWhatIsPublished(t *testing.T) {
	a, h := newApp(t)
	pub := h.(*server.Server).Public(nil)
	ev, _ := a.Store.Create("event", map[string]any{"title": "Open garden", "starts": "2026-10-10T00:00:00Z", "uid": "private-calendar-id"})
	task, _ := a.Store.Create("task", map[string]any{"title": "Secret errand"})
	a.Workspace.Config.Publish.Types = "event"

	list := public(t, pub, http.MethodGet, "/t/event", "").Body.String()
	if !strings.Contains(list, `href="/export/event.csv"`) || !strings.Contains(list, `href="/export/event.ics"`) {
		t.Errorf("a published list offers its files, still as links:\n%s", list)
	}
	if r := public(t, pub, http.MethodGet, "/export/event.ics", ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "Open garden") {
		t.Errorf("a published list goes out: %d", r.Code)
	}
	md := public(t, pub, http.MethodGet, "/export/event/"+ev.ID+".md", "")
	if md.Code != http.StatusOK || !strings.Contains(md.Body.String(), "Open garden") || strings.Contains(md.Body.String(), "private-calendar-id") {
		t.Errorf("a published record goes out without its hidden fields: %d\n%s", md.Code, md.Body.String())
	}
	if ics := public(t, pub, http.MethodGet, "/export/event/"+ev.ID+".ics", "").Body.String(); !strings.Contains(ics, "Open garden") || strings.Contains(ics, "private-calendar-id") {
		t.Errorf("and as a calendar entry, without them either:\n%s", ics)
	}
	for _, path := range []string{"/export/task.csv", "/export/task/" + task.ID + ".md", "/export/workspace.zip", "/export/all.ics"} {
		if r := public(t, pub, http.MethodGet, path, ""); r.Code != http.StatusNotFound || strings.Contains(r.Body.String(), "Secret errand") {
			t.Errorf("%s is not published: %d", path, r.Code)
		}
	}
}
