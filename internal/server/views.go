package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Canvas card controls on / include the note title instead of "card":
// edit controls read as "Edit {{.Title}}", expand as "Expand {{.Title}}",
// and remove as "Remove {{.Title}}" via canvasBlock's context prop.

// listPage shows every record of a type as cards.
func (s *Server) listPage(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	var b strings.Builder
	// The same query a collection block takes, in the address: ?where=…&order=…
	where, order := r.URL.Query()["where"], r.URL.Query().Get("order")
	var recs []*store.Record
	var err error
	if len(where) > 0 || order != "" {
		recs, err = query.Filter(s.app.Store, t, where, order, 0, time.Now())
		if err != nil {
			fmt.Fprintf(&b, `<p class="sw-muted">%s</p><p>%s</p>`, template.HTMLEscapeString(err.Error()), s.component("link", map[string]any{"href": "/t/" + t.Name, "label": "See all " + plural(t.Name), "look": "button"}))
			s.page(w, r, plural(t.Name), template.HTML(b.String()), pageOptions{Status: http.StatusBadRequest})
			return
		}
		fmt.Fprintf(&b, `<p class="sw-muted">%d matching %s%s. %s</p>`, len(recs), template.HTMLEscapeString(strings.Join(where, ", ")), template.HTMLEscapeString(orderWords(order)), s.component("link", map[string]any{"href": "/t/" + t.Name, "label": "See all " + plural(t.Name), "look": "button"}))
	} else {
		recs, err = s.app.Store.List(t.Name, store.ListOptions{})
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	// Files come in through a form, because one field and one button is
	// the better thing here; it can also be placed anywhere as a block.
	if t.Name == FileType {
		b.WriteString(string(s.component("upload", map[string]any{"from": "/t/" + FileType, "id": "upload"})))
		b.WriteString(`<script>(function(){var f=document.querySelector('.sw-upload__field');var err=document.getElementById("upload-error");f.addEventListener('invalid',function(e){err.textContent="Please select a file."},false);document.querySelector(".sw-upload").addEventListener('submit',function(e){if(!f.value){e.preventDefault();err.textContent="Please select a file.";f.reportValidity()}},{once:true});f.addEventListener('change',function(){err.textContent=""})})();</script>`)
	}
	if len(recs) == 0 {
		fmt.Fprintf(&b, `<p class="sw-empty">Add your first <a href="/t/%s/new">%s</a>.</p>`, template.HTMLEscapeString(t.Name), template.HTMLEscapeString(plural(t.Name)))
	} else {
		b.WriteString(s.rows(t, recs, time.Now()))
	}
	// Records can come from a file a person already has, and the page
	// says so, once, quietly, below the list.
	if s.importable(t) {
		b.WriteString(`<p class="sw-quiet-row">` + string(s.component("link", map[string]any{"href": "/t/" + t.Name + "/import", "label": "Import " + plural(t.Name) + " from a file", "look": "button"})) + `</p>`)
	}
	// What just happened to these records is here too, so a deletion can be
	// taken back where the person lands.
	b.WriteString(string(s.recentActivity(5, "/t/"+t.Name)))
	s.page(w, r, capitalize(plural(t.Name)), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name, Lede: howMany(t, recs), Dot: s.dotOf(t.Name)})
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
	// A question still waiting is answered here as well as under the
	// conversation: the page of a proposal is where the two answers belong.
	if t.Name == chat.ProposalType && rec.Fields["state"] == "pending" {
		b.WriteString(string(s.proposalCard(rec, "/t/"+t.Name+"/"+rec.ID)))
	}
	// A file's page shows the picture when it is one, and the way to the
	// original, above the fields read from it.
	if t.Name == FileType {
		b.WriteString(s.fileExtras(rec))
	}
	fmt.Fprintf(&b, `<div class="sw-dl-block" data-block-id="%s" data-edit-action="/t/%s/%s/props">`, rec.ID, t.Name, rec.ID)
	// The record's text comes first and reads as a document, under the
	// title and before its other fields; structured text keeps what was
	// written on the element so the inline editor edits the source.
	textField := ""
	for _, f := range t.Fields {
		if f.Type == "markdown" {
			if val := display(f, rec.Fields[f.Name]); val != "" {
				textField = f.Name
				fmt.Fprintf(&b, `<div class="sw-prose sw-detail__body" data-prop="%s" data-source="%s" data-prose-level="2">%s</div>`, f.Name, template.HTMLEscapeString(val), prose.Render(val, 2))
			}
			break
		}
	}
	b.WriteString(`<dl class="sw-dl">`)
	for _, f := range t.Fields {
		val := display(f, rec.Fields[f.Name])
		if val == "" || f.Name == textField {
			continue
		}
		if f.Type == "markdown" {
			fmt.Fprintf(&b, `<dt>%s</dt><dd class="sw-prose" data-prop="%s" data-source="%s" data-prose-level="3">%s</dd>`, template.HTMLEscapeString(label(f.Name)), f.Name, template.HTMLEscapeString(val), prose.Render(val, 3))
			continue
		}
		// A ref shows the record it points at, as the way there.
		if f.Type == "ref" {
			fmt.Fprintf(&b, `<dt>%s</dt>%s`, template.HTMLEscapeString(label(f.Name)), s.refCell(f, val))
			continue
		}
		fmt.Fprintf(&b, `<dt>%s</dt><dd data-prop="%s"%s>%s</dd>`, template.HTMLEscapeString(label(f.Name)), f.Name, whenAttrs(f, rec.Fields[f.Name]), template.HTMLEscapeString(val))
	}
	b.WriteString("</dl>")
	// Deleting is one step, because it can be taken back: the record goes
	// with everything it had into the activity log, and the listing the
	// person lands on offers to put it back. No page asks "are you sure".
	// An action is a button; its own page has that button.
	if t.Name == chat.ActionType {
		fmt.Fprintf(&b, `<form method="post" action="/act/%s"><input type="hidden" name="from" value="/t/%s/%s">%s</form>`, rec.ID, t.Name, rec.ID,
			s.component("button", map[string]any{"label": "Run", "context": titleOf(t, rec), "type": "submit", "variant": "primary"}))
	}
	// The record's one press, done or pinned or whatever its yes-or-no
	// field is, sits under the title; Delete keeps to the quiet bar.
	fmt.Fprintf(&b, `<div class="sw-bar sw-quiet"><form method="post" action="/t/%s/%s/delete">%s</form></div>`,
		t.Name, rec.ID, s.component("button", map[string]any{"label": "Delete " + t.Name, "type": "submit", "variant": "quiet"}))
	b.WriteString(`</div>`)
	// Recent activity on this page, so a deletion can be taken back where the person lands.
	b.WriteString(string(s.recentActivity(5, "/t/"+t.Name+"/"+rec.ID)))
	// What points at this record, listed here by itself.
	b.WriteString(s.backlinks(t, rec))
	s.page(w, r, titleOf(t, rec), template.HTML(b.String()), pageOptions{
		Kicker:       crumbs("/t/"+t.Name, capitalize(plural(t.Name)), titleOf(t, rec), s.dotOf(t.Name)),
		Lede:         s.lede(t, rec),
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
	title := titleOf(t, rec)
	if err := s.app.Store.Delete(t.Name, rec.ID); err != nil {
		s.fail(w, err)
		return
	}
	// Logged with what it was, so the deletion can be undone.
	chat.Record(s.app.Store, "human", chat.Change{Action: "deleted", Component: t.Name, ID: rec.ID, Detail: title, Before: rec.Fields})

	// Render a confirmation page with an alert before redirecting back to
	// the listing, so the person knows the delete actually worked.
	var b strings.Builder
	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "success",
		"title":   "Deleted",
		"message": title + " deleted.",
	})))
	fmt.Fprintf(&b, `<p>%s</p>`, s.component("link", map[string]any{"href": "/t/" + t.Name, "label": "See all " + plural(t.Name), "look": "button"}))
	b.WriteString(`<meta http-equiv="refresh" content="2;url=/t/` + t.Name + `">`)
	s.page(w, r, title+" deleted", template.HTML(b.String()), pageOptions{})
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
	case "datetime":
		return when.Text(fmt.Sprint(v))
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
func crumbs(listHref, listLabel, here string, dot int) template.HTML {
	mark := ""
	if dot > 0 {
		mark = fmt.Sprintf(` class="sw-dotted" data-dot="%d"`, dot)
	}
	return template.HTML(fmt.Sprintf(`<nav class="sw-crumbs" aria-label="You are here"><ol class="sw-plain sw-crumbs__list"><li%s><a class="sw-link" href="%s">%s</a></li><li aria-current="page">%s</li></ol></nav>`,
		mark, template.HTMLEscapeString(listHref), template.HTMLEscapeString(listLabel), template.HTMLEscapeString(here)))
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

// whenAttrs marks a date on a page for the editor and for a machine: the
// kind, and the stored value under the words a person reads.
func whenAttrs(f schema.Field, v any) string {
	if f.Type != "datetime" || v == nil || v == "" {
		return ""
	}
	return fmt.Sprintf(` data-kind="datetime" data-source="%s"`, template.HTMLEscapeString(fmt.Sprint(v)))
}
