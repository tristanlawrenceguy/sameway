package server

import (
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A person can make a record by hand. Everything else about a record is
// edited where it is (editfields.go), so making one is a single press: a
// new record with a name to change, open in its editor on its own page.
// Before this, an empty list could only say "ask the assistant", which on
// a first run, with no model connected yet, led nowhere.

// addButton is the press that makes a new record of t, quiet under the
// list, beside the way to import them.
func (s *Server) addButton(t *schema.Type) template.HTML {
	if t.Internal || t.Name == FileType {
		return ""
	}
	return template.HTML(`<form method="post" action="/t/` + template.HTMLEscapeString(t.Name) + `/add" class="sw-add">` +
		string(s.component("button", map[string]any{"label": "Add a " + t.Name, "type": "submit", "variant": "secondary"})) + `</form>`)
}

// addRecord makes a new record with its name to change, and opens it for
// editing. What the type requires beyond a name is the person's to fill
// in; if the type will not take a record with a name alone, it says why.
func (s *Server) addRecord(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	list := "/t/" + r.PathValue("type")
	if !ok || t.Internal {
		s.failed(w, r, "Not added", errNoType(r.PathValue("type")), "/")
		return
	}
	fields := map[string]any{}
	if name := titleField(t); name != "" {
		fields[name] = "New " + t.Name
	}
	rec, err := s.app.Store.Create(t.Name, fields)
	if err != nil {
		s.failed(w, r, "Not added", err, list)
		return
	}
	s.record(r, chat.Change{Action: "created", Component: t.Name, ID: rec.ID, Detail: "New " + t.Name, Href: list + "/" + rec.ID})
	// #edit opens the editor on arrival (09-edit-fields.js).
	http.Redirect(w, r, list+"/"+rec.ID+"#edit", http.StatusSeeOther)
}

// titleField is the field a record of t is named by.
func titleField(t *schema.Type) string {
	if t.Title != "" {
		return t.Title
	}
	for _, f := range t.Fields {
		if f.Type == "string" {
			return f.Name
		}
	}
	return ""
}

type errNoType string

func (e errNoType) Error() string { return "there is no kind of record called " + string(e) }
