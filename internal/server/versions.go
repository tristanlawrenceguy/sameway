package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Saving from a copy that has gone out of date (see chat/versions.go), for
// the two who save: a person's page and an agent's request.

// versionAttrs is what a record's page says it showed: the version, and
// a fingerprint of each field. The editor sends both back with a save
// (08-edit.js).
func versionAttrs(rec *store.Record) string {
	prints, _ := json.Marshal(chat.Prints(rec.Fields))
	return ` data-version="` + chat.Version(rec) + `" data-was="` + template.HTMLEscapeString(string(prints)) + `"`
}

// sinceOpened reconciles a save from a page with what changed since the
// page was opened. The editor sends every field it shows, as the page
// showed them: a field the person did not touch that someone else has
// changed since keeps that change, rather than being put back as it was.
// A field both changed is the person's, the last word, and crossed names
// it so they are told; Undo puts the other back. clean is changed in place.
func sinceOpened(t *schema.Type, rec *store.Record, fields, clean map[string]any, form map[string][]string) (crossed []string) {
	get := func(k string) string {
		if v := form[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	if version := get("version"); version == "" || chat.SameVersion(rec, version) {
		return nil
	}
	var was map[string]string
	if json.Unmarshal([]byte(get("was")), &was) != nil {
		return nil
	}
	for name := range fields {
		then, known := was[name]
		now := chat.Print(rec.Fields[name])
		if !known || now == then {
			continue // nobody else changed it
		}
		if chat.Print(clean[name]) == then {
			clean[name] = rec.Fields[name] // the person left it; the other change stays
			continue
		}
		if f, ok := t.Field(name); ok {
			crossed = append(crossed, fieldLabel(*f))
		}
	}
	return crossed
}

// staleFor refuses an agent's change made from an older version than the
// record's, when it says which one it read with If-Match, and answers
// with the record as it is, to change again.
func (s *Server) staleFor(w http.ResponseWriter, r *http.Request, rec *store.Record) bool {
	want := strings.TrimSpace(r.Header.Get("If-Match"))
	if want == "" || want == "*" || chat.SameVersion(rec, want) {
		return false
	}
	writeJSON(w, http.StatusPreconditionFailed, map[string]any{
		"error":   apiError{Code: "stale", Message: "it has changed since the version you read (If-Match " + want + "): nothing was written. current is the record as it is now; make your change to it and send it again with If-Match set to its updated_at"},
		"current": s.titled(rec),
	})
	return true
}
