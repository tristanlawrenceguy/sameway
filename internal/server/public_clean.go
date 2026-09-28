package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What the internet is sent is the page as a reader has it, the same for
// a person and for an AI reading the HTML: the ways to change things and
// to reach the rest of the workspace are taken out of it, not hidden, so
// nothing in it leads where the internet may not go. A form that sends,
// a control bar, the chat, the log, the workspace's own links and the
// editor's templates go; a link to something not published keeps its
// words and loses its address.

// cleaned serves a published page and sends it rewritten for readers.
func (s *Server) cleaned(w http.ResponseWriter, r *http.Request, serve func(http.ResponseWriter, *http.Request)) {
	rec := httptest.NewRecorder()
	serve(rec, r)
	for k, v := range rec.Header() {
		if k != "Content-Length" {
			w.Header()[k] = v
		}
	}
	body := rec.Body.Bytes()
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		pub := s.Published()
		body = forReaders(body, func(path string) bool { return s.publicAllows(pub, path) || path == "/" })
	}
	w.WriteHeader(rec.Code)
	w.Write(body)
}

// forReaders rewrites a page for someone who may only read, going where
// allowed says.
func forReaders(src []byte, allowed func(path string) bool) []byte {
	z := html.NewTokenizer(bytes.NewReader(src))
	var out bytes.Buffer
	skip, skipTag := 0, ""
	var links []bool // for each open <a>, whether it lost its address
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return out.Bytes()
		}
		raw := append([]byte(nil), z.Raw()...)
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tok := z.Token()
			if skip > 0 {
				if tt == html.StartTagToken && tok.Data == skipTag {
					skip++
				}
				continue
			}
			if dropped(tok, allowed) {
				if tt == html.StartTagToken && !voidElement(tok.Data) {
					skip, skipTag = 1, tok.Data
				}
				continue
			}
			if tok.Data == "a" && tt == html.StartTagToken {
				gone := !reachable(attr(tok, "href"), allowed)
				links = append(links, gone)
				if gone {
					out.WriteString("<span>")
					continue
				}
			}
		case html.EndTagToken:
			tok := z.Token()
			if skip > 0 {
				if tok.Data == skipTag {
					skip--
				}
				continue
			}
			if tok.Data == "a" && len(links) > 0 {
				gone := links[len(links)-1]
				links = links[:len(links)-1]
				if gone {
					out.WriteString("</span>")
					continue
				}
			}
		default:
			if skip > 0 {
				continue
			}
		}
		out.Write(raw)
	}
}

// dropped says whether an element has no place on a page for readers.
func dropped(tok html.Token, allowed func(string) bool) bool {
	switch {
	case tok.Data == "form":
		return !strings.EqualFold(attr(tok, "method"), "get") || !reachable(attr(tok, "action"), allowed)
	case tok.Data == "template" && hasAttr(tok, "data-edit-fields"):
		return true
	case attr(tok, "data-block-component") == chat.ComponentName:
		return true
	}
	for _, c := range strings.Fields(attr(tok, "class")) {
		switch c {
		case "sw-bar", "sw-nav--more", "sw-activity", "sw-footer__agents", "sw-strip--header":
			return true
		}
	}
	return false
}

// reachable says whether an address leads somewhere a reader may go: off
// this site, on this page, or to something published.
func reachable(href string, allowed func(string) bool) bool {
	if href == "" || strings.HasPrefix(href, "#") || strings.Contains(href, "://") || strings.HasPrefix(href, "mailto:") {
		return true
	}
	if !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "//") {
		return true
	}
	path, _, _ := strings.Cut(href, "?")
	path, _, _ = strings.Cut(path, "#")
	return strings.HasPrefix(path, "/design/") || allowed(path)
}

func attr(tok html.Token, key string) string {
	for _, a := range tok.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasAttr(tok html.Token, key string) bool {
	for _, a := range tok.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func voidElement(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr":
		return true
	}
	return false
}

// publicFile says whether a kept file is part of something published: a
// block on a published tab, or a published record, points at it.
func (s *Server) publicFile(pub Published, id string) bool {
	// A file's parts, such as a video's captions, go with the file.
	id, _, _ = strings.Cut(id, "/")
	ref := "/files/" + id
	blocks, _ := s.app.Store.List(chat.BlockType, store.ListOptions{})
	for _, b := range blocks {
		canvas, _ := b.Fields["canvas"].(string)
		if _, ok := pub.Tabs[canvas]; !ok {
			continue
		}
		if raw, _ := json.Marshal(b.Fields["props"]); bytes.Contains(raw, []byte(ref)) {
			return true
		}
	}
	for typ := range pub.Types {
		recs, _ := s.app.Store.List(typ, store.ListOptions{})
		for _, r := range recs {
			if raw, _ := json.Marshal(r.Fields); bytes.Contains(raw, []byte(ref)) || bytes.Contains(raw, []byte(`"`+id+`"`)) {
				return true
			}
		}
	}
	return false
}
