package server

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// returnTo is where a person goes after an edit: the page they edited from,
// when the browser says which page that was and it is one of ours, else the
// record's own page. A record edited on the canvas returns to the canvas; on
// its detail page, to the detail page.
func returnTo(r *http.Request, fallback string) string {
	ref, err := url.Parse(r.Referer())
	if err != nil || ref.Host != r.Host || !strings.HasPrefix(ref.Path, "/") {
		return fallback
	}
	return ref.RequestURI()
}

// recordProps receives an inline edit form for a content type record. Fields
// arrive as prop-<name> values in the POST body. On success it updates the
// record and redirects back to the page the edit came from; on validation
// failure it re-renders the detail page with 422 and error messages.
func (s *Server) recordProps(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	detail := "/t/" + t.Name + "/" + r.PathValue("id")
	rec, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Not saved", err, "/t/"+t.Name)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.failed(w, r, "Not saved", err, detail)
		return
	}

	fields, err := editedFields(r.PostForm)
	if err != nil {
		s.refused(w, r, t, err, detail)
		return
	}

	// No fields provided — nothing to say.
	if len(fields) == 0 {
		http.Redirect(w, r, returnTo(r, detail), http.StatusSeeOther)
		return
	}

	// Merge incoming into existing record and validate against schema.
	merged := map[string]any{}
	for k, v := range rec.Fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}

	clean, err := t.Normalize(merged)
	if err != nil {
		s.refused(w, r, t, err, detail)
		return
	}

	_, err = s.app.Store.Update(t.Name, rec.ID, clean)
	if err != nil {
		s.failed(w, r, "Not saved", err, detail)
		return
	}
	// A change a person made by hand is a change like any other: in the
	// log with what it was, so it glows where it shows and can be undone.
	undo := s.record(r, chat.Change{Action: "updated", Component: t.Name, ID: rec.ID, Detail: s.title(t, rec), Href: detail, Before: rec.Fields})
	s.tellAt(w, r, outcome{Title: "Changes saved", Undo: undo}, returnTo(r, detail))
}

// refused says why an edit was not taken, a sentence for each field in
// the order the type lists them, on the page it was made from.
func (s *Server) refused(w http.ResponseWriter, r *http.Request, t *schema.Type, err error, detail string) {
	var said []string
	if ve, ok := err.(*schema.ValidationError); ok {
		seen := map[string]bool{}
		for _, f := range t.Fields {
			if msg, ok := ve.Problems[f.Name]; ok {
				said = append(said, label(f.Name)+" "+msg+".")
				seen[f.Name] = true
			}
		}
		// A field the type does not have, named too.
		var rest []string
		for name := range ve.Problems {
			if !seen[name] {
				rest = append(rest, name)
			}
		}
		sort.Strings(rest)
		for _, name := range rest {
			said = append(said, label(name)+" "+ve.Problems[name]+".")
		}
	}
	text := strings.Join(said, " ")
	if text == "" {
		text = plainError(err)
	}
	s.tellAt(w, r, outcome{Failed: true, Title: "Not saved", Text: text}, returnTo(r, detail))
}
