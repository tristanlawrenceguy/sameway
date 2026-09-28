package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

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

// documentLinks is a record page's way out as a document.
func (s *Server) documentLinks(t *schema.Type, rec *store.Record) string {
	if !exportable(t) {
		return ""
	}
	var links []string
	for _, f := range docFormats {
		// Named by what it is of, not its title, which the heading says once.
		links = append(links, fmt.Sprintf(`<a class="sw-link" href="/export/%s/%s.%s" download>%s<span class="sw-visually-hidden"> of this %s</span></a>`, t.Name, rec.ID, f.ext, f.label, template.HTMLEscapeString(t.Name)))
	}
	return `<p class="sw-export sw-small">Download this: ` + strings.Join(links, ", ") + `.</p>`
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
	title := s.title(t, rec)
	var body []byte
	typ := ""
	for _, f := range docFormats {
		if f.ext == ext {
			typ = f.typ
		}
	}
	switch ext {
	case "md":
		body = []byte(s.recordMarkdown(t, rec, true))
	case "html":
		body, err = s.recordHTML(t, rec)
	case "docx":
		var buf bytes.Buffer
		err = export.DOCX(&buf, title, s.lang(), s.recordMarkdown(t, rec, false))
		body = buf.Bytes()
	case "pdf":
		body, err = s.recordPDF(r.Context(), t, rec)
	default:
		http.Error(w, "a record can be taken out as .md, .html, .docx or .pdf", http.StatusNotFound)
		return
	}
	if errors.Is(err, look.ErrNoBrowser) {
		http.Error(w, "A PDF is printed by Chrome or Edge on the computer that hosts the workspace, and there is none. The web page (.html) can be printed to PDF from any browser.", http.StatusNotImplemented)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", typ)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, strings.NewReplacer(`"`, "", "/", "-", `\`, "-").Replace(title), ext))
	w.Write(body)
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
		if v := export.Value(f, rec.Fields[f.Name], s.exportTitles); v != "" && v != "no" {
			fmt.Fprintf(&b, "- **%s:** %s\n", fieldLabel(f), v)
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
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s %s.zip"`, strings.NewReplacer(`"`, "", "/", "-", `\`, "-").Replace(name), time.Now().Format("2006-01-02")))
	if err := export.Everything(w, name, s.app.Store, s.app.Types, s.app.Mirror, s.app.Workspace.FilesDir(), s.exportTitles); err != nil {
		log.Printf("export: %v", err)
	}
}
