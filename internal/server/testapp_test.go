package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/testkit"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// newApp is an app on a fresh copy of the starter workspace (testkit), in
// a temp dir of its own, with no model attached.
func newApp(t *testing.T) (*app.App, http.Handler) {
	t.Helper()
	return newAppWith(t, app.Options{})
}

// newAppWith is newApp opened with options: a fixed clock, say.
func newAppWith(t *testing.T, o app.Options) (*app.App, http.Handler) {
	t.Helper()
	// A machine of its own, so the workspaces this computer has opened, its
	// copies and its pasted keys are never read or written by a test.
	if o.Machine == (workspace.Machine{}) {
		o.Machine = testMachine(t)
	}
	a, err := app.Open(testkit.Starter(t), o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Chat.Provider, a.Chat.ProviderErr = nil, llm.ErrNotConfigured
	return a, server.New(a)
}

// scripted is a model that answers with a fixed sequence of responses.
type scripted struct {
	steps []*llm.Response
	seen  []llm.Request
}

func (s *scripted) Name() string { return "scripted" }

func (s *scripted) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	s.seen = append(s.seen, req)
	if len(s.steps) == 0 {
		return &llm.Response{Text: "ok"}, nil
	}
	next := s.steps[0]
	s.steps = s.steps[1:]
	return next, nil
}

func toolCall(name string, args map[string]any) *llm.Response {
	raw, _ := json.Marshal(args)
	return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "call", Name: name, Args: raw}}}
}

// do performs a request and returns the recorder.
func do(t *testing.T, h http.Handler, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	return do(t, h, http.MethodGet, path, nil, "")
}

func postForm(t *testing.T, h http.Handler, path string, values url.Values) *httptest.ResponseRecorder {
	return do(t, h, http.MethodPost, path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
}

func postJSON(t *testing.T, h http.Handler, method, path string, v any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(v)
	return do(t, h, method, path, strings.NewReader(string(raw)), "application/json")
}

func parse(t *testing.T, rec *httptest.ResponseRecorder) *htmltest.Doc {
	t.Helper()
	doc, err := htmltest.Parse(rec.Body.String())
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("bad JSON (%d): %s", rec.Code, rec.Body.String())
	}
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, code int) {
	t.Helper()
	if rec.Code != code {
		t.Fatalf("expected %d, got %d: %s", code, rec.Code, truncate(rec.Body.String()))
	}
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

// assertAllComponentsKnown walks the parsed HTML and fails if any element uses a
// data-component value that is not registered in the app's component registry.
func assertAllComponentsKnown(t *testing.T, doc *htmltest.Doc, names []string) {
	t.Helper()
	known := make(map[string]bool, len(names))
	for _, n := range names {
		known[n] = true
	}
	doc.Walk(func(n *html.Node) {
		if c, ok := htmltest.Attr(n, "data-component"); ok && !known[c] {
			t.Errorf("%s: data-component=%q is not a registered component (known: %v)", n.Data, c, names)
		}
	})
}

// testMachine is a computer's folders for one test: an empty known list,
// no keys, no copies (workspace.Machine).
func testMachine(t *testing.T) workspace.Machine {
	dir := t.TempDir()
	m := workspace.Machine{Known: filepath.Join(dir, "known.json"), Keys: filepath.Join(dir, "keys.json")}
	os.WriteFile(m.Known, []byte("[]"), 0o644)
	return m
}
