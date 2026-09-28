package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Whatever comes in goes out again: any list as a spreadsheet, people as
// contacts, anything with a date as a calendar, taking the same query the
// list page shows, so what is on the page is what is in the file. A
// calendar's address is also a feed a calendar app can subscribe to.

// exportable says whether a type's records are a person's to take out:
// the system's own (the conversation, the log, the questions) are not.
func exportable(t *schema.Type) bool { return !t.Internal && !t.Hidden }

// exportFile answers /export/<type>.<ext>?where=…&order=… with the file.
func (s *Server) exportFile(w http.ResponseWriter, r *http.Request) {
	name, ext, _ := strings.Cut(r.PathValue("file"), ".")
	t, ok := s.app.Types.Get(name)
	if !ok || !exportable(t) {
		http.NotFound(w, r)
		return
	}
	f, ok := export.ByExt(t, ext)
	if !ok {
		http.Error(w, fmt.Sprintf("%s cannot be taken out as .%s; it can be %s", plural(t.Name), ext, formatList(t)), http.StatusNotFound)
		return
	}
	recs, err := s.exportRecords(t, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	if err := export.Write(&buf, f, t, recs, s.exportTitles); err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", f.Type)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s %s.%s"`, capitalize(plural(t.Name)), time.Now().Format("2006-01-02"), f.Ext))
	w.Write(buf.Bytes())
}

// exportRecords are the records a list page with this query shows.
func (s *Server) exportRecords(t *schema.Type, q url.Values) ([]*store.Record, error) {
	where, order := q["where"], q.Get("order")
	if len(where) > 0 || order != "" {
		return query.Filter(s.app.Store, t, where, order, 0, time.Now())
	}
	return s.app.Store.List(t.Name, store.ListOptions{})
}

func (s *Server) exportTitles(f schema.Field, id string) string { return s.refTitle(f, id) }

func formatList(t *schema.Type) string {
	var names []string
	for _, f := range export.For(t) {
		names = append(names, "."+f.Ext)
	}
	return strings.Join(names, ", ")
}

// exportLinks is the list page's way out: each format the type can be,
// with the page's own query.
func (s *Server) exportLinks(t *schema.Type, q url.Values) string {
	if !exportable(t) {
		return ""
	}
	query := ""
	if keep := (url.Values{"where": q["where"], "order": {q.Get("order")}}); len(q["where"]) > 0 || q.Get("order") != "" {
		if keep.Get("order") == "" {
			keep.Del("order")
		}
		query = "?" + keep.Encode()
	}
	var links []string
	for _, f := range export.For(t) {
		links = append(links, fmt.Sprintf(`<a class="sw-link" href="/export/%s.%s%s" download>%s<span class="sw-visually-hidden"> of these %s</span></a>`,
			t.Name, f.Ext, template.HTMLEscapeString(query), template.HTMLEscapeString(f.Label), template.HTMLEscapeString(plural(t.Name))))
	}
	return `<p class="sw-export sw-small">Download these: ` + strings.Join(links, ", ") + `.</p>`
}

// transcriptFile answers a recording's words as subtitles (.srt) or as
// plain text (.txt), from its text, so a correction goes out too.
func (s *Server) transcriptFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cues := s.heard(rec)
	if len(cues) == 0 {
		http.NotFound(w, r)
		return
	}
	title, _ := rec.Fields["title"].(string)
	ext := strings.TrimPrefix(filepathExt(r.URL.Path), ".")
	var body string
	switch ext {
	case "srt":
		w.Header().Set("Content-Type", "application/x-subrip; charset=utf-8")
		body = convert.SRT(cues)
	case "txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		body = convert.Transcript(cues) + "\n"
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, strings.ReplaceAll(title, `"`, ""), ext))
	w.Write([]byte(body))
}

func filepathExt(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 && !strings.Contains(p[i:], "/") {
		return p[i:]
	}
	return ""
}
