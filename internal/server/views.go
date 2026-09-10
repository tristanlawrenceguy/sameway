package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/sameway-dev/sameway/internal/schema"
	"github.com/sameway-dev/sameway/internal/store"
)

// listPage shows every record of a type as cards.
func (s *Server) listPage(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	recs, err := s.app.Store.List(t.Name, store.ListOptions{})
	if err != nil {
		s.fail(w, err)
		return
	}
	var b strings.Builder
	b.WriteString(`<div class="sw-cluster">`)
	b.WriteString(string(s.component("link", map[string]any{"href": "/t/" + t.Name + "/new", "label": "New " + t.Name})))
	b.WriteString(`</div>`)
	if len(recs) == 0 {
		fmt.Fprintf(&b, `<p>No %s yet.</p>`, template.HTMLEscapeString(plural(t.Name)))
	} else {
		fmt.Fprintf(&b, `<ol class="sw-stack" aria-label="%s">`, template.HTMLEscapeString(plural(t.Name)))
		for _, rec := range recs {
			props := map[string]any{"title": titleOf(t, rec), "href": "/t/" + t.Name + "/" + rec.ID, "level": 2, "meta": "Updated " + rec.UpdatedAt.Local().Format("2006-01-02 15:04")}
			b.WriteString("<li>" + string(s.component("card", props)) + "</li>")
		}
		b.WriteString("</ol>")
	}
	s.page(w, r, plural(t.Name), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name})
}

// detailPage shows one record as a definition list with edit and delete.
func (s *Server) detailPage(w http.ResponseWriter, r *http.Request) {
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
	var b strings.Builder
	b.WriteString(`<dl class="sw-dl">`)
	for _, f := range t.Fields {
		fmt.Fprintf(&b, "<dt>%s</dt><dd>%s</dd>", template.HTMLEscapeString(label(f.Name)), template.HTMLEscapeString(display(f, rec.Fields[f.Name])))
	}
	fmt.Fprintf(&b, "<dt>Created</dt><dd>%s</dd><dt>Updated</dt><dd>%s</dd></dl>", rec.CreatedAt.Local().Format("2006-01-02 15:04"), rec.UpdatedAt.Local().Format("2006-01-02 15:04"))
	b.WriteString(`<div class="sw-cluster" style="margin-top:var(--sw-space-6)">`)
	b.WriteString(string(s.component("link", map[string]any{"href": "/t/" + t.Name + "/" + rec.ID + "/edit", "label": "Edit " + t.Name})))
	fmt.Fprintf(&b, `<form method="post" action="/t/%s/%s/delete">%s</form>`, t.Name, rec.ID, s.component("button", map[string]any{"label": "Delete " + t.Name, "type": "submit", "variant": "danger"}))
	b.WriteString(`</div>`)
	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name + "/" + rec.ID})
}

func (s *Server) newPage(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.page(w, r, "New "+t.Name, s.form(t, "/t/"+t.Name, nil, nil), pageOptions{})
}

func (s *Server) editPage(w http.ResponseWriter, r *http.Request) {
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
	s.page(w, r, "Edit "+titleOf(t, rec), s.form(t, "/t/"+t.Name+"/"+rec.ID, rec.Fields, nil), pageOptions{})
}

func (s *Server) createForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	values := formValues(t, r)
	rec, err := s.app.Store.Create(t.Name, values)
	if err != nil {
		s.formError(w, r, t, "/t/"+t.Name, "New "+t.Name, values, err)
		return
	}
	http.Redirect(w, r, "/t/"+t.Name+"/"+rec.ID, http.StatusSeeOther)
}

func (s *Server) updateForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	id := r.PathValue("id")
	values := formValues(t, r)
	if _, err := s.app.Store.Update(t.Name, id, values); err != nil {
		s.formError(w, r, t, "/t/"+t.Name+"/"+id, "Edit "+t.Name, values, err)
		return
	}
	http.Redirect(w, r, "/t/"+t.Name+"/"+id, http.StatusSeeOther)
}

func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := s.app.Store.Delete(t.Name, r.PathValue("id")); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/t/"+t.Name, http.StatusSeeOther)
}

// formError re-renders the form with the submitted values and per-field errors.
func (s *Server) formError(w http.ResponseWriter, r *http.Request, t *schema.Type, action, title string, values map[string]any, err error) {
	var ve *schema.ValidationError
	if !errors.As(err, &ve) {
		s.fail(w, err)
		return
	}
	s.page(w, r, title, s.form(t, action, values, ve.Problems), pageOptions{Status: http.StatusUnprocessableEntity})
}

// formValues reads every field of a type from the posted form. Absent
// checkboxes become false so an edit can clear them.
func formValues(t *schema.Type, r *http.Request) map[string]any {
	r.ParseForm()
	out := map[string]any{}
	for _, f := range t.Fields {
		if f.Type == "bool" {
			out[f.Name] = r.PostForm.Get(f.Name) != ""
			continue
		}
		if v := strings.TrimSpace(r.PostForm.Get(f.Name)); v != "" {
			out[f.Name] = v
		}
	}
	return out
}

// form builds a form from the type's fields using the design system controls.
func (s *Server) form(t *schema.Type, action string, values map[string]any, problems map[string]string) template.HTML {
	var b strings.Builder
	fmt.Fprintf(&b, `<form method="post" action="%s" class="sw-stack">`, template.HTMLEscapeString(action))
	if len(problems) > 0 {
		b.WriteString(string(s.component("alert", map[string]any{"kind": "danger", "title": "Please fix the fields below", "message": fmt.Sprintf("%d field(s) need attention.", len(problems))})))
	}
	for _, f := range t.Fields {
		b.WriteString(string(s.control(f, values[f.Name], problems[f.Name])))
	}
	b.WriteString(`<div class="sw-cluster">` + string(s.component("button", map[string]any{"label": "Save", "type": "submit"})) + `</div></form>`)
	return template.HTML(b.String())
}

func (s *Server) control(f schema.Field, value any, problem string) template.HTML {
	base := map[string]any{"label": label(f.Name), "name": f.Name, "required": f.Required}
	if f.Description != "" {
		base["hint"] = f.Description
	}
	if problem != "" {
		base["error"] = problem
	}
	switch f.Type {
	case "bool":
		return s.component("checkbox", map[string]any{"label": label(f.Name), "name": f.Name, "checked": value == true, "hint": f.Description})
	case "enum":
		base["options"] = f.Values
		base["value"] = display(f, value)
		return s.component("select", base)
	case "text", "markdown":
		base["value"] = display(f, value)
		base["rows"] = 8
		return s.component("textarea", base)
	case "json":
		base["value"] = display(f, value)
		base["rows"] = 6
		base["hint"] = strings.TrimSpace(f.Description + " Enter JSON.")
		return s.component("textarea", base)
	case "list":
		base["value"] = display(f, value)
		base["hint"] = strings.TrimSpace(f.Description + " Separate items with commas.")
		return s.component("text-field", base)
	case "int", "float":
		base["value"] = display(f, value)
		base["type"] = "number"
		return s.component("text-field", base)
	default:
		base["value"] = display(f, value)
		return s.component("text-field", base)
	}
}

// display renders a stored value as the text a form or page shows.
func display(f schema.Field, v any) string {
	if v == nil {
		return ""
	}
	switch f.Type {
	case "list":
		items, _ := v.([]any)
		parts := make([]string, 0, len(items))
		for _, it := range items {
			parts = append(parts, fmt.Sprint(it))
		}
		return strings.Join(parts, ", ")
	case "json":
		if s, ok := v.(string); ok {
			return s
		}
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	case "bool":
		if b, _ := v.(bool); b {
			return "yes"
		}
		return "no"
	}
	return fmt.Sprint(v)
}

func titleOf(t *schema.Type, rec *store.Record) string {
	if t.Title != "" {
		if s, ok := rec.Fields[t.Title].(string); ok && s != "" {
			return s
		}
	}
	return t.Name + " " + rec.ID
}

func label(field string) string {
	s := strings.ReplaceAll(field, "_", " ")
	return strings.ToUpper(s[:1]) + s[1:]
}
