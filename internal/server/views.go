package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
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

	var right strings.Builder
	right.WriteString(string(s.component("link", map[string]any{"href": "/t/" + t.Name + "/" + rec.ID + "/confirm-delete", "label": "Delete " + titleOf(t, rec), "context": titleOf(t, rec)})))

	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name + "/" + rec.ID, Right: template.HTML(right.String())})
}

// display renders a stored value as the text shown on pages.
func display(f schema.Field, v any) string {
	if v == nil {
		return ""
	}
	switch f.Type {
	case "list":
		if s, ok := v.(string); ok {
			return s
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

func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	id := r.PathValue("id")
	if err := s.app.Store.Delete(t.Name, id); err != nil {
		s.fail(w, err)
		return
	}
	if t.Name == chat.BlockType {
		http.Redirect(w, r, "/#canvas", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/t/"+t.Name, http.StatusSeeOther)
}

func label(field string) string {
	s := strings.ReplaceAll(field, "_", " ")
	return strings.ToUpper(s[:1]) + s[1:]
}
