package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// apiError is the JSON error shape. code is stable, message is for people,
// fields lists per-field validation problems when there are any.
type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var ve *schema.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: "some fields are invalid", Fields: ve.Problems}})
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found", Message: err.Error()}})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": apiError{Code: "bad_request", Message: err.Error()}})
	}
}

func readBody(r *http.Request) (map[string]any, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, errors.New("body must be a JSON object of fields: " + jsonTrouble(err))
	}
	return fields, nil
}

// apiDescribe is the index: how to build, the routes, and a line for each
// component and type, a few kilobytes. ?full=1 is everything, which is
// hundreds of kilobytes and was read by an agent routes last.
func (s *Server) apiDescribe(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("full") != "" {
		writeJSON(w, http.StatusOK, s.app.Describe())
		return
	}
	writeJSON(w, http.StatusOK, s.app.Describe().Index())
}

func (s *Server) apiList(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			msg := "invalid limit parameter"
			if err != nil {
				msg = fmt.Sprintf("invalid limit parameter: %v", err)
			}
			writeError(w, errors.New(msg))
			return
		}
	}
	// A page at a time and only some fields, when asked; see api_read.go.
	size, page, err := listPage(r.URL.Query(), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	if page > 0 {
		limit = 0
	}
	only, err := s.onlyFields(r, r.PathValue("type"))
	if err != nil {
		writeError(w, err)
		return
	}
	// ?where= (repeatable) and ?order= take the same query a collection
	// block does: field=value, due<today, -due; see the collection component.
	// The list page's own query, one way for every surface: query.Filter.
	// ?dir=desc is the older way to say -order.
	order := r.URL.Query().Get("order")
	if r.URL.Query().Get("dir") == "desc" && order != "" && !strings.HasPrefix(order, "-") {
		order = "-" + order
	}
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		writeError(w, fmt.Errorf("no content type %q", r.PathValue("type")))
		return
	}
	recs, err := query.Filter(s.app.Store, t, r.URL.Query()["where"], order, limit, time.Now())
	if err != nil {
		writeError(w, err)
		return
	}
	// Each record as it always was, with its title and who wrote it beside it.
	type written struct {
		*store.Record
		Title     string `json:"title"`
		WrittenBy string `json:"written_by"`
	}
	var about map[string]any
	if page > 0 {
		recs, about = onePage(r, recs, size, page)
	}
	writers := s.app.Chat.Writers()
	out := make([]written, 0, len(recs))
	for _, rec := range recs {
		out = append(out, written{trimmed(rec, only), s.apiTitle(rec), writers.Of(rec.Type, rec).Words})
	}
	answer := map[string]any{"type": r.PathValue("type"), "count": len(recs), "records": out, "untrusted": "each record's title and fields were written by its written_by: " + chat.Untrusted}
	for k, v := range about {
		answer[k] = v
	}
	writeJSON(w, http.StatusOK, answer)
}

// apiDescribePart serves one section of the description, or one item in it,
// cut by the same Part every other surface uses.
func (s *Server) apiDescribePart(w http.ResponseWriter, r *http.Request) {
	part := s.app.Describe().Part
	if r.URL.Query().Get("full") != "" {
		part = s.app.Describe().FullPart
	}
	v, err := part(r.PathValue("part"), r.PathValue("name"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found", Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// apiFileUpload accepts a base64-encoded file from an agent. The JSON body
// has optional title, optional filename, and optional content (base64 string).
// Without content it creates a stub record; with content it decodes, writes to
// disk, reads text for built-in formats, and returns 201 with the record.
func (s *Server) apiFileUpload(w http.ResponseWriter, r *http.Request) {
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}

	title, _ := fields["title"].(string)
	filename, _ := fields["filename"].(string)
	contentB64, hasContent := fields["content"]

	// If content is provided it must be valid base64.
	if hasContent {
		raw, ok := contentB64.(string)
		if !ok || raw == "" {
			writeError(w, errors.New("content must be a non-empty base64 string"))
			return
		}
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			writeError(w, fmt.Errorf("invalid base64 content: %w", err))
			return
		}
		if len(data) > maxUpload {
			writeError(w, errors.New("the file is too large to send inside JSON: 64 MB is the most; send it as a form (multipart, up to 4 GB) instead"))
			return
		}

		name := filename
		if name == "" {
			name = "upload"
		} else {
			name = filepath.Base(name)
		}
		if title == "" {
			title = strings.TrimSuffix(name, filepath.Ext(name))
		}
		rec, path, err := s.keepFile(apiAgent(r).As(), bytes.NewReader(data), name, title, "")
		if err != nil {
			writeError(w, err)
			return
		}
		// wait: true answers once the text is read, even by a converter
		// that takes a while; without it status says converting until then.
		wait, _ := fields["wait"].(bool)
		s.readKept(rec.ID, name, path, wait)
		rec, _ = s.app.Store.Get(FileType, rec.ID)

		w.Header().Set("Location", "/api/"+FileType+"/"+rec.ID)
		writeJSON(w, http.StatusCreated, s.titled(rec))
		return
	}

	// No content: if a filename is present create a stub record; otherwise
	// the request needs either content or at least a name to be useful.
	if filename == "" {
		writeError(w, errors.New("upload needs either content (base64) or a filename"))
		return
	}
	name := filepath.Base(filename)
	if title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	rec, _, err := chat.WriteKept(s.app.Store, apiAgent(r).As(), "created", FileType, "", map[string]any{
		"title": title, "name": name, "kind": convert.Kind(name), "size": 0, "status": "ready",
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+FileType+"/"+rec.ID)
	writeJSON(w, http.StatusCreated, s.titled(rec))
}

// apiChatClear clears all messages from the current chat session and returns
// confirmation JSON. The canvas, other chats, and blocks are left alone.
func (s *Server) apiChatClear(w http.ResponseWriter, r *http.Request) {
	if err := s.chatFor(r).Clear(); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "cleared"})
}

// apiNotFound answers any path under /api that nothing serves in the shape
// every other error there has, so an agent that typed a route wrong reads
// JSON like always rather than a plain-text page.
func (s *Server) apiNotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{
		Code: "not_found", Message: r.Method + " " + r.URL.Path + " is not a route; GET /api/describe/routes lists them",
	}})
}
