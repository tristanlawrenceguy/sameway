package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Every action on a page is an agent's too. A person presses a button and
// lands back where they were with one message; an agent posts to the same
// address with Accept: application/json and gets that message as JSON,
// sending its fields as a form or as a JSON object. One handler, so an
// action cannot exist for one and not the other: undo, the workspaces,
// accepting a proposal were each a person's alone until an agent went
// looking, because each was written as a page first and nothing made it
// anything else.

// wantsJSON says whether the request asked to be answered in JSON.
func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// pageAction is a request an agent makes of a page's form: a POST outside
// /api, /mcp and /hook, which answer JSON already. A JSON body is an
// agent's whatever it accepts: no browser form sends one, and read as a
// form it was nothing, so the handler saw no fields and the agent was sent
// back with nothing done and nothing said (the agent evaluation, C-t5).
func pageAction(r *http.Request) bool {
	sentJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
	if r.Method != http.MethodPost || !wantsJSON(r) && !sentJSON {
		return false
	}
	for _, p := range []string{"/api/", "/mcp", "/hook/", "/sync", "/canvas/measure"} {
		if strings.HasPrefix(r.URL.Path, p) {
			return false
		}
	}
	return true
}

// forAgents readies an agent's page action: a JSON body becomes the form
// the handler reads, and the answer is JSON whatever the handler writes.
// finish must be called once the handler returns.
func forAgents(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request, func()) {
	if !pageAction(r) {
		return w, r, func() {}
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if form, err := jsonForm(r.Body); err == nil {
			r.Body = io.NopCloser(strings.NewReader(form.Encode()))
			r.ContentLength = int64(len(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	aw := &agentWriter{w: w, header: http.Header{}}
	return aw, r, aw.finish
}

// serve answers a request, an agent's page action in JSON.
func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.doOnce(w, r, func(w http.ResponseWriter, r *http.Request) { // once.go
		w, r, finish := forAgents(w, r)
		s.mux.ServeHTTP(w, r)
		finish()
	})
}

// jsonForm reads a JSON object as the form a page would have sent: a list
// is the field repeated, a number or yes-no its words, an object its JSON.
func jsonForm(body io.Reader) (url.Values, error) {
	var in map[string]any
	if err := json.NewDecoder(io.LimitReader(body, 4<<20)).Decode(&in); err != nil {
		return nil, err
	}
	form := url.Values{}
	for k, v := range in {
		vals, ok := v.([]any)
		if !ok {
			vals = []any{v}
		}
		for _, one := range vals {
			switch x := one.(type) {
			case string:
				form.Add(k, x)
			case nil:
				form.Add(k, "")
			case map[string]any:
				raw, _ := json.Marshal(x)
				form.Add(k, string(raw))
			default:
				form.Add(k, fmt.Sprint(x))
			}
		}
	}
	return form, nil
}

// tellJSON is an outcome said to an agent: what a person would read on the
// page they land on, and where that is.
func tellJSON(w http.ResponseWriter, o outcome, to string) {
	out := map[string]any{"ok": !o.Failed, "title": o.Title, "location": to}
	if o.Text != "" {
		out["text"] = o.Text
	}
	if o.Undo != "" {
		out["undo"] = "/activity/" + o.Undo + "/undo"
	}
	status := http.StatusOK
	if o.Failed {
		status = http.StatusBadRequest
	}
	if len(o.Problems) > 0 {
		fields := map[string]string{}
		for _, p := range o.Problems {
			fields[p.Field] = p.Text
		}
		out["fields"], status = fields, http.StatusUnprocessableEntity
	}
	writeJSON(w, status, out)
}

// agentWriter turns whatever a page action writes into JSON: JSON passes
// through, a redirect says where it leads, and a page or plain text is cut
// down to what it says. A page still counts as a fault (see
// TestEveryPageActionAnswersAnAgent): its handler should end in tell.
type agentWriter struct {
	w      http.ResponseWriter
	header http.Header
	status int
	buf    bytes.Buffer
	pass   bool // the handler answered in JSON itself
	done   bool
}

func (a *agentWriter) Header() http.Header { return a.header }

func (a *agentWriter) WriteHeader(code int) {
	if a.status != 0 {
		return
	}
	a.status = code
	if strings.HasPrefix(a.header.Get("Content-Type"), "application/json") {
		a.pass = true
		for k, v := range a.header {
			a.w.Header()[k] = v
		}
		a.w.WriteHeader(code)
	}
}

func (a *agentWriter) Write(p []byte) (int, error) {
	if a.status == 0 {
		a.WriteHeader(http.StatusOK)
	}
	if a.pass {
		return a.w.Write(p)
	}
	return a.buf.Write(p)
}

func (a *agentWriter) Flush() {
	if f, ok := a.w.(http.Flusher); ok && a.pass {
		f.Flush()
	}
}

var (
	tagRE   = regexp.MustCompile(`(?s)<[^>]*>`)
	h1RE    = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)
	alertRE = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*sw-alert[ "][^>]*>(.*?)</div>\s*</div>`)
)

func words(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagRE.ReplaceAllString(s, " "))), " ")
}

func (a *agentWriter) finish() {
	if a.pass || a.done {
		return
	}
	a.done = true
	code := a.status
	if code == 0 {
		code = http.StatusOK
	}
	w := a.w
	if loc := a.header.Get("Location"); code >= 300 && code < 400 {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "location": loc})
		return
	}
	body := a.buf.String()
	out := map[string]any{"ok": code < 400}
	if strings.HasPrefix(a.header.Get("Content-Type"), "text/html") || strings.Contains(body, "<html") {
		out["answered_with_page"] = true
		if m := h1RE.FindStringSubmatch(body); m != nil {
			out["title"] = words(m[1])
		}
		if m := alertRE.FindStringSubmatch(body); m != nil {
			out["text"] = words(m[1])
		}
	} else if t := strings.TrimSpace(body); t != "" {
		out["text"] = t
	}
	if code < 400 {
		code = http.StatusOK
	}
	writeJSON(w, code, out)
}
