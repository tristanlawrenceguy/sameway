package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// Looked is a page as an agent sees it: what was asked for, what came back,
// where it led, and the outline of what is on it.
type Looked struct {
	Path   string `json:"path"`
	Status int    `json:"status"`
	// Landed is the page a redirect ended on, when the request was one a
	// person makes and the server sent them somewhere.
	Landed  string        `json:"landed,omitempty"`
	Outline *look.Outline `json:"outline"`
}

// apiLook serves the outline of a page, or of one component rendered from
// props, so an agent verifies what a person would get without a browser.
// GET takes a path; POST takes {path, method, form} to do what a person
// does and see where it leads, or {component, props} for a fragment.
func (s *Server) apiLook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.lookAt(w, r.URL.Query().Get("path"), http.MethodGet, nil)
		return
	}
	var body struct {
		Path      string            `json:"path"`
		Method    string            `json:"method"`
		Form      map[string]string `json:"form"`
		Component string            `json:"component"`
		Props     map[string]any    `json:"props"`
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeError(w, errors.New("body must be a JSON object: "+err.Error()))
			return
		}
	}
	if body.Component != "" {
		s.lookAtComponent(w, body.Component, body.Props)
		return
	}
	form := url.Values{}
	for k, v := range body.Form {
		form.Set(k, v)
	}
	method := strings.ToUpper(body.Method)
	if method == "" {
		method = http.MethodGet
		if len(form) > 0 {
			method = http.MethodPost
		}
	}
	s.lookAt(w, body.Path, method, form)
}

func (s *Server) lookAt(w http.ResponseWriter, path, method string, form url.Values) {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		writeError(w, errors.New("path must be a page on this server, such as /t/note"))
		return
	}
	rec := s.request(method, path, form)
	out := Looked{Path: path, Status: rec.Code}
	// A person who posts a form is sent on; what they see is where they land.
	if loc := rec.Header().Get("Location"); rec.Code >= 300 && rec.Code < 400 && loc != "" {
		out.Landed = loc
		rec = s.request(http.MethodGet, loc, nil)
	}
	outline, err := look.Page(rec.Body.String())
	if err != nil {
		writeError(w, err)
		return
	}
	out.Outline = outline
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) request(method, path string, form url.Values) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil && method != http.MethodGet {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

// lookAtComponent renders one component from props and reads the fragment,
// so an agent sees what a block would be before adding it, and what a
// screen reader would be stuck on.
func (s *Server) lookAtComponent(w http.ResponseWriter, name string, props map[string]any) {
	c, ok := s.app.Registry.Get(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found",
			Message: fmt.Sprintf("no component %q; the components are %s", name, strings.Join(s.app.Registry.Names(), ", "))}})
		return
	}
	if props == nil {
		props = map[string]any{}
	}
	if _, err := c.Validate(props); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: err.Error()}})
		return
	}
	html, err := s.app.Registry.Render(name, props)
	if err != nil {
		writeError(w, err)
		return
	}
	outline, err := look.Fragment(string(html))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"component": name, "html": string(html), "outline": outline})
}
