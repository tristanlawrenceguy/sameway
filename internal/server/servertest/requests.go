package servertest

import (
	"bytes"
	"encoding/json"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// Do performs a request and returns the recorder.
func Do(t *testing.T, h http.Handler, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Get reads a page.
func Get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	return Do(t, h, http.MethodGet, path, nil, "")
}

// PostForm posts a form, as a page's form does.
func PostForm(t *testing.T, h http.Handler, path string, values url.Values) *httptest.ResponseRecorder {
	return Do(t, h, http.MethodPost, path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
}

// PostJSON sends v as JSON, as an agent does.
func PostJSON(t *testing.T, h http.Handler, method, path string, v any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(v)
	return Do(t, h, method, path, strings.NewReader(string(raw)), "application/json")
}

// As makes a request as someone on another device, the way the tailnet
// marks one it has let in.
func As(t *testing.T, h http.Handler, v records.Visitor, method, path string, body string, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req = req.WithContext(records.WithVisitor(req.Context(), v))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Landed follows an action's redirect the way a browser does, carrying
// the outcome cookie, and returns the page the person is back on and
// where it is.
func Landed(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) (page *httptest.ResponseRecorder, at string) {
	t.Helper()
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("an action returns the person to a page (303), got %d: %.300s", rec.Code, rec.Body.String())
	}
	at = rec.Header().Get("Location")
	// A browser keeps the part after # to itself.
	path, _, _ := strings.Cut(at, "#")
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	page = httptest.NewRecorder()
	h.ServeHTTP(page, req)
	return page, at
}

// After is the page an action returns the person to, as they see it.
func After(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	page, _ := Landed(t, h, rec)
	return page
}

// MultipartFile builds a form with one file part and any other fields.
func MultipartFile(t *testing.T, name, content string, fields url.Values) (io.Reader, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, vs := range fields {
		for _, v := range vs {
			w.WriteField(k, v)
		}
	}
	if name != "" {
		part, err := w.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(part, content)
	}
	w.Close()
	return &body, w.FormDataContentType()
}

// Parse reads a page's HTML.
func Parse(t *testing.T, rec *httptest.ResponseRecorder) *htmltest.Doc {
	t.Helper()
	doc, err := htmltest.Parse(rec.Body.String())
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// Decode reads a JSON answer into v.
func Decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("bad JSON (%d): %s", rec.Code, rec.Body.String())
	}
}

// WantStatus fails unless the answer has the status.
func WantStatus(t *testing.T, rec *httptest.ResponseRecorder, code int) {
	t.Helper()
	if rec.Code != code {
		t.Fatalf("expected %d, got %d: %s", code, rec.Code, Truncate(rec.Body.String()))
	}
}

// Truncate is the start of a long answer, for a failure to show.
func Truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

// WaitFor waits for something done in the background, or fails.
func WaitFor(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; i < 300; i++ {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("it did not happen in time")
}

var (
	scriptOrStyle = regexp.MustCompile(`(?is)<(script|style)\b.*?</(script|style)>`)
	anyTag        = regexp.MustCompile(`(?s)<[^>]*>`)
	spaces        = regexp.MustCompile(`\s+`)
)

// Said is the text a page says, as a person reads it: no tags, entities
// read as their characters, and whitespace collapsed to single spaces.
func Said(page string) string {
	page = scriptOrStyle.ReplaceAllString(page, " ")
	page = anyTag.ReplaceAllString(page, "")
	page = html.UnescapeString(page)
	return strings.TrimSpace(spaces.ReplaceAllString(page, " "))
}
