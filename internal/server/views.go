package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
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
	s.page(w, r, capitalize(plural(t.Name)), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name})
}

// detailPage shows one record as a definition list with delete.
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
	b.WriteString(string(s.component("link", map[string]any{
		"href":  "/t/" + t.Name + "/" + rec.ID + "/confirm-delete",
		"label": "Delete " + t.Name,
	})))
	b.WriteString(`</div>`)
	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name + "/" + rec.ID})
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

// display renders a stored value as the text a form or page shows.
func display(f schema.Field, v any) string {
	if v == nil {
		return ""
	}
	switch f.Type {
	case "list":
		if s, ok := v.(string); ok {
			return s // a value the person just typed, coming back after an error
		}
		items, _ := v.([]any)
		parts := make([]string, 0, len(items))
		for _, it := range items {
			parts = append(parts, fmt.Sprint(it))
		}
		if f.Multiline {
			return strings.Join(parts, "\n")
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

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + s[1:]
}

func label(field string) string {
	s := strings.ReplaceAll(field, "_", " ")
	return strings.ToUpper(s[:1]) + s[1:]
}
