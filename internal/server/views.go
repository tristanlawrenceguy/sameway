package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
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
	// Empty-state text: "Ask the assistant to add your first" — replaces old "/Add your first" at /t/note/new.
	// The new link points to /chat (the working surface) instead of dead form routes.
	if t.Name == FileType {
		b.WriteString(string(s.component("upload", map[string]any{"from": "/t/" + FileType, "id": "upload"})))
		b.WriteString(`<script>(function(){var f=document.querySelector('.sw-upload__field');var err=document.getElementById("upload-error");f.addEventListener('invalid',function(e){err.textContent="Please select a file."},false);document.querySelector(".sw-upload").addEventListener('submit',function(e){if(!f.value){e.preventDefault();err.textContent="Please select a file.";f.reportValidity()}},{once:true});f.addEventListener('change',function(){err.textContent=""})})();</script>`)
	}
	if len(recs) == 0 {
		prompt := "Create a " + t.Name + "."
		linkText := fmt.Sprintf("Add %s", t.Name) // e.g. "Add note" for notes type
		fmt.Fprintf(&b, `<p class="sw-empty">Ask the assistant to add your first <a href="/chat?prompt=%s">%s</a>.</p>`, template.HTMLEscapeString(url.PathEscape(prompt)), template.HTMLEscapeString(linkText))
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
	if r.URL.Query().Has("saved") {
		b.WriteString(string(s.component("alert", map[string]any{
			"kind":    "success",
			"title":   "Changes saved",
			"message": "Your edits were applied.",
			"dismiss": true,
		})))
	}
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
	// What this view has been asked to show beyond the least it can say:
	// see parts.go. Nothing here is on unless somebody asked for it.
	always, here := s.shown(r)
	b.WriteString(s.nextThings(t, rec, append(append([]string{}, always...), here...)))
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
	// The list leaves out what the heading and the chips above it have
	// already said, so the page says each thing once; ?show=fields brings
	// the whole record back except for those already-in-chips fields.
	head := headFields(t, rec)
	var dl strings.Builder
	for _, f := range t.Fields {
		val := display(f, rec.Fields[f.Name])
		if val == "" || f.Name == textField {
			continue
		}
		if head[f.Name] {
			continue
		}
		if f.Type == "markdown" {
			fmt.Fprintf(&dl, `<dt>%s</dt><dd class="sw-prose" data-prop="%s" data-source="%s" data-prose-level="3">%s</dd>`, template.HTMLEscapeString(label(f.Name)), f.Name, template.HTMLEscapeString(val), prose.Render(val, 3))
			continue
		}
		// A reminder's about is the thing it is for, as the way there.
		if t.Name == ReminderType && f.Name == "about" {
			fmt.Fprintf(&dl, `<dt>%s</dt>%s`, template.HTMLEscapeString(label(f.Name)), s.aboutCell(f, val))
			continue
		}
		// A ref shows the record it points at, as the way there.
		if f.Type == "ref" {
			fmt.Fprintf(&dl, `<dt>%s</dt>%s`, template.HTMLEscapeString(label(f.Name)), s.refCell(f, val))
			continue
		}
		fmt.Fprintf(&dl, `<dt>%s</dt><dd data-prop="%s"%s%s>%s</dd>`, template.HTMLEscapeString(label(f.Name)), f.Name, whenAttrs(f, rec.Fields[f.Name]), s.choices(f, val), template.HTMLEscapeString(val))
	}
	if dl.Len() > 0 {
		b.WriteString(`<dl class="sw-dl">` + dl.String() + "</dl>")
	}
	// The way back out, when the address is what opened the whole record.
	b.WriteString(s.fewer("/t/"+t.Name+"/"+rec.ID, FieldsPart, "fields of "+s.title(t, rec), here))
	// A habit's page is where it stands: its row, the best run, a chart.
	if t.Name == HabitType {
		b.WriteString(string(s.habitSection(rec)))
	}
	// Deleting is one step, because it can be taken back: the record goes
	// with everything it had into the activity log, and the listing the
	// person lands on offers to put it back. No page asks "are you sure".
	// An action is a button; its own page has that button.
	if t.Name == chat.ActionType {
		fmt.Fprintf(&b, `<form method="post" action="/act/%s"><input type="hidden" name="from" value="/t/%s/%s">%s</form>`, rec.ID, t.Name, rec.ID,
			s.component("button", map[string]any{"label": "Run", "context": s.title(t, rec), "type": "submit", "variant": "primary"}))
	}
	// The record's one press, done or pinned or whatever its yes-or-no
	// field is, sits under the title; Delete keeps to the quiet bar.
	fmt.Fprintf(&b, `<div class="sw-bar sw-quiet"><form method="post" action="/t/%s/%s/delete">%s</form></div>`,
		t.Name, rec.ID, s.component("button", map[string]any{"label": "Delete " + t.Name, "type": "submit", "variant": "quiet"}))
	b.WriteString(`</div>`)
	// Recent activity on this page, so a deletion can be taken back where the person lands.
	b.WriteString(string(s.recentActivity(5, "/t/"+t.Name+"/"+rec.ID)))
	// What this record is connected to, as a line of counts; the address
	// says which of them are open. See related.go.
	b.WriteString(s.related(t, rec, always, here))
	s.page(w, r, s.title(t, rec), template.HTML(b.String()), pageOptions{
		Kicker:       crumbs("/t/"+t.Name, capitalize(plural(t.Name)), "", s.dotOf(t.Name)),
		Lede:         s.lede(t, rec),
		JSONURL:      "/api/" + t.Name + "/" + rec.ID,
		ExtraScripts: detailPageExtraScripts,
	})
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
// then the record itself (when here is non-empty). When here is empty,
// only the listing link renders — useful on detail pages where the title
// already appears as h1 and repeating it in crumbs would be redundant.
func crumbs(listHref, listLabel, here string, dot int) template.HTML {
	mark := ""
	if dot > 0 {
		mark = fmt.Sprintf(` class="sw-dotted" data-dot="%d"`, dot)
	}
	listLabelEscaped := template.HTMLEscapeString(listLabel)
	listHrefEscaped := template.HTMLEscapeString(listHref)
	if here == "" {
		return template.HTML(fmt.Sprintf(`<nav class="sw-crumbs" aria-label="You are here"><ol class="sw-plain sw-crumbs__list"><li%s><a class="sw-link" href="%s">%s</a></li></ol></nav>`,
			mark, listHrefEscaped, listLabelEscaped))
	}
	return template.HTML(fmt.Sprintf(`<nav class="sw-crumbs" aria-label="You are here"><ol class="sw-plain sw-crumbs__list"><li%s><a class="sw-link" href="%s">%s</a></li><li aria-current="page">%s</li></ol></nav>`,
		mark, listHrefEscaped, listLabelEscaped, template.HTMLEscapeString(here)))
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
