package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// landed follows an action's redirect the way a browser does, carrying
// the outcome cookie, and returns the page the person is back on and
// where it is.
func landed(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) (page *httptest.ResponseRecorder, at string) {
	t.Helper()
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("an action returns the person to a page (303), got %d: %.300s", rec.Code, rec.Body.String())
	}
	at = rec.Header().Get("Location")
	req := httptest.NewRequest(http.MethodGet, at, nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	page = httptest.NewRecorder()
	h.ServeHTTP(page, req)
	return page, at
}

// withReferer is a request made from a page, as a browser sends it.
func withReferer(t *testing.T, h http.Handler, method, path, referer, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if referer != "" {
		req.Header.Set("Referer", "http://example.com"+referer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// after is the page an action returns the person to, as they see it.
func after(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	page, _ := landed(t, h, rec)
	return page
}
