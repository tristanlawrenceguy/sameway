package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Saving from anywhere: a phone's Share button (the installed app is a
// share target, icon.go), a link in the bookmarks bar on a computer, or
// the form on /share (share_page.go). What arrives becomes one note: the
// words shared, the link, the page's own words read from it, and any
// files beside it, so a recipe, an article or a photo is kept the moment
// it is seen. The same link shared again with nothing new is not a second
// note: the person is taken to the one they have.

const shareType = "note"

var firstLink = regexp.MustCompile(`https?://\S+`)

// shared is what arrived: from the form, a bookmark or a phone.
type shared struct {
	title, text, link string
	fromText          bool // the link was found in the words, as a phone sends it
}

func shareInput(w http.ResponseWriter, r *http.Request) shared {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, media.MaxFile+(1<<20))
		r.ParseMultipartForm(8 << 20)
	} else {
		r.ParseForm()
	}
	in := shared{title: oneLineOf(r.FormValue("title")), text: strings.TrimSpace(r.FormValue("text")), link: strings.TrimSpace(r.FormValue("url"))}
	if in.link == "" {
		in.link = firstLink.FindString(in.text)
		in.fromText = in.link != ""
	}
	return in
}

// shareSave makes the note: from the form, a bookmark or a phone's share.
func (s *Server) shareSave(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.app.Types.Get(shareType); !ok {
		s.failed(w, r, "Not saved", errors.New("this workspace has no notes to save into"), "/")
		return
	}
	in := shareInput(w, r)
	var parts []string
	title, said := in.title, ""
	if in.link != "" {
		u, err := keptLink(in.link)
		if err != nil {
			s.shareAgain(w, r, in, "url", err)
			return
		}
		in.text = strings.Replace(in.text, in.link, u.String(), 1)
		if rec := s.savedBefore(u.String()); rec != nil && noneNew(r, rec, in) {
			name, _ := rec.Fields["title"].(string)
			s.tellAt(w, r, outcome{Title: "Already saved", Text: name + " is in your notes, saved " + s.savedOn(rec.CreatedAt) + "."}, "/t/"+shareType+"/"+rec.ID)
			return
		}
		in.link = u.String()
		var pageTitle string
		pageTitle, parts, said = s.readShared(r, u, in.fromText)
		if title == "" {
			title = pageTitle
		}
	}
	if in.text != "" && in.text != in.link {
		parts = append([]string{in.text}, parts...)
	}
	files, err := s.shareFiles(r)
	if err != nil {
		s.shareAgain(w, r, in, "", err)
		return
	}
	for _, f := range files {
		parts = append(parts, "["+f.name+"](/t/"+FileType+"/"+f.id+")")
	}
	if len(parts) == 0 {
		s.shareAgain(w, r, in, "url", errors.New("nothing came to save: add a link, some words or a file"))
		return
	}
	title = s.shareTitle(title, in, files)
	rec, act, err := s.writeShare(r, clipRunes(title, 200), strings.Join(parts, "\n\n"), in.link == "")
	if err != nil {
		s.shareAgain(w, r, in, "", err)
		return
	}
	if len(files) > 0 {
		said += fmt.Sprintf(" %s kept in your files.", schema.Count(len(files), FileType))
	}
	s.tellAt(w, r, outcome{Title: "Saved", Text: title + " is in your notes." + said, Undo: act, Of: title}, "/t/"+shareType+"/"+rec.ID)
}

// noneNew says a note already holds all that came: no files, and the
// words, if any, already in it.
func noneNew(r *http.Request, rec *store.Record, in shared) bool {
	if r.MultipartForm != nil && len(r.MultipartForm.File["files"]) > 0 {
		return false
	}
	body, _ := rec.Fields["body"].(string)
	return in.text == "" || in.text == in.link || strings.Contains(body, in.text)
}

// readShared reads the page a link names: its title, the parts of the
// note it makes (the link, then the page's words), and what to add to
// the outcome when it could not be read.
func (s *Server) readShared(r *http.Request, u *url.URL, fromText bool) (title string, parts []string, said string) {
	ctx, cancel := context.WithTimeout(r.Context(), pageWait)
	defer cancel()
	title, words, err := readPage(ctx, s.pages(), u.String()) // share_read.go
	name := title
	if name == "" {
		name = u.Host
	}
	if !fromText || title != "" {
		parts = append(parts, "["+name+"]("+u.String()+")")
	}
	if err != nil {
		said = " Its page could not be read (" + err.Error() + "), so only the link is kept."
	} else if words != "" {
		parts = append(parts, "---", words)
	}
	return title, parts, said
}

// shareTitle is the note's title: the one given or the page's, else the
// first line of the words, the first file's name, or when it was saved.
func (s *Server) shareTitle(title string, in shared, files []sharedFile) string {
	if words := strings.TrimSpace(strings.Replace(in.text, in.link, "", 1)); title == "" && words != "" {
		title = clipRunes(oneLineOf(strings.SplitN(words, "\n", 2)[0]), 80)
	}
	if title == "" && len(files) > 0 {
		title = files[0].name
	}
	if title == "" {
		title = "Saved " + s.now().Format("2 Jan 15:04")
	}
	return title
}

// shareAgain says why nothing was saved, on the form filled again with
// what came, so nothing has to be typed twice. A file has to be chosen
// again: a page cannot be given one back.
func (s *Server) shareAgain(w http.ResponseWriter, r *http.Request, in shared, field string, err error) {
	q := url.Values{}
	for k, v := range map[string]string{"url": in.link, "title": in.title, "text": in.text} {
		if v != "" && !(k == "url" && in.fromText) {
			q.Set(k, v)
		}
	}
	to := "/share"
	if len(q) > 0 {
		to += "?" + q.Encode()
	}
	o := outcome{Failed: true, Title: "Not saved", Text: web.PlainError(err)}
	if field != "" {
		o.Text, o.Problems = "", []problem{{Field: field, Text: web.PlainError(err)}}
	}
	s.tellAt(w, r, o, to)
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
		rec, path, err := s.media.KeepFile(s.who(r), f, h.Filename, "", "")
		f.Close()
		if err != nil {
			return nil, err
		}
		s.media.ReadKept(rec.ID, h.Filename, path, false)
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
