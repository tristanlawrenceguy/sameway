package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
)

// lookFor is the look the assistant takes for the person: the same one an
// agent gets from /api/look, with the page's scripts run. On a machine
// with no browser it reads the page as served, which is most of it,
// unless the person's own steps are what it has to see.
func (s *Server) lookFor(ctx context.Context, ask map[string]any) (string, error) {
	steps, _ := ask["steps"].([]any)
	ask["scripts"] = true
	code, body := s.lookJSON(ctx, ask)
	if code == http.StatusNotImplemented && len(steps) == 0 {
		delete(ask, "scripts")
		code, body = s.lookJSON(ctx, ask)
	}
	if code != http.StatusOK {
		var e struct {
			Error apiError `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
			return "", errors.New(e.Error.Message)
		}
		return "", errors.New(string(body))
	}
	return string(bytes.TrimSpace(body)), nil
}

func (s *Server) lookJSON(ctx context.Context, ask map[string]any) (int, []byte) {
	raw, _ := json.Marshal(ask)
	req := httptest.NewRequest(http.MethodPost, "/api/look", bytes.NewReader(raw)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.apiLook(rec, req)
	return rec.Code, rec.Body.Bytes()
}
