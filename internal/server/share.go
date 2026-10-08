package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Saving from anywhere: a phone's Share button (the installed app is a
// share target, icon.go), a link in the bookmarks bar on a computer, or
// the form on /share. What arrives becomes one note: the words shared,
// the link, the page's own words read from it, and any files beside it,
// so a recipe, an article or a photo is kept the moment it is seen.

const shareType = "note"

var firstLink = regexp.MustCompile(`https?://\S+`)

// sharePage is the form, filled with whatever the bookmark sent, and how
// to save from a browser and from a phone.
func (s *Server) sharePage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<form method="post" action="/share" enctype="multipart/form-data" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Title", "name": "title", "value": q.Get("title"), "hint": "Left empty, the page's own title is used."})))
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Link", "name": "url", "type": "url", "value": q.Get("url"), "spellcheck": false, "hint": "A web page's words are read and kept with it, so it can be found and read later even if the page goes."})))
	b.WriteString(string(s.component("textarea", map[string]any{"label": "Words", "name": "text", "value": q.Get("text"), "rows": 4})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Save as a note", "type": "submit"})) + `</form>`)
	bookmark := "javascript:(()=>{const e=encodeURIComponent;open('" + s.origin(r) + "/share?url='+e(location.href)+'&title='+e(document.title)+'&text='+e(getSelection().toString()),'_blank')})()"
	b.WriteString(`<h2>From your browser</h2><p>Drag this link to your bookmarks bar: <a class="sw-link" href="` + esc(bookmark) + `">Save to Sameway</a>. On any page, press it to save the page here, with any words you selected.</p>`)
	b.WriteString(`<h2>From your phone</h2><p>Open Sameway on an Android phone and install it (<a class="sw-link" href="/help#help-app">how</a>): it is then in the Share menu of every app, so a page, a photo or some words shared to Sameway are saved as a note. On an iPhone, the bookmark above works in Safari.</p>`)
	s.page(w, r, "Save to Sameway", template.HTML(b.String()), pageOptions{Lede: "Keep a page, a photo or a few words the moment you see them."})
}

// origin is how the one asking reached this server, for a link that
// comes back to it.
func (s *Server) origin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// shareSave makes the note: from the form, a bookmark or a phone's share.
func (s *Server) shareSave(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.app.Types.Get(shareType); !ok {
		s.failed(w, r, "Not saved", errors.New("this workspace has no notes to save into"), "/")
		return
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, maxFile+(1<<20))
		r.ParseMultipartForm(8 << 20)
	} else {
		r.ParseForm()
	}
	title := oneLineOf(r.FormValue("title"))
	text := strings.TrimSpace(r.FormValue("text"))
	link := strings.TrimSpace(r.FormValue("url"))
	// A phone often sends the link inside the words, or as the words.
	fromText := false
	if link == "" {
		link = firstLink.FindString(text)
		fromText = link != ""
	}
	var parts []string
	if text != "" && text != link {
		parts = append(parts, text)
	}
	said := ""
	if link != "" {
		u, err := url.Parse(link)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			s.failed(w, r, "Not saved", errors.New("the link must begin http:// or https://"), "/share")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		pageTitle, words, err := readPage(ctx, link) // share_read.go
		cancel()
		if title == "" {
			title = pageTitle
		}
		name := pageTitle
		if name == "" {
			name = u.Host
		}
		if !fromText || pageTitle != "" {
			parts = append(parts, "["+name+"]("+link+")")
		}
		if err != nil {
			said = " Its page could not be read (" + err.Error() + "), so only the link is kept."
		} else if words != "" {
			parts = append(parts, "---", words)
		}
	}
	files, err := s.shareFiles(r)
	if err != nil {
		s.failed(w, r, "Not saved", err, "/share")
		return
	}
	for _, f := range files {
		parts = append(parts, "["+f.name+"](/t/"+FileType+"/"+f.id+")")
	}
	if len(parts) == 0 {
		s.failed(w, r, "Not saved", errors.New("nothing came to save: add a link, some words or a file"), "/share")
		return
	}
	if words := strings.TrimSpace(strings.Replace(text, link, "", 1)); title == "" && words != "" {
		title = clipRunes(oneLineOf(strings.SplitN(words, "\n", 2)[0]), 80)
	}
	if title == "" && len(files) > 0 {
		title = files[0].name
	}
	if title == "" {
		title = "Saved " + time.Now().Format("2 Jan 15:04")
	}
	rec, act, err := chat.WriteAs(s.app.Store, s.who(r), "created", shareType, "", map[string]any{"title": clipRunes(title, 200), "body": strings.Join(parts, "\n\n")})
	if err != nil {
		s.failed(w, r, "Not saved", err, "/share")
		return
	}
	if len(files) > 0 {
		said += fmt.Sprintf(" %s kept in your files.", schema.Count(len(files), FileType))
	}
	s.tellAt(w, r, outcome{Title: "Saved", Text: title + " is in your notes." + said, Undo: act, Of: title}, "/t/"+shareType+"/"+rec.ID)
}

type sharedFile struct{ id, name string }

// shareFiles keeps each file shared, as the upload form does.
func (s *Server) shareFiles(r *http.Request) ([]sharedFile, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	var out []sharedFile
	for _, h := range r.MultipartForm.File["files"] {
		f, err := h.Open()
		if err != nil {
			return nil, fmt.Errorf("%s did not arrive whole", h.Filename)
		}
		rec, path, err := s.keepFile(s.who(r), f, h.Filename, "", "")
		f.Close()
		if err != nil {
			return nil, err
		}
		s.readKept(rec.ID, h.Filename, path, false)
		name, _ := rec.Fields["title"].(string)
		out = append(out, sharedFile{rec.ID, name})
	}
	return out, nil
}

func oneLineOf(s string) string { return strings.Join(strings.Fields(s), " ") }

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return strings.TrimSpace(string(r[:n-1])) + "…"
	}
	return s
}

func (s *Server) shareRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /share", s.sharePage)
	m.HandleFunc("POST /share", s.shareSave)
}
