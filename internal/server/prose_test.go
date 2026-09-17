package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// A person edits structured text as it is shown; what they made comes back
// as Markdown, with headings at the level the source had.
func TestEditedProseComesBackAsMarkdown(t *testing.T) {
	a, h := newApp(t)
	rec, err := a.Store.Create("note", map[string]any{"title": "Plan", "body": "# Beds\n\nThree of them."})
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/note/"+rec.ID).Body.String()
	if !strings.Contains(page, `data-prose-level="3"`) || !strings.Contains(page, `data-source="# Beds`) {
		t.Errorf("the page says what level the source was shown at, and keeps the source: %.600s", page)
	}

	form := url.Values{
		"html-body":  {`<h3>Beds</h3><p>Three of them, <strong>raised</strong>.</p><ul><li>Dig the pond</li><li>Order compost</li></ul><h4>Later</h4><p>Plant garlic.</p>`},
		"level-body": {"3"},
	}
	wantStatus(t, postForm(t, h, "/t/note/"+rec.ID+"/props", form), http.StatusSeeOther)
	updated, _ := a.Store.Get("note", rec.ID)
	body, _ := updated.Fields["body"].(string)
	for _, want := range []string{"# Beds\n", "Three of them, **raised**.", "- Dig the pond\n- Order compost", "## Later\n", "Plant garlic."} {
		if !strings.Contains(body, want) {
			t.Errorf("the edit should be Markdown with %q, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "### ") || strings.Contains(body, "<") {
		t.Errorf("no shifted headings and no HTML in the source: %q", body)
	}

	// The same for a text block on the canvas, shown with # as h2.
	h2, id := canvasWithABlock(t)
	wantStatus(t, postForm(t, h2, "/canvas/"+id+"/props", url.Values{"html-title": {"<p>Weekly <em>shop</em></p>"}, "level-title": {"2"}}), http.StatusSeeOther)
	var blocks struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h2, "/api/block"), &blocks)
	if title := blocks.Records[0].Fields["props"].(map[string]any)["title"]; title != "Weekly *shop*\n" {
		t.Errorf("a block prop edited as rich text is saved as Markdown, got %q", title)
	}
}

// The editor switches between the rendered view and the source without
// losing anything, through one route that converts either way.
func TestProseConvertsEitherWay(t *testing.T) {
	_, h := newApp(t)
	var out struct{ HTML, Markdown, Error string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/prose", map[string]any{"markdown": "# Beds\n\nThree.", "level": 3}), &out)
	if !strings.Contains(out.HTML, "<h3") || !strings.Contains(out.HTML, "<p>Three.</p>") {
		t.Errorf("markdown in, the page's html out: %q", out.HTML)
	}
	decode(t, postJSON(t, h, http.MethodPost, "/api/prose", map[string]any{"html": out.HTML, "level": 3}), &out)
	if !strings.HasPrefix(out.Markdown, "# Beds\n\nThree.") {
		t.Errorf("html in, the same markdown out: %q", out.Markdown)
	}
	rec := postJSON(t, h, http.MethodPost, "/api/prose", map[string]any{"level": 3})
	wantStatus(t, rec, http.StatusBadRequest)
}
