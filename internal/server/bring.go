package server

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
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

func (s *Server) bringPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<p>Tasks and notes from another app come in with their titles, words, days, tags and what is done. Undo takes them away again.</p><h2>Getting the file</h2><dl class="sw-stack">`)
	for _, h := range bringHow {
		b.WriteString(`<dt><strong>` + template.HTMLEscapeString(h.app) + `</strong></dt><dd>` + template.HTMLEscapeString(h.how) + `</dd>`)
	}
	b.WriteString(`</dl><h2>Bringing it in</h2><form method="post" action="/bring" enctype="multipart/form-data" class="sw-stack">`)
	b.WriteString(`<label for="bring-file">The file from the app</label><input type="file" id="bring-file" name="file" accept=".csv,.json,.enex,.zip" required>`)
	b.WriteString(string(s.component("button", map[string]any{"label": "Bring them in", "type": "submit"})) + `</form>`)
	s.page(w, r, "Bring your things", template.HTML(b.String()), pageOptions{Lede: "From Todoist, Google, Evernote, Notion or Obsidian."})
}

func (s *Server) bring(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, bringLimit)
	f, hdr, err := r.FormFile("file")
	if err != nil {
		s.failed(w, r, "Nothing brought in", errors.New("choose the file the app made"), "/bring")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		s.failed(w, r, "Nothing brought in", err, "/bring")
		return
	}
	got, err := ingest.Recognise(hdr.Filename, data)
	if err != nil {
		s.failed(w, r, "Nothing brought in", err, "/bring")
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
		chat.Record(s.app.Store, "human", chat.Change{Action: "imported", Component: t.Name,
			Detail: fmt.Sprintf("%d %s from %s", report.Made, schema.Plural(t.Name), got.From), Href: "/t/" + t.Name, Before: chat.Imported(t.Name, report.IDs)})
		said = append(said, fmt.Sprintf("%d %s", report.Made, schema.Plural(t.Name)))
		if to == "/" {
			to = "/t/" + t.Name
		}
	}
	if len(said) == 0 {
		s.failed(w, r, "Nothing brought in", errors.New("the file from "+got.From+" had nothing in it to bring"), "/bring")
		return
	}
	s.tellAt(w, r, outcome{Title: "Brought in from " + got.From, Text: strings.Join(said, " and ") + ". Undo in Activity takes each kind away again."}, to)
}

func (s *Server) bringRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /bring", s.bringPage)
	m.HandleFunc("POST /bring", s.bring)
}

// importLinks are a list's ways in from a file: Import for any file, and
// for tasks and notes, the apps Bring your things knows.
func (s *Server) importLinks(t *schema.Type) string {
	if !s.importable(t) {
		return ""
	}
	out := `<p class="sw-quiet-row">` + string(s.component("link", map[string]any{"href": "/t/" + t.Name + "/import", "label": "Import", "context": schema.Plural(t.Name), "look": "button"}))
	if t.Name == "task" || t.Name == "note" {
		out += " " + string(s.component("link", map[string]any{"href": "/bring", "label": "Bring yours from another app"}))
	}
	return out + `</p>`
}
