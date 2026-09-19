package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
		return nil, errors.New("body must be a JSON object of fields: " + err.Error())
	}
	return fields, nil
}

func (s *Server) apiDescribe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Describe())
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
	// ?where= (repeatable) and ?order= take the same query a collection
	// block does: field=value, due<today, -due; see the collection component.
	var recs []*store.Record
	var err error
	if where := r.URL.Query()["where"]; len(where) > 0 || strings.HasPrefix(r.URL.Query().Get("order"), "-") {
		t, ok := s.app.Types.Get(r.PathValue("type"))
		if !ok {
			writeError(w, fmt.Errorf("no content type %q", r.PathValue("type")))
			return
		}
		recs, err = query.Filter(s.app.Store, t, where, r.URL.Query().Get("order"), limit, time.Now())
	} else {
		recs, err = s.app.Store.List(r.PathValue("type"), store.ListOptions{
			OrderBy: r.URL.Query().Get("order"),
			Desc:    r.URL.Query().Get("dir") == "desc",
			Limit:   limit,
		})
	}
	if err != nil {
		writeError(w, err)
		return
	}
	if recs == nil {
		recs = []*store.Record{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": r.PathValue("type"), "count": len(recs), "records": recs})
}

func (s *Server) apiGet(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) apiCreate(w http.ResponseWriter, r *http.Request) {
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.app.Store.Create(r.PathValue("type"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+rec.Type+"/"+rec.ID)
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) apiUpdate(w http.ResponseWriter, r *http.Request) {
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.app.Store.Update(r.PathValue("type"), r.PathValue("id"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Store.Delete(r.PathValue("type"), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": r.PathValue("id")})
}

// apiChat lets an agent talk to the assistant the same way a person does.
func (s *Server) apiChat(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	text, _ := body["message"].(string)
	// canvas is the tab to build on: a canvas id, or absent for Home.
	canvas, _ := body["canvas"].(string)
	rec, err := s.app.Chat.SendOn(r.Context(), canvas, text)
	if rec == nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reply": rec, "ok": err == nil})
}

// apiDescribePart serves one section of the description, or one item in it,
// cut by the same Part every other surface uses.
func (s *Server) apiDescribePart(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.Describe().Part(r.PathValue("part"), r.PathValue("name"))
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
			writeError(w, errors.New("the file is too large: 64 MB is the most one can be"))
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
		rec, err := s.app.Store.Create(FileType, map[string]any{
			"title": title, "name": name, "kind": convert.Kind(name), "size": len(data), "status": "converting",
		})
		if err != nil {
			writeError(w, err)
			return
		}
		stored := rec.ID + strings.ToLower(filepath.Ext(name))
		dir := s.app.Workspace.FilesDir()
		if err := os.MkdirAll(dir, 0o755); err == nil {
			err = os.WriteFile(filepath.Join(dir, stored), data, 0o644)
		}
		if err != nil {
			s.app.Store.Delete(FileType, rec.ID)
			writeError(w, fmt.Errorf("could not keep the file: %w", err))
			return
		}
		s.app.Store.Update(FileType, rec.ID, map[string]any{"path": stored})
		chat.Record(s.app.Store, "human", chat.Change{Action: "added", Component: FileType, ID: rec.ID, Detail: title, Href: "/t/" + FileType + "/" + rec.ID})

		if converter := s.app.Workspace.Config.Files.Convert[convert.Ext(name)]; converter != "" {
			go func() { s.convertLater(rec.ID, converter, name, filepath.Join(dir, stored)) }()
		} else {
			s.readNow(rec.ID, name, data)
		}

		w.Header().Set("Location", "/api/"+FileType+"/"+rec.ID)
		writeJSON(w, http.StatusCreated, rec)
		return
	}

	// No content: if a filename is present create a stub record; otherwise
	// the request needs either content or at least a name to be useful.
	if filename == "" {
		writeError(w, errors.New("upload needs either content (base64) or a filename"))
		return
	}
	name := filepath.Base(filename)
	rec, err := s.app.Store.Create(FileType, map[string]any{
		"title": title, "name": name, "kind": convert.Kind(name), "size": 0, "status": "ready",
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+FileType+"/"+rec.ID)
	writeJSON(w, http.StatusCreated, rec)
}

// apiChatClear clears all messages from the current chat session and returns
// confirmation JSON. The canvas, other chats, and blocks are left alone.
func (s *Server) apiChatClear(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Chat.Clear(); err != nil {
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
