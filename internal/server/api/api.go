package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// apiDescribe is the index: how to build, the routes, and a line for each
// component and type, a few kilobytes. ?full=1 is everything, which is
// hundreds of kilobytes and was read by an agent routes last.
func (s *Service) apiDescribe(w http.ResponseWriter, r *http.Request) {
	if refusedWords(w, r, "/api/describe", describeWords) { // api_params.go
		return
	}
	if r.URL.Query().Get("full") != "" {
		web.WriteJSON(w, http.StatusOK, s.app.Describe())
		return
	}
	web.WriteJSON(w, http.StatusOK, s.app.Describe().Index())
}

func (s *Service) apiList(w http.ResponseWriter, r *http.Request) {
	if !s.knownType(w, r) { // api_refusals.go
		return
	}
	if why := unknownParams(r.URL.Query(), r.PathValue("type")); why != "" { // api_params.go
		web.WriteError(w, errors.New(why))
		return
	}
	limitStr := r.URL.Query().Get("limit")
	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			web.WriteError(w, fmt.Errorf("limit is how many to answer with, a whole number from 1, not %q", limitStr))
			return
		}
	}
	// A page at a time and only some fields, when asked; see api_read.go.
	size, page, err := listPage(r.URL.Query(), limit)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	if page > 0 {
		limit = 0
	}
	only, err := s.onlyFields(r, r.PathValue("type"))
	if err != nil {
		web.WriteError(w, err)
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
	t, _ := s.app.Types.Get(r.PathValue("type")) // known: above
	recs, err := query.Filter(s.app.Store, t, r.URL.Query()["where"], order, limit, s.Now())
	if err != nil {
		web.WriteError(w, err)
		return
	}
	recs = tabOnly(r, recs) // api_tab_blocks.go
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
	writers := s.app.Records.WritersFor(false)
	out := make([]written, 0, len(recs))
	for _, rec := range recs {
		out = append(out, written{listed(rec, only), s.apiTitle(rec), writers.Of(rec.Type, rec).Words})
	}
	answer := map[string]any{"type": r.PathValue("type"), "count": len(recs), "records": out, "untrusted": "each record's title and fields were written by its written_by: " + records.Untrusted}
	for k, v := range about {
		answer[k] = v
	}
	if t.Name == records.CanvasType {
		// Home is a tab with no record of its own: an agent told the tabs
		// were none could not tell where to put a block.
		answer["home"] = map[string]any{"id": "", "name": "Home", "page": "/", "said": `Home is always there, as canvas "" (or left out); the records are the tabs beside it`}
	}
	web.WriteJSON(w, http.StatusOK, answer)
}

// apiDescribePart serves one section of the description, or one item in it,
// cut by the same Part every other surface uses.
func (s *Service) apiDescribePart(w http.ResponseWriter, r *http.Request) {
	if refusedWords(w, r, r.URL.Path, describeWords) {
		return
	}
	part := s.app.Describe().Part
	if r.URL.Query().Get("full") != "" {
		part = s.app.Describe().FullPart
	}
	v, err := part(r.PathValue("part"), r.PathValue("name"))
	if err != nil {
		web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": web.APIError{Code: "not_found", Message: err.Error()}})
		return
	}
	web.WriteJSON(w, http.StatusOK, v)
}

// apiFileUpload accepts a base64-encoded file from an agent. The JSON body
// has optional title, optional filename, and optional content (base64 string).
// Without content it creates a stub record; with content it decodes, writes to
// disk, reads text for built-in formats, and returns 201 with the record.
func (s *Service) apiFileUpload(w http.ResponseWriter, r *http.Request) {
	fields, err := web.ReadBody(r)
	if err != nil {
		web.WriteError(w, err)
		return
	}

	title, _ := fields["title"].(string)
	filename, _ := fields["filename"].(string)
	contentB64, hasContent := fields["content"]

	// If content is provided it must be valid base64.
	if hasContent {
		raw, ok := contentB64.(string)
		if !ok || raw == "" {
			web.WriteError(w, errors.New("content must be a non-empty base64 string"))
			return
		}
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			web.WriteError(w, fmt.Errorf("invalid base64 content: %w", err))
			return
		}
		if len(data) > media.MaxUpload {
			web.WriteError(w, errors.New("the file is too large to send inside JSON: 64 MB is the most; send it as a form (multipart, up to 4 GB) instead"))
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
		rec, path, err := s.Media().KeepFile(Agent(r).As(), bytes.NewReader(data), name, title, "")
		if err != nil {
			web.WriteError(w, err)
			return
		}
		// wait: true answers once the text is read, even by a converter
		// that takes a while; without it status says converting until then.
		wait, _ := fields["wait"].(bool)
		s.Media().ReadKept(rec.ID, name, path, wait)
		rec, _ = s.app.Store.Get(records.FileType, rec.ID)

		w.Header().Set("Location", "/api/"+records.FileType+"/"+rec.ID)
		web.WriteJSON(w, http.StatusCreated, s.titled(rec))
		return
	}

	// No content: if a filename is present create a stub record; otherwise
	// the request needs either content or at least a name to be useful.
	if filename == "" {
		web.WriteError(w, errors.New("upload needs either content (base64) or a filename"))
		return
	}
	name := filepath.Base(filename)
	if title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	rec, _, err := records.WriteKept(s.app.Store, Agent(r).As(), "created", records.FileType, "", map[string]any{
		"title": title, "name": name, "kind": convert.Kind(name), "size": 0, "status": "ready",
	})
	if err != nil {
		web.WriteError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+records.FileType+"/"+rec.ID)
	web.WriteJSON(w, http.StatusCreated, s.titled(rec))
}

// apiChatClear clears all messages from the current chat session and returns
// confirmation JSON. The canvas, other chats, and blocks are left alone.
func (s *Service) apiChatClear(w http.ResponseWriter, r *http.Request) {
	if err := s.ChatFor(r).Clear(); err != nil {
		web.WriteError(w, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"status": "cleared"})
}

// apiNotFound answers any path under /api that nothing serves in the shape
// every other error there has, so an agent that typed a route wrong reads
// JSON like always rather than a plain-text page.
func (s *Service) apiNotFound(w http.ResponseWriter, r *http.Request) {
	web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": web.APIError{
		Code: "not_found", Message: r.Method + " " + r.URL.Path + " is not a route; GET /api/describe/routes lists them", Describe: "/api/describe/routes",
	}})
}

// listed is a record as a list over the API gives it: only the fields
// asked for, and an entry in the log with its names cleaned, on a copy.
func listed(rec *store.Record, only []string) *store.Record {
	tr := trimmed(rec, only)
	if rec.Type != records.ActivityType || tr.Fields == nil {
		return tr
	}
	fieldsCopy := make(map[string]any, len(tr.Fields))
	for k, v := range tr.Fields {
		fieldsCopy[k] = v
	}
	cleanActivityFields(fieldsCopy)
	return &store.Record{
		ID:        rec.ID,
		Type:      rec.Type,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
		Fields:    fieldsCopy,
	}
}

// cleanActivityFields resolves raw schema identifiers in type-setting activity
// entries. When target is "type", detail holds the raw identifier (e.g.,
// "test_type") and summary may contain it too ("System added type test_type").
// Both are replaced with their display-form names ("Test Type") so agents
// reading the API list see human-readable values, not internal IDs.
func cleanActivityFields(fields map[string]any) {
	target, _ := fields["target"].(string)
	if target != "type" {
		return // nothing to clean for non-type-setting entries
	}
	detail, _ := fields["detail"].(string)
	summary, _ := fields["summary"].(string)

	cleanedDetail := schema.DisplayName(detail)
	fields["detail"] = cleanedDetail

	if strings.Contains(summary, " "+detail) {
		fields["summary"] = strings.Replace(summary, " "+detail, " "+cleanedDetail, 1)
	} else if strings.HasSuffix(summary, detail) && summary != cleanedDetail {
		fields["summary"] = summary[:len(summary)-len(detail)] + cleanedDetail
	}
}
