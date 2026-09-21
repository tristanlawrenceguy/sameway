package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	store "github.com/tristanlawrenceguy/sameway/internal/store"
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

	rec, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	fields, err := editedFields(r.PostForm)
	if err != nil {
		s.renderDetailError(w, r, t, rec, err, fields)
		return
	}

	// No fields provided — no-op redirect.
	if len(fields) == 0 {
		http.Redirect(w, r, returnTo(r, "/t/"+t.Name+"/"+rec.ID), http.StatusSeeOther)
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
		s.renderDetailError(w, r, t, rec, err, fields)
		return
	}

	_, err = s.app.Store.Update(t.Name, rec.ID, clean)
	if err != nil {
		s.fail(w, err)
		return
	}
	// A change a person made by hand is a change like any other: in the
	// log with what it was, so it glows where it shows and can be undone.
	chat.Record(s.app.Store, "human", chat.Change{Action: "updated", Component: t.Name, ID: rec.ID, Detail: titleOf(t, rec), Href: "/t/" + t.Name + "/" + rec.ID, Before: rec.Fields})

	http.Redirect(w, r, returnTo(r, "/t/"+t.Name+"/"+rec.ID), http.StatusSeeOther)
}

// renderDetailError re-renders the detail page with validation errors as a 422,
// showing each field's problem alongside the current record values so the person
// can see what they are editing and try again.
func (s *Server) renderDetailError(w http.ResponseWriter, r *http.Request, t *schema.Type, rec *store.Record, verr error, submittedFields map[string]any) {
	var b strings.Builder

	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "warning",
		"title":   "That did not save",
		"message": "Fix the fields below and try again.",
	})))

	if ve, ok := verr.(*schema.ValidationError); ok {
		for field, msg := range ve.Problems {
			b.WriteString(fmt.Sprintf("<p>%s %s</p>",
				template.HTMLEscapeString(label(field)), template.HTMLEscapeString(msg)))
		}
	}

	// Render the definition list like detailPage does, so the person can see
	// what they are editing and try again.
	fmt.Fprintf(&b, `<div class="sw-dl-block" data-block-id="%s" data-edit-action="/t/%s/%s/props">`, rec.ID, t.Name, rec.ID)
	b.WriteString(`<dl class="sw-dl">`)
	for _, f := range t.Fields {
		var raw any
		if s, ok := submittedFields[f.Name]; ok {
			raw = s
		} else {
			raw = rec.Fields[f.Name]
		}
		val := display(f, raw)
		// On an error page we must render every field the user submitted so that
		// 08-edit.js can build an input for it (even when the value is empty).
		if _, ok := submittedFields[f.Name]; !ok && val == "" {
			continue
		}
		fmt.Fprintf(&b, `<dt>%s</dt><dd data-prop="%s"%s>%s</dd>`,
			template.HTMLEscapeString(label(f.Name)), f.Name, whenAttrs(f, rec.Fields[f.Name]), template.HTMLEscapeString(val))
	}
	b.WriteString("</dl>" + `<p class="sw-lede">` + whenMade(rec) + `</p>`)
	b.WriteString(`<div class="sw-bar sw-quiet"></div>`)
	b.WriteString(`</div>`)

	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name + "/" + rec.ID, Status: http.StatusUnprocessableEntity, ExtraScripts: detailPageExtraScripts})
}
