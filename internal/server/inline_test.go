package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// The division of labour: the assistant owns design, a person owns content
// and actions. These tests hold that line from both sides.

// TestPeopleEditContentInPlace covers the whole of what a person can change
// about a block: the words it shows. No page to visit, no layout controls.
func TestPeopleEditContentInPlace(t *testing.T) {
	h, id := canvasWithABlock(t)

	rec := postForm(t, h, "/canvas/"+id+"/props", url.Values{"prop-title": {"Groceries"}})
	wantStatus(t, rec, http.StatusSeeOther)
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("an inline edit should leave you where you were, got %q", loc)
	}

	page := parse(t, get(t, h, "/"))
	block := page.WithAttr("data-block-id", id)[0]
	if !strings.Contains(htmltest.Text(block), "Groceries") {
		t.Errorf("the new text should be on the canvas")
	}
	if actor, _ := htmltest.Attr(block, "data-actor"); actor != "human" {
		t.Errorf("an edit by a person should be attributed to them, got %q", actor)
	}
	if changed, _ := htmltest.Attr(block, "data-changed"); changed == "" {
		t.Errorf("the edit should glow, like any other change")
	}

	// Props the form did not mention are left alone.
	var blocks struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	props := blocks.Records[0].Fields["props"].(map[string]any)
	if props["title"] != "Groceries" {
		t.Errorf("props: %v", props)
	}
}

// TestEditableTextIsMarkedForTheEditor checks the contract the inline editor
// relies on: a component says which element carries which prop.
func TestEditableTextIsMarkedForTheEditor(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))
	marked := doc.WithAttr("data-prop", "")
	if len(marked) == 0 {
		t.Fatalf("a card's text should be marked with data-prop so it can be edited in place")
	}
	found := false
	for _, n := range marked {
		if p, _ := htmltest.Attr(n, "data-prop"); p == "title" {
			found = true
		}
	}
	if !found {
		t.Errorf("the card's title should be marked as the title prop")
	}
	// The Edit control itself is added by the enhancement, never rendered
	// by the server, so it cannot exist where it would not work.
	for _, n := range doc.Elements("button") {
		if doc.AccessibleName(n) == "Edit card" {
			t.Errorf("the server must not render an Edit button; the script adds it")
		}
	}
}

// TestBadEditIsReportedNotSwallowed checks a rejected edit says so where the
// person is looking, rather than failing silently or throwing a 500.
func TestBadEditIsReportedNotSwallowed(t *testing.T) {
	h, id := canvasWithABlock(t)
	rec := postForm(t, h, "/canvas/"+id+"/props", url.Values{"prop-title": {""}})
	wantStatus(t, rec, http.StatusSeeOther)

	var msgs struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/message"), &msgs)
	reported := false
	for _, m := range msgs.Records {
		if m.Fields["role"] == "error" && strings.Contains(m.Fields["content"].(string), "did not save") {
			reported = true
		}
	}
	if !reported {
		t.Errorf("a rejected edit should be reported in the conversation")
	}
	// And the block is untouched.
	page := parse(t, get(t, h, "/"))
	block := page.WithAttr("data-block-id", id)[0]
	if !strings.Contains(htmltest.Text(block), "Shopping") {
		t.Errorf("a rejected edit must leave the block as it was")
	}
}

// TestPeopleGetNoDesignControls is the other half of the line: layout is the
// assistant's to set, so none of it is reachable from the page.
func TestPeopleGetNoDesignControls(t *testing.T) {
	h, id := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))
	for _, n := range doc.Elements("select") {
		t.Errorf("the canvas should offer no design controls, found a select named %q", doc.AccessibleName(n))
	}
	// The editor page is gone entirely.
	wantStatus(t, get(t, h, "/canvas/"+id+"/edit"), http.StatusNotFound)

	// An inline edit cannot smuggle layout in through the same form.
	postForm(t, h, "/canvas/"+id+"/props", url.Values{
		"prop-title": {"Still content"}, "span": {"12"}, "frame": {"bare"}, "tone": {"danger"},
	})
	var blocks struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	f := blocks.Records[0].Fields
	if f["span"] == int64(12) || f["frame"] == "bare" || f["tone"] == "danger" {
		t.Errorf("layout must not be settable through a content edit: %v", f)
	}
}
