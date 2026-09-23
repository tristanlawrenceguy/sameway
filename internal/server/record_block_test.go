package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// A record block is the record itself on the canvas, not a copy: it shows
// the record's words, it edits the record through the record's own props
// endpoint, and the person lands back on the canvas afterwards. When the
// record goes, the block says so instead of vanishing or breaking.
func TestARecordBlockIsTheRecordOnTheCanvas(t *testing.T) {
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the dentist", "body": "Ask about Thursday.", "tags": []string{"health"}}), &note)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "record", "props": map[string]any{"type": "note", "record": note.ID}, "span": 6,
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()
	for _, want := range []string{
		`data-component="record"`, `data-record-id="` + note.ID + `"`,
		`data-prop="title">Call the dentist</h2>`, `data-prop="body" data-source="Ask about Thursday." data-prose-level="3"><p>Ask about Thursday.</p>`,
		`<dt>Tags</dt><dd>health</dd>`,
		`data-edit-action="/t/note/` + note.ID + `/props"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the canvas should show the record with %s", want)
		}
	}

	// Structured text keeps its structure on the canvas, one level under
	// the block's title, and is marked for the editor like the page is.
	var plan struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Garden plan", "body": "# Beds\n\n- Dig the pond"}), &plan)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "record", "props": map[string]any{"type": "note", "record": plan.ID}}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "<h3>Beds</h3>") || !strings.Contains(page, "<li>Dig the pond</li>") || !strings.Contains(page, `data-source="# Beds`) {
		t.Errorf("a note's structure should show on the canvas: %.600s", page[strings.Index(page, "Garden plan"):])
	}

	// Editing from the canvas changes the note and comes back to the canvas.
	edit := doWithReferer(t, h, "/t/note/"+note.ID+"/props", url.Values{"prop-title": {"Call the dentist at nine"}}, "http://example.com/")
	wantStatus(t, edit, http.StatusSeeOther)
	if loc := edit.Header().Get("Location"); !strings.HasPrefix(loc, "/") {
		t.Errorf("an edit made on the canvas should return to the canvas, got %q", loc)
	}
	if body := get(t, h, "/t/note/"+note.ID).Body.String(); !strings.Contains(body, "Call the dentist at nine") {
		t.Error("the note on its own page should show the edit made on the canvas")
	}
	if body := get(t, h, "/").Body.String(); !strings.Contains(body, "Call the dentist at nine") {
		t.Error("the canvas should show the edit too: it is the same note")
	}

	// Expanded, the block is the record at page size, named after it.
	var blocks struct{ Records []struct{ ID string } }
	decode(t, get(t, h, "/api/block"), &blocks)
	focus := get(t, h, "/canvas/"+blocks.Records[len(blocks.Records)-1].ID).Body.String()
	if !strings.Contains(focus, "Open this note on its page") || !strings.Contains(focus, "<title>Call the dentist at nine") {
		t.Error("the expanded block should be the note at page size, under its own name")
	}

	// A note that is gone is said to be gone.
	wantStatus(t, do(t, h, http.MethodDelete, "/api/note/"+note.ID, nil, ""), http.StatusOK)
	rec := get(t, h, "/")
	wantStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "This note is no longer here.") {
		t.Error("a record block whose record is gone should say so")
	}
}

// doWithReferer posts a form the way a browser does from a page, so the
// server can send the person back to it.
func doWithReferer(t *testing.T, h http.Handler, path string, values url.Values, referer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", referer)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
