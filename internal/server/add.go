package server

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

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
		string(s.component("button", map[string]any{"label": addLabel(t.Name), "type": "submit", "variant": "secondary"})) + `</form>`)
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
	act := s.record(r, chat.Change{Action: "created", Component: t.Name, ID: rec.ID, Detail: "New " + t.Name, Href: list + "/" + rec.ID})
	// #edit opens the editor on arrival (09-edit-fields.js); added names the
	// entry that made it, so Cancel before a first Save can take it back.
	http.Redirect(w, r, list+"/"+rec.ID+"?added="+url.QueryEscape(act)+"#edit", http.StatusSeeOther)
}

// discard is Cancel on a record added a moment ago and never saved: the
// person changed their mind, so the adding is taken back, the way Undo
// would, and they return to the list with nothing new in it. A record that
// has been saved since is left alone.
func (s *Server) discard(w http.ResponseWriter, r *http.Request) {
	t, rec := r.PathValue("type"), r.PathValue("id")
	back := "/t/" + t + "/" + rec
	got, err := s.app.Store.Get(t, rec)
	entry, err2 := s.app.Store.Get(chat.ActivityType, r.FormValue("added"))
	if err != nil || err2 != nil || !got.UpdatedAt.Equal(got.CreatedAt) ||
		entry.Fields["action"] != "created" || entry.Fields["target_id"] != rec {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	if err := s.app.Chat.UndoAs("human", entry.ID); err != nil {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/t/"+t, http.StatusSeeOther)
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

// addLabel builds a button label for adding records: "Add an action" or
// "Add a note", trimmed to three words total when it would be longer.
func addLabel(name string) string {
	var label string
	if strings.HasPrefix(strings.ToLower(name), "a") ||
		strings.HasPrefix(strings.ToLower(name), "e") ||
		strings.HasPrefix(strings.ToLower(name), "i") ||
		strings.HasPrefix(strings.ToLower(name), "o") ||
		strings.HasPrefix(strings.ToLower(name), "u") {
		label = "Add an " + name
	} else {
		label = "Add a " + name
	}
	return trimLabel(label)
}

type errNoType string

func (e errNoType) Error() string { return "there is no kind of record called " + string(e) }
