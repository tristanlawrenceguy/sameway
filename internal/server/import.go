package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Records from a file. Every list page offers to import from a file: a
// CSV from anywhere, a vCard of contacts, a mailbox export. The file is
// kept as a file record like any other, read as a table, and shown with
// each column matched to a field, which the person can change before
// the rows become records. An agent does the same through the API and
// the assistant through its tool, with the matching left to the guess.

func (s *Server) importable(t *schema.Type) bool {
	return !t.Internal && t.Name != FileType
}

// importPage is the form to choose a file, or, with ?file=, the preview
// of that file with the matching to confirm.
func (s *Server) importPage(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok || !s.importable(t) {
		http.NotFound(w, r)
		return
	}
	if id := r.URL.Query().Get("file"); id != "" {
		s.importPreview(w, r, t, id, "")
		return
	}
	var b strings.Builder
	switch t.Name {
	case "person":
		b.WriteString(`<p class="sw-muted">A CSV with a header row, a vCard (.vcf) of contacts, or a mailbox (.mbox) of mail. The file is kept with your files; its rows become ` + template.HTMLEscapeString(plural(t.Name)) + `, and you see how each column lands before anything is made.</p>`)
	default:
		b.WriteString(`<p class="sw-muted">A CSV with a header row, a tab-separated file (.tsv), or plain text (.txt). The file is kept with your files; its rows become ` + template.HTMLEscapeString(plural(t.Name)) + `, and you see how each column lands before anything is made.</p>`)
	}
	fmt.Fprintf(&b, `<form method="post" action="/t/%s/import" enctype="multipart/form-data" class="sw-stack sw-import"><div class="sw-field"><label class="sw-field__label" for="import-file">File</label><input class="sw-field__input" id="import-file" type="file" name="file" accept=".csv,.tsv,.txt,.vcf,.vcard,.mbox,.eml" required aria-describedby="import-error"><span class="sw-visually-hidden" id="import-error" role="status" aria-live="assertive"></span></div>%s</form>`,
		t.Name, s.component("button", map[string]any{"label": "Read the file", "type": "submit"}))
	b.WriteString(`<script>(function(){var f=document.getElementById("import-file");var err=document.getElementById("import-error");f.addEventListener('invalid',function(e){err.textContent="select a file."},false);document.querySelector(".sw-import").addEventListener('submit',function(e){if(!f.value){e.preventDefault();err.textContent="select a file.";f.reportValidity()}},{once:true});f.addEventListener('change',function(){err.textContent=""})})();</script>`)
	s.page(w, r, "Import "+plural(t.Name), template.HTML(b.String()), pageOptions{Kicker: crumbs("/t/"+t.Name, capitalize(plural(t.Name)), "Import", s.dotOf(t.Name)), Dot: s.dotOf(t.Name)})
}

// importUpload keeps the file and goes to its preview.
func (s *Server) importUpload(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok || !s.importable(t) {
		http.NotFound(w, r)
		return
	}
	rec, err := s.storeUpload(r)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(err.Error(), "select a file") {
			err = errors.New("choose a file first")
		}
		s.tellAt(w, r, outcome{Failed: true, Title: "Nothing was read", Text: plainError(err)}, "/t/"+t.Name+"/import")
		return
	}
	http.Redirect(w, r, "/t/"+t.Name+"/import?file="+rec.ID, http.StatusSeeOther)
}

// original is a kept file's bytes and name.
func (s *Server) original(id string) (name string, data []byte, err error) {
	rec, err := s.app.Store.Get(FileType, id)
	if err != nil {
		return "", nil, err
	}
	stored, _ := rec.Fields["path"].(string)
	if stored == "" || strings.ContainsAny(stored, `/\`) {
		return "", nil, fmt.Errorf("the file %s has no original kept", id)
	}
	name, _ = rec.Fields["name"].(string)
	data, err = os.ReadFile(filepath.Join(s.app.Workspace.FilesDir(), stored))
	return name, data, err
}

// importPreview shows the table as it will land: the first rows, and for
// each column which field it feeds, to change before importing.
func (s *Server) importPreview(w http.ResponseWriter, r *http.Request, t *schema.Type, id, problem string) {
	name, data, err := s.original(id)
	if err != nil {
		s.fail(w, err)
		return
	}
	tb, err := ingest.Read(name, data)
	if err != nil {
		s.page(w, r, "Import "+plural(t.Name), s.component("alert", map[string]any{"kind": "warning", "title": "Nothing was read", "message": err.Error()}), pageOptions{Status: http.StatusBadRequest})
		return
	}
	m := ingest.Guess(t, tb.Columns)
	var b strings.Builder
	if problem != "" {
		b.WriteString(string(s.component("alert", map[string]any{"kind": "warning", "message": problem})))
	}
	fmt.Fprintf(&b, `<p class="sw-muted">%d rows in %s. Each column below feeds the field it names; change any that landed wrong, or set it to nothing to leave it out. A column for an email, a phone or a name that feeds nothing still links each row to its person.</p>`, len(tb.Rows), template.HTMLEscapeString(name))
	fmt.Fprintf(&b, `<form method="post" action="/t/%s/import/%s/run" class="sw-stack sw-import">`, t.Name, id)
	b.WriteString(`<div class="sw-table-wrap"><table class="sw-table sw-import__table"><caption class="sw-visually-hidden">The first rows, with the field each column feeds</caption><thead><tr>`)
	for _, col := range tb.Columns {
		fmt.Fprintf(&b, `<th scope="col">%s</th>`, template.HTMLEscapeString(col))
	}
	b.WriteString(`</tr><tr class="sw-import__map">`)
	// Each column's choice is the select component, naming the fields
	// the way the editor does, so a person picks "Due" rather than "due".
	options := []any{map[string]any{"value": "", "label": "Nothing"}}
	for _, f := range t.Fields {
		options = append(options, map[string]any{"value": f.Name, "label": fieldLabel(f)})
	}
	for i, col := range tb.Columns {
		b.WriteString(`<td>`)
		b.WriteString(string(s.component("select", map[string]any{
			"label": col + " goes into", "name": "map-" + col, "id": fmt.Sprintf("map-%d", i), "options": options, "value": m[col],
		})))
		b.WriteString(`</td>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for i, row := range tb.Rows {
		if i == 8 {
			break
		}
		b.WriteString(`<tr>`)
		for _, col := range tb.Columns {
			v := row[col]
			if len(v) > 80 {
				v = v[:80] + "…"
			}
			fmt.Fprintf(&b, `<td>%s</td>`, template.HTMLEscapeString(v))
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	b.WriteString(string(s.component("button", map[string]any{"label": fmt.Sprintf("Import %d %s", len(tb.Rows), plural(t.Name)), "type": "submit"})))
	b.WriteString(`</form>`)
	s.page(w, r, "Import "+plural(t.Name), template.HTML(b.String()), pageOptions{Kicker: crumbs("/t/"+t.Name, capitalize(plural(t.Name)), "Import", s.dotOf(t.Name)), Dot: s.dotOf(t.Name)})
}

// importRun makes the records as the form says and goes to the list,
// with what happened said in the chat and the log.
func (s *Server) importRun(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok || !s.importable(t) {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	m := ingest.Mapping{}
	for key, vals := range r.PostForm {
		if col, ok := strings.CutPrefix(key, "map-"); ok && len(vals) > 0 && vals[0] != "" {
			m[col] = vals[0]
		}
	}
	report, err := s.importFile(t, r.PathValue("file"), m)
	if err != nil {
		s.importPreview(w, r, t, r.PathValue("file"), err.Error())
		return
	}
	s.tellAt(w, r, outcome{Title: "Imported", Text: capitalize(plural(t.Name)) + " from the file: " + report.String() + "."}, "/t/"+t.Name)
}

// importFile reads a kept file as a table and makes records of t from
// it, with the mapping given or the guess. The log has the import as one
// entry, so the person sees it happened and how many.
func (s *Server) importFile(t *schema.Type, fileID string, m ingest.Mapping) (ingest.Report, error) {
	name, data, err := s.original(fileID)
	if err != nil {
		return ingest.Report{}, err
	}
	tb, err := ingest.Read(name, data)
	if err != nil {
		return ingest.Report{}, err
	}
	if len(m) == 0 {
		m = ingest.Guess(t, tb.Columns)
	}
	report := ingest.Import(s.app.Store, t, tb, m)
	chat.Record(s.app.Store, "human", chat.Change{Action: "imported", Component: t.Name, Detail: fmt.Sprintf("%d %s from %s", report.Made, plural(t.Name), name), Href: "/t/" + t.Name, Before: chat.Imported(t.Name, report.IDs)})
	return report, nil
}

// apiImport makes records of a type from a kept file: {"file": id} with
// an optional {"mapping": {column: field}}; the answer is the report.
func (s *Server) apiImport(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok || !s.importable(t) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found", Message: "no such content type to import into"}})
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	fileID, _ := body["file"].(string)
	m := ingest.Mapping{}
	if given, ok := body["mapping"].(map[string]any); ok {
		for col, field := range given {
			if f, ok := field.(string); ok {
				m[col] = f
			}
		}
	}
	report, err := s.importFile(t, fileID, m)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"made": report.Made, "skipped": report.Skipped, "linked": report.Linked, "problems": report.Problems, "ids": report.IDs, "summary": report.String()})
}
