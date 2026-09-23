package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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
	// Scripts is what reading with the page's scripts run adds, when it
	// was asked for.
	Scripts *Scripted `json:"scripts,omitempty"`
}

// Scripted is what only a page with its scripts run can say: what each
// step reached, what has focus, where Tab goes, and what went wrong.
type Scripted struct {
	Did        []string `json:"did,omitempty"`
	Focused    string   `json:"focused,omitempty"`
	FocusOrder []string `json:"focus_order"`
	// Errors is every exception, console error and failed request on the
	// page. Empty means none; it is always present.
	Errors []string `json:"errors"`
}

// lookAsk is what a look is asked for, however it came.
type lookAsk struct {
	Path      string            `json:"path"`
	Method    string            `json:"method"`
	Form      map[string]string `json:"form"`
	Component string            `json:"component"`
	Props     map[string]any    `json:"props"`
	// Scripts reads the page in a browser, its scripts run; Steps are
	// what a person does there first, and imply it.
	Scripts bool        `json:"scripts"`
	Steps   []look.Step `json:"steps"`
	// Only, Kind and Name narrow a long answer: the sections wanted, and
	// the controls of one kind or with words in their name.
	Only []string `json:"only"`
	Kind string   `json:"kind"`
	Name string   `json:"name"`
}

// apiLook serves the outline of a page, or of one component rendered from
// props, so an agent verifies what a person would get. GET takes a path;
// POST takes {path, method, form} to do what a person does and see where
// it leads, or {component, props} for a fragment. With scripts (or steps)
// the page is read in a headless browser with its scripts run, after the
// steps a person takes there.
func (s *Server) apiLook(w http.ResponseWriter, r *http.Request) {
	var ask lookAsk
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		ask.Path, ask.Kind, ask.Name = q.Get("path"), q.Get("kind"), q.Get("name")
		ask.Scripts = q.Get("scripts") == "1" || q.Get("scripts") == "true"
		if only := q.Get("only"); only != "" {
			ask.Only = strings.Split(only, ",")
		}
	} else {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if len(raw) > 0 {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&ask); err != nil {
				writeError(w, errors.New("body must be a JSON object of path, method, form, component, props, scripts, steps, only, kind and name: "+err.Error()))
				return
			}
		}
	}
	for _, section := range ask.Only {
		if !lookSections[strings.TrimSpace(section)] {
			writeError(w, fmt.Errorf("only takes landmarks, headings, controls, live and components; not %q", section))
			return
		}
	}
	if ask.Component != "" {
		s.lookAtComponent(w, ask.Component, ask.Props)
		return
	}
	if ask.Scripts || len(ask.Steps) > 0 {
		if ask.Method != "" || len(ask.Form) > 0 {
			writeError(w, errors.New("with scripts, a form is filled and sent as steps: type into its fields and press its button"))
			return
		}
		s.lookScripted(w, r, ask)
		return
	}
	form := url.Values{}
	for k, v := range ask.Form {
		form.Set(k, v)
	}
	method := strings.ToUpper(ask.Method)
	if method == "" {
		method = http.MethodGet
		if len(form) > 0 {
			method = http.MethodPost
		}
	}
	s.lookAt(w, ask, method, form)
}

func (s *Server) lookAt(w http.ResponseWriter, ask lookAsk, method string, form url.Values) {
	path := ask.Path
	if !lookable(path) {
		writeError(w, errors.New("path must be a page on this server, such as /t/note"))
		return
	}
	rec := s.request(method, path, form)
	out := Looked{Path: path, Status: rec.Code}
	// A person who posts a form is sent on; what they see is where they land.
	if loc := rec.Header().Get("Location"); rec.Code >= 300 && rec.Code < 400 && loc != "" {
		out.Landed = loc
		rec = s.request(http.MethodGet, loc, nil, rec.Result().Cookies()...)
	}
	outline, err := look.Page(rec.Body.String())
	if err != nil {
		writeError(w, err)
		return
	}
	out.Outline = narrow(outline, ask)
	writeJSON(w, http.StatusOK, out)
}

func lookable(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")
}

// escaped is a path as it travels, however it was written: an agent may
// write /chat?prompt=Create a note. as a person would say it.
func escaped(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		return url.PathEscape(path)
	}
	u.RawQuery = u.Query().Encode()
	return u.RequestURI()
}

// lookScripted serves this workspace to a headless browser for as long as
// the reading takes, on a port of its own on this machine, and reads the
// page there.
func (s *Server) lookScripted(w http.ResponseWriter, r *http.Request, ask lookAsk) {
	if !lookable(ask.Path) {
		writeError(w, errors.New("path must be a page on this server, such as /t/note"))
		return
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		writeError(w, err)
		return
	}
	srv := &http.Server{Handler: s.mux}
	go srv.Serve(ln)
	defer srv.Close()
	run, err := look.Scripted(r.Context(), "http://"+ln.Addr().String()+escaped(ask.Path), ask.Steps)
	if errors.Is(err, look.ErrNoBrowser) {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": apiError{Code: "no_browser", Message: err.Error()}})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: err.Error()}})
		return
	}
	outline, err := look.Page(run.HTML)
	if err != nil {
		writeError(w, err)
		return
	}
	out := Looked{Path: ask.Path, Status: http.StatusOK, Outline: narrow(outline, ask),
		Scripts: &Scripted{Did: run.Did, Focused: run.Focused, FocusOrder: run.FocusOrder, Errors: run.Errors}}
	if out.Scripts.Errors == nil {
		out.Scripts.Errors = []string{}
	}
	if out.Scripts.FocusOrder == nil {
		out.Scripts.FocusOrder = []string{}
	}
	if run.URL != ask.Path {
		out.Landed = run.URL
	}
	writeJSON(w, http.StatusOK, out)
}

// lookSections are the parts of an outline only can keep.
var lookSections = map[string]bool{"landmarks": true, "headings": true, "controls": true, "live": true, "components": true}

// narrow keeps what was asked for from a long outline: the sections named
// in only, and the controls of kind, or with name among their words. The
// problems always stay, so an agent can always check them.
func narrow(o *look.Outline, ask lookAsk) *look.Outline {
	if len(ask.Only) > 0 {
		keep := map[string]bool{}
		for _, section := range ask.Only {
			keep[strings.TrimSpace(section)] = true
		}
		if !keep["landmarks"] {
			o.Landmarks = nil
		}
		if !keep["headings"] {
			o.Headings = nil
		}
		if !keep["controls"] {
			o.Controls = nil
		}
		if !keep["live"] {
			o.Live = nil
		}
		if !keep["components"] {
			o.Components = nil
		}
	}
	if ask.Kind != "" || ask.Name != "" {
		var kept []look.Control
		for _, c := range o.Controls {
			if (ask.Kind == "" || c.Kind == ask.Kind) && (ask.Name == "" || strings.Contains(strings.ToLower(c.Name), strings.ToLower(ask.Name))) {
				kept = append(kept, c)
			}
		}
		o.Controls = kept
	}
	return o
}

func (s *Server) request(method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	path = escaped(path)
	var body io.Reader
	if form != nil && method != http.MethodGet {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, c := range cookies {
		req.AddCookie(c)
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
