package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// UndoPageStripsThroughTheAPISuffix checks that the undo success page does
// not show ", through the API" even for old entries stored with that suffix
// in their summary (acceptance 1–4). The undo handler should strip it.
func TestUndoPageStripsThroughTheAPISuffix(t *testing.T) {
	a, h := newApp(t)

	// Seed an old-style activity record with the "through the API" suffix
	// in its stored summary — this is what older records have.
	oldID := "old-undo-api"
	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary":   "You deleted note Water the plants, through the API.",
		"actor":     "human",
		"action":    "deleted",
		"target":    "note",
		"detail":    "Water the plants",
		"via":       "through the API",
		"target_id": oldID,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Post to the undo endpoint. The actual undo will fail (the thing is gone),
	// but we still get an outcome page with what was undone — read it via
	// the redirect location rather than the empty post body.
	rec := do(t, h, http.MethodPost, "/activity/"+oldID+"/undo", strings.NewReader(url.Values{"from": {"/activity"}}.Encode()), "application/x-www-form-urlencoded")

	// The undo page is served as a redirect (SeeOther) to "/", so fetch the
	// final destination and check its body for the success text.
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("undo should return a Location header")
	}

	body := get(t, h, location).Body.String()

	if strings.Contains(body, "through the API") {
		t.Errorf("success page must not contain 'through the API'\n%s", truncate(body))
	}
	if strings.Contains(body, ", through the API") {
		t.Errorf("success page must not contain ', through the API'\n%s", truncate(body))
	}

	// The undo page should say what was undone in plain words.
	s := said(body)
	if !strings.Contains(s, "deleted note Water the plants") &&
		!strings.Contains(s, "You deleted note Water the plants") {
		t.Errorf("undo success page should mention the action in plain words\n%q", s)
	}
}

// UndoPageStripsThroughCLISuffix checks that CLI suffixes are also stripped.
func TestUndoPageStripsThroughCLISuffix(t *testing.T) {
	a, h := newApp(t)

	oldID := "old-undo-cli"
	_, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary":   "You deleted note Seed, through the command line.",
		"actor":     "human",
		"action":    "deleted",
		"target":    "note",
		"detail":    "Seed",
		"via":       "through the command line",
		"target_id": oldID,
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := do(t, h, http.MethodPost, "/activity/"+oldID+"/undo", strings.NewReader(url.Values{"from": {"/activity"}}.Encode()), "application/x-www-form-urlencoded")

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("undo should return a Location header")
	}

	body := get(t, h, location).Body.String()

	if strings.Contains(body, "through the command line") {
		t.Errorf("success page must not contain 'through the command line'\n%s", truncate(body))
	}
}
