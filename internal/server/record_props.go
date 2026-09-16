package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	store "github.com/tristanlawrenceguy/sameway/internal/store"
)

// recordProps receives an inline edit form for a content type record. Fields
// arrive as prop-<name> values in the POST body. On success it updates the
// record and redirects back to the detail page; on validation failure it
// re-renders the detail page with 422 and error messages.
func (s *Server) recordProps(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	rec, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	fields := map[string]any{}
	for key, values := range r.PostForm {
		prop, found := strings.CutPrefix(key, "prop-")
		if !found || len(values) == 0 {
			continue
		}
		fields[prop] = strings.ReplaceAll(values[0], "\r\n", "\n")
	}

	// No fields provided — no-op redirect.
	if len(fields) == 0 {
		http.Redirect(w, r, "/t/"+t.Name+"/"+rec.ID, http.StatusSeeOther)
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
		s.renderDetailError(w, r, t, rec, err)
		return
	}

	_, err = s.app.Store.Update(t.Name, rec.ID, clean)
	if err != nil {
		s.fail(w, err)
		return
	}

	http.Redirect(w, r, "/t/"+t.Name+"/"+rec.ID, http.StatusSeeOther)
}

// renderDetailError re-renders the detail page with validation errors as a 422,
// showing each field's problem alongside the current record values so the person
// can see what they are editing and try again.
func (s *Server) renderDetailError(w http.ResponseWriter, r *http.Request, t *schema.Type, rec *store.Record, verr error) {
	var b strings.Builder

	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "warning",
		"title":   "Validation error",
		"message": "Please fix the issues below.",
	})))

	if ve, ok := verr.(*schema.ValidationError); ok {
		for field, msg := range ve.Problems {
			b.WriteString(fmt.Sprintf("<p><strong>%s</strong>: %s</p>",
				template.HTMLEscapeString(field), template.HTMLEscapeString(msg)))
		}
	}

	// Render the definition list like detailPage does, so the person can see
	// what they are editing and try again.
	b.WriteString(`<dl class="sw-dl">`)
	for _, f := range t.Fields {
		val := display(f, rec.Fields[f.Name])
		if val == "" {
			continue
		}
		fmt.Fprintf(&b, "<dt>%s</dt><dd>%s</dd>",
			template.HTMLEscapeString(label(f.Name)), template.HTMLEscapeString(val))
	}
	fmt.Fprintf(&b, "<dt>Created</dt><dd>%s</dd><dt>Updated</dt><dd>%s</dd></dl>", rec.CreatedAt.Local().Format("2006-01-02 15:04"), rec.UpdatedAt.Local().Format("2006-01-02 15:04"))

	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name + "/" + rec.ID, Status: http.StatusUnprocessableEntity})
}
