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
			meta := "Updated " + rec.UpdatedAt.Local().Format("2006-01-02 15:04")
			// A type's first enum field is its state (pending, published,
			// dismissed), which is what a person scanning a list wants to see.
			for _, f := range t.Fields {
				if f.Type == "enum" {
					if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
						meta = capitalize(v) + " · " + meta
					}
					break
				}
			}
			props := map[string]any{"title": titleOf(t, rec), "href": "/t/" + t.Name + "/" + rec.ID, "level": 2, "meta": meta}
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
	b.WriteString(string(crumbs("/t/"+t.Name, capitalize(plural(t.Name)), titleOf(t, rec))))
	// A question still waiting is answered here as well as under the
	// conversation: the page of a proposal is where the two answers belong.
	if t.Name == chat.ProposalType && rec.Fields["state"] == "pending" {
		b.WriteString(string(s.proposalCard(rec, "/t/"+t.Name+"/"+rec.ID)))
	}
	fmt.Fprintf(&b, `<div class="sw-dl-block" data-block-id="%s" data-edit-action="/t/%s/%s/props">`, rec.ID, t.Name, rec.ID)
	b.WriteString(`<dl class="sw-dl">`)
	for _, f := range t.Fields {
		val := display(f, rec.Fields[f.Name])
		if val == "" {
			continue
		}
		fmt.Fprintf(&b, `<dt>%s</dt><dd data-prop="%s">%s</dd>`, template.HTMLEscapeString(label(f.Name)), f.Name, template.HTMLEscapeString(val))
	}
	fmt.Fprintf(&b, "<dt>Created</dt><dd>%s</dd><dt>Updated</dt><dd>%s</dd></dl>", rec.CreatedAt.Local().Format("2006-01-02 15:04"), rec.UpdatedAt.Local().Format("2006-01-02 15:04"))
	b.WriteString(`<div class="sw-bar sw-quiet">` + string(s.component("link", map[string]any{
		"href":  "/t/" + t.Name + "/" + rec.ID + "/confirm-delete",
		"label": "Delete " + t.Name,
	})) + `</div>`)
	b.WriteString(`</div>`)
	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{
		JSONURL:      "/api/" + t.Name + "/" + rec.ID,
		ExtraScripts: detailPageExtraScripts,
	})
}

func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
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
	if err := s.app.Store.Delete(t.Name, rec.ID); err != nil {
		s.fail(w, err)
		return
	}
	// Logged with what it was, so the deletion can be undone.
	chat.Record(s.app.Store, "human", chat.Change{Action: "deleted", Component: t.Name, ID: rec.ID, Detail: titleOf(t, rec), Before: rec.Fields})
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

// titleOf names a record: its title field, else the first string field
// with something in it, else its type and id. A blank title used to fall
// straight to the id, so a list of activities read as a column of "said".
func titleOf(t *schema.Type, rec *store.Record) string {
	if t.Title != "" {
		if s, ok := rec.Fields[t.Title].(string); ok && s != "" {
			return s
		}
	}
	for _, f := range t.Fields {
		if f.Type != "string" && f.Type != "text" && f.Type != "enum" {
			continue
		}
		if s, ok := rec.Fields[f.Name].(string); ok && strings.TrimSpace(s) != "" {
			return truncateTitle(s)
		}
	}
	return t.Name + " " + rec.ID
}

// crumbs is the way back from a detail page: the listing it belongs to,
// then the record itself. A person who read one item and wants the next
// one should not have to find the footer or the browser's back button.
func crumbs(listHref, listLabel, here string) template.HTML {
	return template.HTML(fmt.Sprintf(`<nav class="sw-crumbs" aria-label="You are here"><ol class="sw-plain sw-crumbs__list"><li><a class="sw-link" href="%s">%s</a></li><li aria-current="page">%s</li></ol></nav>`,
		template.HTMLEscapeString(listHref), template.HTMLEscapeString(listLabel), template.HTMLEscapeString(here)))
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
