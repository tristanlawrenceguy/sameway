package server

import (
	"bytes"
	"net/http"
)

// notFoundHandler wraps the mux and renders an HTML 404 page when no route matches.
type notFoundHandler struct {
	mux *http.ServeMux
	s   *Server
}

func (nh *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		nh.mux.ServeHTTP(w, r)
		return
	}
	rw := &catchWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
	nh.mux.ServeHTTP(rw, r)
	if rw.status == http.StatusNotFound && bytes.HasPrefix(rw.buf.Bytes(), []byte("404 page not found")) {
		nh.s.notFoundPage(w, r)
	} else if rw.status != 0 {
		w.WriteHeader(rw.status)
		rw.buf.WriteTo(w)
	}
}

// catchWriter is a passthrough ResponseWriter that buffers the body for 404 responses.
type catchWriter struct {
	http.ResponseWriter
	status int
	buf    *bytes.Buffer
}

func (cw *catchWriter) WriteHeader(code int) {
	cw.status = code
}

func (cw *catchWriter) Write(b []byte) (int, error) {
	if cw.status == 0 {
		cw.status = http.StatusOK
	}
	cw.buf.Write(b)
	return len(b), nil
}

func (s *Server) wrapNotFound(m *http.ServeMux) *http.ServeMux {
	nh := &notFoundHandler{mux: m, s: s}
	out := http.NewServeMux()
	out.Handle("/", nh)
	return out
}
