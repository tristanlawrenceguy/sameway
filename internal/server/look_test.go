package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// An agent reads a page the way a screen reader gets it, does what a
// person does and reads where they land, and reads one component from
// props before adding it, all without a browser.
func TestAnAgentLooksAtAPageWithoutABrowser(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Every Sunday."})
	wantStatus(t, created, http.StatusCreated)
	var note struct{ ID string }
	decode(t, created, &note)

	var seen struct {
		Path    string
		Status  int
		Landed  string
		Outline look.Outline
	}
	rec := get(t, h, "/api/look?path=/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)
	if seen.Status != 200 || len(seen.Outline.Headings) == 0 || seen.Outline.Headings[0].Level != 1 {
		t.Errorf("the note page should read with its h1, got %+v", seen)
	}
	if len(seen.Outline.Problems) != 0 {
		t.Errorf("the note page should have no structural problems, got %v", seen.Outline.Problems)
	}
	var crumbs bool
	for _, l := range seen.Outline.Landmarks {
		crumbs = crumbs || (l.Role == "navigation" && l.Label == "Breadcrumb")
	}
	if !crumbs {
		t.Errorf("the crumbs are a labelled navigation landmark, got %v", seen.Outline.Landmarks)
	}

	// Doing what a person does: an over-long title is refused on the page,
	// and a good one lands them back on the note.
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"path": "/t/note/" + note.ID + "/props", "form": map[string]string{"prop-title": strings.Repeat("x", 250)},
	})
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)
	refused := false
	for _, l := range seen.Outline.Live {
		refused = refused || (l.Politeness == "assertive" && strings.Contains(l.Text, "Not saved"))
	}
	if !strings.HasPrefix(seen.Landed, "/t/note/"+note.ID) || !refused {
		t.Errorf("an invalid edit is refused on the note, said as an alert, got landed %q live %v", seen.Landed, seen.Outline.Live)
	}
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"path": "/t/note/" + note.ID + "/props", "method": "POST", "form": map[string]string{"prop-title": "Water the garden"},
	})
	decode(t, rec, &seen)
	if !strings.HasPrefix(seen.Landed, "/t/note/"+note.ID) || seen.Outline.Headings[0].Text != "Water the garden" {
		t.Errorf("a good edit lands back on the note with the new title, got landed %q headings %v", seen.Landed, seen.Outline.Headings)
	}

	// One component from props, and a component that does not exist.
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"component": "card", "props": map[string]any{"title": "Plan"}})
	wantStatus(t, rec, http.StatusOK)
	var frag struct {
		Component string
		HTML      string
		Outline   look.Outline
	}
	decode(t, rec, &frag)
	if !strings.Contains(frag.HTML, `data-component="card"`) || len(frag.Outline.Problems) != 0 || strings.Join(frag.Outline.Components, ",") != "card" {
		t.Errorf("a card from props should read clean, got %+v", frag)
	}
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"component": "nope"})
	wantStatus(t, rec, http.StatusNotFound)
	if !strings.Contains(rec.Body.String(), "card") {
		t.Error("an unknown component should be answered with the ones that exist")
	}

	// A path that is not a page here is refused in the API's own words.
	wantStatus(t, get(t, h, "/api/look?path=http://elsewhere"), http.StatusBadRequest)
}

// TestLookAtComponentRawSettingChangeTransformsAction checks that when
// raw setting-change props (action "set", component "ui.text") are sent
// via POST /api/look, the rendered HTML shows the transformed action
// "changed" rather than the raw word "set". This covers acceptance item 1.
func TestLookAtComponentRawSettingChangeTransformsAction(t *testing.T) {
	_, h := newApp(t)

	rec := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"component": "message",
		"props": map[string]any{
			"role":    "assistant",
			"content": "Text size is now set to large.",
			"changes": []any{
				map[string]any{
					"action":    "set",
					"component": "ui.text",
					"detail":    "large",
				},
			},
		},
	})
	wantStatus(t, rec, http.StatusOK)

	var frag struct {
		HTML string `json:"html"`
	}
	decode(t, rec, &frag)

	if !strings.Contains(frag.HTML, "changed") {
		t.Errorf("raw setting-change action 'set' should be transformed to 'changed'\nwant: 'changed' in HTML\ngot:\n%s", frag.HTML)
	}
	if strings.Contains(frag.HTML, `class="sw-message__change-action">set`) {
		t.Errorf("raw word 'set' must not appear as the change action\nwant: 'changed'\ngot:\n%s", frag.HTML)
	}
}

// TestLookAtComponentRawSettingChangeShowsReadableTarget checks that a raw
// setting-change (component "ui.text") is rendered with a human-readable label
// such as "(text size)" instead of the internal key "(ui.text)". This covers
// acceptance item 2.
func TestLookAtComponentRawSettingChangeShowsReadableTarget(t *testing.T) {
	_, h := newApp(t)

	rec := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"component": "message",
		"props": map[string]any{
			"role":    "assistant",
			"content": "Text size is now set to large.",
			"changes": []any{
				map[string]any{
					"action":    "set",
					"component": "ui.text",
					"detail":    "large",
				},
			},
		},
	})
	wantStatus(t, rec, http.StatusOK)

	var frag struct {
		HTML string `json:"html"`
	}
	decode(t, rec, &frag)

	if strings.Contains(frag.HTML, "(ui.text)") || strings.Contains(frag.HTML, "ui.text") {
		t.Errorf("raw field path 'ui.text' must not appear in rendered changes list\nwant: human-readable label like 'text size'\ngot:\n%s", frag.HTML)
	}

	if !strings.Contains(frag.HTML, "text size to Large") && !strings.Contains(frag.HTML, "Text size to Large") {
		t.Errorf("setting key should be rendered as a human-readable phrase\nwant: 'text size to Large' or similar\ngot:\n%s", frag.HTML)
	}
}

// TestLookAtComponentRawSettingChangeReadableUndoName checks that the Undo
// button's accessible name for a raw setting-change uses plain words, not
// internal keys. For example "changed text size to Large" rather than
// "set ui.text large". This covers acceptance item 3.
func TestLookAtComponentRawSettingChangeReadableUndoName(t *testing.T) {
	_, h := newApp(t)

	rec := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"component": "message",
		"props": map[string]any{
			"role":    "assistant",
			"content": "Pace is set to calm.",
			"changes": []any{
				map[string]any{
					"action":    "set",
					"component": "ui.pace",
					"detail":    "calm",
					"activity":  "act-abc123",
				},
			},
		},
	})
	wantStatus(t, rec, http.StatusOK)

	var frag struct {
		HTML string `json:"html"`
	}
	decode(t, rec, &frag)

	// The accessible name lives in the visually-hidden span inside the Undo button.
	if strings.Contains(frag.HTML, "set ui.pace") || strings.Contains(frag.HTML, "ui.pace calm") {
		t.Errorf("undo button accessible name must not contain raw field paths\nwant: 'changed pace to Calm'\ngot:\n%s", frag.HTML)
	}

	if !strings.Contains(frag.HTML, "Undo") {
		t.Error("message with activity id should render an Undo button: " + frag.HTML)
	}
}

// TestLookAtComponentNonSettingChangePassesThrough checks that non-setting
// changes (e.g. action "added", component "card") are not transformed and pass
// through unchanged. This ensures we do not break non-setting change rendering.
func TestLookAtComponentNonSettingChangePassesThrough(t *testing.T) {
	_, h := newApp(t)

	rec := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"component": "message",
		"props": map[string]any{
			"role":    "assistant",
			"content": "I added a list.",
			"changes": []any{
				map[string]any{
					"action":    "added",
					"component": "list",
					"detail":    "To pack",
				},
			},
		},
	})
	wantStatus(t, rec, http.StatusOK)

	var frag struct {
		HTML string `json:"html"`
	}
	decode(t, rec, &frag)

	if !strings.Contains(frag.HTML, "added") {
		t.Errorf("non-setting action should still show as 'added'\nwant: 'added' in output\ngot:\n%s", frag.HTML)
	}
}

// TestLookAtComponentMixedChangesTransformsOnlySetting checks that when a
// message has both setting-change and non-setting changes, only the setting
// changes are transformed while non-setting ones pass through unchanged.
func TestLookAtComponentMixedChangesTransformsOnlySetting(t *testing.T) {
	_, h := newApp(t)

	rec := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"component": "message",
		"props": map[string]any{
			"role":    "assistant",
			"content": "Done. Pace is set to calm.",
			"changes": []any{
				map[string]any{
					"action":    "set",
					"component": "ui.pace",
					"detail":    "calm",
				},
				map[string]any{
					"action":    "added",
					"component": "card",
					"detail":    "Plan",
				},
			},
		},
	})
	wantStatus(t, rec, http.StatusOK)

	var frag struct {
		HTML string `json:"html"`
	}
	decode(t, rec, &frag)

	if !strings.Contains(frag.HTML, "changed") {
		t.Errorf("setting change should be transformed to 'changed'\ngot:\n%s", frag.HTML)
	}
	if strings.Contains(frag.HTML, `class="sw-message__change-action">set`) {
		t.Errorf("raw word 'set' must not appear\ngot:\n%s", frag.HTML)
	}
	if !strings.Contains(frag.HTML, "added") && !strings.Contains(frag.HTML, `"action":"added"`) {
		t.Errorf("non-setting action should still show as 'added'\ngot:\n%s", frag.HTML)
	}
}
