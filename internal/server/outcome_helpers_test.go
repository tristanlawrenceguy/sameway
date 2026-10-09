package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
