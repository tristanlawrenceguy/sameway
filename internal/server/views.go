package server

import (
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
		fmt.Fprintf(&b, `<p class="sw-muted">%d matching %s%s. %s</p>`, len(recs), template.HTMLEscapeString(query.Words(t, where)), template.HTMLEscapeString(orderWords(t, order)), s.component("link", map[string]any{"href": "/t/" + t.Name, "label": "See all " + plural(t.Name), "look": "button"}))
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
		b.WriteString(`<script>(function(){var f=document.querySelector('.sw-upload__field');var err=document.getElementById("upload-error");f.addEventListener('invalid',function(e){err.textContent="select a file."},false);document.querySelector(".sw-upload").addEventListener('submit',function(e){if(!f.value){e.preventDefault();err.textContent="select a file.";f.reportValidity()}},{once:true});f.addEventListener('change',function(){err.textContent=""})})();</script>`)
	}
	pg := paged{page: 1, pages: 1}
	if len(recs) == 0 {
		prompt := "Create a " + t.Name + "."
		b.WriteString(string(s.component("empty", map[string]any{
			"title": "No " + plural(t.Name) + " yet", "message": "Add one yourself, or", "action": map[string]any{"href": "/chat?prompt=" + url.PathEscape(prompt), "label": "ask the assistant"},
		})))
	} else if t.Name == HabitType && len(where) == 0 && order == "" {
		// Habits are where each stands this period, and a press to log:
		// the tracker, not rows of names. Archived ones follow, as rows.
		b.WriteString(string(s.component(trackerComponent, s.resolveTracker(map[string]any{"label": "Keeping up"}))))
		var archived []*store.Record
		for _, rec := range recs {
			if on, _ := rec.Fields["archived"].(bool); on {
				archived = append(archived, rec)
			}
		}
		if len(archived) > 0 {
			b.WriteString(`<h2 class="sw-group">Archived <span class="sw-group__count">` + fmt.Sprint(len(archived)) + `</span></h2>`)
			b.WriteString(s.rows(t, archived, time.Now()))
		}
	} else {
		pg = pageOf(r, len(recs), listPageSize)
		b.WriteString(s.rows(t, recs[pg.lo:pg.hi], time.Now()))
		b.WriteString(string(s.pageNav(r, pg, "Pages of "+plural(t.Name))))
	}
	// A new one by hand, and records from a file a person already has,
	// each said once, quietly, below the list.
	b.WriteString(string(s.addButton(t)))
	if s.importable(t) {
		b.WriteString(`<p class="sw-quiet-row">` + string(s.component("link", map[string]any{"href": "/t/" + t.Name + "/import", "label": "Import", "context": plural(t.Name), "look": "button"})) + `</p>`)
	}
	// What just happened to these records is here too, so a deletion can be
	// taken back where the person lands.
	b.WriteString(string(s.recentActivityAbout(5, "/t/"+t.Name, func(target, _ string) bool { return target == t.Name })))
	s.page(w, r, pg.title(capitalize(plural(t.Name))), template.HTML(b.String()), pageOptions{JSONURL: "/api/" + t.Name, Lede: howMany(t, recs), Dot: s.dotOf(t.Name)})
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
	// What this view has been asked to show beyond the least it can say:
	// see parts.go. Nothing here is on unless somebody asked for it.
	always, here := s.shown(r)
	b.WriteString(s.nextThings(t, rec, append(append([]string{}, always...), here...)))
	// Just added and not yet saved: Cancel takes the adding back (add.go).
	discard := ""
	if added := r.URL.Query().Get("added"); added != "" && rec.UpdatedAt.Equal(rec.CreatedAt) {
		discard = ` data-discard="/t/` + t.Name + `/` + rec.ID + `/discard?added=` + template.URLQueryEscaper(added) + `"`
	}
	fmt.Fprintf(&b, `<div class="sw-dl-block" data-block-id="%s" data-edit-action="/t/%s/%s/props"%s%s>`, rec.ID, t.Name, rec.ID, langOf(rec), discard)
	// The record's text comes first and reads as a document, under the
	// title and before its other fields; structured text keeps what was
	// written on the element so the inline editor edits the source.
	textField := ""
	for _, f := range t.Shown() {
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

	// For test_type records with no record values beyond title and chips,
	// show the schema's field definitions instead of leaving the content area blank.
	if t.Name == "test_type" && hasNoRecordValues(t, rec) {
		b.WriteString(string(s.component("fields", map[string]any{"items": s.schemaFields(t)})))
	} else {
		var items []any
		for _, f := range t.Shown() {
			val := display(f, rec.Fields[f.Name])
			if val == "" || f.Name == textField || head[f.Name] {
				continue
			}
			items = append(items, s.fieldItem(t, f, rec.Fields[f.Name], val))
		}
		if len(items) > 0 {
			b.WriteString(string(s.component("fields", map[string]any{"items": items})))
		}
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
		title := s.title(t, rec)
		fmt.Fprintf(&b, `<form method="post" action="/act/%s"><input type="hidden" name="from" value="/t/%s/%s">%s</form>`, rec.ID, t.Name, rec.ID,
			s.component("button", map[string]any{"label": "Run " + trimLabel(title), "context": title, "type": "submit", "variant": "primary"}))
	}
	// The record's one press, done or pinned or whatever its yes-or-no
	// field is, sits under the title; Delete keeps to the quiet bar.
	fmt.Fprintf(&b, `<div class="sw-bar sw-quiet"><form method="post" action="/t/%s/%s/delete">%s</form></div>`,
		t.Name, rec.ID, s.component("button", map[string]any{"label": "Delete " + t.Name, "type": "submit", "variant": "quiet"}))
	b.WriteString(s.editFields(t, rec))
	b.WriteString(`</div>`)
	// Recent activity on this page, so a deletion can be taken back where the person lands.
	b.WriteString(string(s.recentActivityAbout(5, "/t/"+t.Name+"/"+rec.ID, func(target, id string) bool { return target == t.Name && id == rec.ID })))
	// What this record is connected to, as a line of counts; the address
	// says which of them are open. See related.go.
	b.WriteString(s.related(t, rec, always, here))
	s.page(w, r, trimTitle(s.title(t, rec)), template.HTML(b.String()), pageOptions{
		Kicker:       s.crumbs("/t/"+t.Name, capitalize(plural(t.Name)), "", s.dotOf(t.Name)),
		Lede:         s.lede(t, rec),
		JSONURL:      "/api/" + t.Name + "/" + rec.ID,
		ExtraScripts: detailPageExtraScripts,
	})
}

// crumbs is the way back from a detail page: the listing it belongs to,
// then the record itself (when here is non-empty). When here is empty,
// only the listing link renders — useful on detail pages where the title
// already appears as h1 and repeating it in crumbs would be redundant.
func (s *Server) crumbs(listHref, listLabel, here string, dot int) template.HTML {
	place := map[string]any{"href": listHref, "label": listLabel}
	if dot > 0 {
		place["dot"] = dot
	}
	props := map[string]any{"items": []any{place}}
	if here != "" {
		props["current"] = here
	}
	return s.component("crumbs", props)
}

// maxTitleRunes is the character ceiling for a heading title. Six medium
// words fit comfortably under 80 runes, so this catches runaway single-word
// titles while leaving normal six-word titles alone.
const maxTitleRunes = 75

// trimTitle cuts a title to at most six words and at most maxTitleRunes
// characters so no heading exceeds reasonable length. Titles already within
// both limits pass through unchanged. An ellipsis is appended when trimmed.
func trimLabel(s string) string {
	fields := strings.Fields(s)
	if len(fields) <= 3 {
		return s
	}
	return strings.Join(fields[:3], " ")
}

func trimTitle(s string) string {
	fields := strings.Fields(s)
	if len(fields) <= 6 && len([]rune(s)) <= maxTitleRunes {
		return s
	}
	if len(fields) > 6 {
		return strings.Join(fields[:6], " ") + "…"
	}
	// ≤ 6 words but too many characters: truncate at character boundary.
	runes := []rune(s)
	return string(runes[:maxTitleRunes-1]) + "…"
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

// fieldItem is one field of a record as the fields component shows it:
// structured text as it reads, a ref or a reminder's about as the way to
// what it names, a choice by its name with the stored value kept for the
// editor, a date in words with the moment kept under it.
func (s *Server) fieldItem(t *schema.Type, f schema.Field, v any, val string) map[string]any {
	switch {
	case f.Type == "markdown":
		return map[string]any{"label": fieldLabel(f), "markdown": val, "prop": f.Name}
	case t.Name == ReminderType && f.Name == "about":
		return s.aboutItem(f, val)
	case f.Type == "ref":
		return s.refItem(f, val)
	}
	item := map[string]any{"label": fieldLabel(f), "value": val, "prop": f.Name}
	switch f.Type {
	case "datetime":
		item["kind"], item["source"] = "datetime", fmt.Sprint(v)
	case "enum":
		item["source"] = fmt.Sprint(v)
		item["options"] = s.choiceList(f, fmt.Sprint(v))
	}
	return item
}
