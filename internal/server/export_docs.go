package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/content"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/look"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record goes out as a document: Markdown, a web page, Word or PDF. The
// web page is the record's own page as a reader has it (no controls, the
// design's styles and its pictures inside it), so it reads the same as
// here with nothing else needed; the PDF is that page printed, tagged, by
// the Chrome or Edge on this computer; Word is written from its words.

var docFormats = []struct{ ext, label, typ string }{
	{"md", "Markdown", "text/markdown; charset=utf-8"},
	{"html", "Web page", "text/html; charset=utf-8"},
	{"docx", "Word", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	{"pdf", "PDF", "application/pdf"},
}

// recordFormats are the files one record can be: the documents always,
// and a calendar entry or a contact when its list offers those and the
// record has the day a calendar needs.
func recordFormats(t *schema.Type, rec *store.Record) []string {
	out := []string{}
	for _, f := range docFormats {
		out = append(out, f.ext)
	}
	for _, f := range export.For(t) {
		if f == export.VCard || f == export.ICS && export.Dated(t, rec) {
			out = append(out, f.Ext)
		}
	}
	return out
}

// documentLinks is a record page's way out, after the record: the export
// component, with a size where making the file is cheap. The web page and
// the PDF are made by rendering and printing, so they say none.
func (s *Server) documentLinks(r *http.Request, t *schema.Type, rec *store.Record) string {
	if !exportable(t) {
		return ""
	}
	var items []any
	for _, ext := range recordFormats(t, rec) {
		item := map[string]any{"href": "/export/" + t.Name + "/" + rec.ID + "." + ext, "format": ext}
		if ext != "html" && ext != "pdf" {
			if body, _, err := s.recordFile(r, t, rec, ext); err == nil {
				item["size"] = sizeWords(int64(len(body)))
			}
		}
		items = append(items, item)
	}
	// Named by what it is of, not its title, which the heading says once.
	return string(s.component("export", map[string]any{"what": "this " + schema.Words(t.Name), "items": items}))
}

// exportDocument answers /export/<type>/<id>.<ext>.
func (s *Server) exportDocument(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	id, ext, _ := strings.Cut(r.PathValue("file"), ".")
	if !ok || !exportable(t) {
		http.NotFound(w, r)
		return
	}
	rec, err := s.app.Store.Get(t.Name, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !has(recordFormats(t, rec), ext) {
		http.Error(w, "this "+schema.Words(t.Name)+" can be taken out as ."+strings.Join(recordFormats(t, rec), ", ."), http.StatusNotFound)
		return
	}
	body, typ, err := s.recordFile(r, t, rec, ext)
	if errors.Is(err, look.ErrNoBrowser) {
		http.Error(w, "A PDF is printed by Chrome or Edge on the computer that hosts the workspace, and there is none. The web page (.html) can be printed to PDF from any browser.", http.StatusNotImplemented)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", typ)
	attachment(w, s.title(t, rec)+"."+ext)
	w.Write(body)
}

// recordFile makes one record's file and says its type. The internet is
// given only the fields a published page shows.
func (s *Server) recordFile(r *http.Request, t *schema.Type, rec *store.Record, ext string) ([]byte, string, error) {
	if isPublic(r) {
		shown := &store.Record{ID: rec.ID, Type: rec.Type, CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt, Fields: map[string]any{}}
		for _, f := range t.Shown() {
			if v, ok := rec.Fields[f.Name]; ok {
				shown.Fields[f.Name] = v
			}
		}
		rec = shown
	}
	for _, f := range docFormats {
		if f.ext != ext {
			continue
		}
		switch ext {
		case "md":
			return []byte(s.recordMarkdown(t, rec, true)), f.typ, nil
		case "html":
			body, err := s.recordHTML(t, rec)
			return body, f.typ, err
		case "docx":
			var buf bytes.Buffer
			err := export.DOCX(&buf, s.title(t, rec), s.lang(), s.recordMarkdown(t, rec, false))
			return buf.Bytes(), f.typ, err
		case "pdf":
			body, err := s.recordPDF(r.Context(), t, rec)
			return body, f.typ, err
		}
	}
	f, _ := export.ByExt(t, ext)
	var buf bytes.Buffer
	err := export.Write(&buf, f, t, []*store.Record{rec}, s.RefTitle)
	return buf.Bytes(), f.Type, err
}

func (s *Server) lang() string {
	if l := strings.TrimSpace(s.app.Workspace.Config.UI.Language); l != "" {
		return l
	}
	return "en"
}

// recordMarkdown is a record as Markdown: its title (when heading says),
// its facts as a list of names and values, then its text.
func (s *Server) recordMarkdown(t *schema.Type, rec *store.Record, heading bool) string {
	var b strings.Builder
	if heading {
		b.WriteString("# " + s.title(t, rec) + "\n\n")
	}
	text := ""
	for _, f := range t.Shown() {
		if f.Name == t.Title {
			continue
		}
		if f.Type == "markdown" && text == "" {
			text, _ = rec.Fields[f.Name].(string)
			continue
		}
		if v := export.Value(f, rec.Fields[f.Name], s.RefTitle); v != "" && v != "no" {
			fmt.Fprintf(&b, "- **%s:** %s\n", f.Display(), v)
		}
	}
	if text != "" {
		b.WriteString("\n" + strings.TrimSpace(text) + "\n")
	}
	return b.String()
}

var mainPart = regexp.MustCompile(`(?s)<main\b[^>]*>(.*)</main>`)
var picture = regexp.MustCompile(`src="/files/([A-Za-z0-9]+)"`)

// recordHTML is the record's page as a reader has it, whole in one file.
func (s *Server) recordHTML(t *schema.Type, rec *store.Record) ([]byte, error) {
	page := s.request(http.MethodGet, "/t/"+t.Name+"/"+rec.ID, nil)
	if page.Code != http.StatusOK {
		return nil, fmt.Errorf("the page could not be read (%d)", page.Code)
	}
	clean := forReaders(page.Body.Bytes(), func(string) bool { return false })
	m := mainPart.FindSubmatch(clean)
	if m == nil {
		return nil, errors.New("the page has no main part")
	}
	body := picture.ReplaceAllFunc(m[1], func(src []byte) []byte {
		id := string(picture.FindSubmatch(src)[1])
		f, err := s.app.Store.Get(FileType, id)
		if err != nil {
			return src
		}
		path, ok := s.storedPath(f)
		st, err := os.Stat(path)
		if !ok || err != nil || st.Size() > 8<<20 {
			return src
		}
		data, _ := os.ReadFile(path)
		return []byte(`src="data:` + http.DetectContentType(data) + `;base64,` + base64.StdEncoding.EncodeToString(data) + `"`)
	})
	var out bytes.Buffer
	fmt.Fprintf(&out, "<!doctype html>\n<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<title>%s</title>\n<style>\n%s\n</style>\n</head>\n<body>\n<main class=\"sw-main\">%s</main>\n</body>\n</html>\n",
		template.HTMLEscapeString(s.lang()), template.HTMLEscapeString(s.title(t, rec)), s.css, body)
	return out.Bytes(), nil
}

// recordPDF prints the record's web page in the headless browser, from a
// port of this computer's own, for as long as it takes.
func (s *Server) recordPDF(ctx context.Context, t *schema.Type, rec *store.Record) ([]byte, error) {
	doc, err := s.recordHTML(t, rec)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(doc)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return look.PrintPDF(ctx, "http://"+ln.Addr().String()+"/")
}

// exportEverything answers /export/workspace.zip: the whole workspace, its
// owner's to take.
func (s *Server) exportEverything(w http.ResponseWriter, r *http.Request) {
	name := s.app.Workspace.Config.Name
	w.Header().Set("Content-Type", "application/zip")
	attachment(w, name+" "+time.Now().Format("2006-01-02")+".zip")
	if err := export.Everything(w, name, s.app.Store, s.app.Types, s.app.Mirror, s.app.Workspace.FilesDir(), s.RefTitle); err != nil {
		log.Printf("export: %v", err)
	}
}

// takeEverything is the owner's way to take the whole workspace, with
// about how big it is before it is packed: the files as kept and each
// record's Markdown.
func (s *Server) takeEverything(r *http.Request) string {
	if !chat.VisitorOf(r.Context()).Owner() {
		return ""
	}
	var n int64
	filepath.WalkDir(s.app.Workspace.FilesDir(), func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, err := d.Info(); err == nil {
				n += info.Size()
			}
		}
		return nil
	})
	for _, t := range s.app.Types.Types {
		if recs, err := s.app.Store.List(t.Name, store.ListOptions{}); err == nil && exportable(t) {
			for _, rec := range recs {
				data, _ := content.Encode(t, rec)
				n += int64(len(data))
			}
		}
	}
	return string(s.component("export", map[string]any{"what": "everything in " + s.app.Workspace.Config.Name,
		"note":  "Every record as Markdown, every file as it was added, and a spreadsheet of each kind of record. Conversations and the log stay here.",
		"items": []any{map[string]any{"href": "/export/workspace.zip", "format": "zip", "size": "about " + sizeWords(n)}}}))
}
