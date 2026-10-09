package exchange

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Bring your things: one page for what a person has in another app. It
// says in a line for each app how to get its export, takes the file, and
// brings in what it holds (ingest.Recognise), tasks and notes, each kind
// as one change with its Undo, saying how many came from where. Nothing is
// asked about columns: the exports are known.

const bringLimit = 200 << 20

// bringHow is how to get each export, in the words of the app's menus.
var bringHow = []struct{ app, how string }{
	{"Todoist", "Settings, then Backups, or a project's menu and Export as a CSV file."},
	{"Google Tasks or Google Keep", "takeout.google.com: choose Tasks or Keep, then Export once. Bring the zip as it comes."},
	{"Evernote", "select notes, then File and Export notes, as an .enex file."},
	{"Notion", "Settings, then Export all workspace content, as Markdown and CSV. Bring the zip."},
	{"Obsidian, Bear or any notes in Markdown", "zip the folder they are in, and bring the zip."},
}

func (s *Service) bringPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<p>Tasks and notes from another app come in with their titles, words, days, tags and what is done. Undo takes them away again.</p><h2>Getting the file</h2><dl class="sw-stack">`)
	for _, h := range bringHow {
		b.WriteString(`<dt><strong>` + template.HTMLEscapeString(h.app) + `</strong></dt><dd>` + template.HTMLEscapeString(h.how) + `</dd>`)
	}
	b.WriteString(`</dl><h2>Bringing it in</h2>`)
	b.WriteString(string(s.Form(ui.Form{Action: "/bring", Enctype: "multipart/form-data", Class: "sw-stack",
		Body:   `<label for="bring-file">The file from the app</label><input type="file" id="bring-file" name="file" accept=".csv,.json,.enex,.zip" required>`,
		Button: &ui.Button{Label: "Bring them in"}})))
	s.Page(w, r, "Bring your things", template.HTML(b.String()), web.PageOptions{Lede: "From Todoist, Google, Evernote, Notion or Obsidian."})
}

func (s *Service) bring(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, bringLimit)
	f, hdr, err := r.FormFile("file")
	if err != nil {
		s.Failed(w, r, "Nothing brought in", errors.New("choose the file the app made"), "/bring")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		s.Failed(w, r, "Nothing brought in", err, "/bring")
		return
	}
	got, err := ingest.Recognise(hdr.Filename, data)
	if err != nil {
		s.Failed(w, r, "Nothing brought in", err, "/bring")
		return
	}
	var said []string
	to := "/"
	for _, kind := range []string{"task", "note"} {
		tb := got.Kinds[kind]
		t, ok := s.app.Types.Get(kind)
		if tb == nil || !ok {
			continue
		}
		report := ingest.Import(s.app.Store, t, tb, ingest.Guess(t, tb.Columns))
		if report.Made == 0 {
			continue
		}
		records.Record(s.app.Store, "human", records.Change{Action: "imported", Component: t.Name,
			Detail: fmt.Sprintf("%d %s from %s", report.Made, schema.Plural(t.Name), got.From), Href: "/t/" + t.Name, Ops: records.Made(s.app.Store, t.Name, report.IDs)})
		said = append(said, fmt.Sprintf("%d %s", report.Made, schema.Plural(t.Name)))
		if to == "/" {
			to = "/t/" + t.Name
		}
	}
	if len(said) == 0 {
		s.Failed(w, r, "Nothing brought in", errors.New("the file from "+got.From+" had nothing in it to bring"), "/bring")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: "Brought in from " + got.From, Text: strings.Join(said, " and ") + ". Undo in Activity takes each kind away again."}, to)
}

// ImportLinks are a list's ways in from a file: Import for any file, and
// for tasks and notes, the apps Bring your things knows.
func (s *Service) ImportLinks(t *schema.Type) string {
	if !Importable(t) {
		return ""
	}
	out := `<p class="sw-quiet-row">` + string(s.Part(ui.Link{Href: "/t/" + t.Name + "/import", Label: "Import", Context: schema.Plural(t.Name), Look: ui.LookButton}))
	if t.Name == "task" || t.Name == "note" {
		out += " " + string(s.Part(ui.Link{Href: "/bring", Label: "Bring yours from another app"}))
	}
	return out + `</p>`
}
