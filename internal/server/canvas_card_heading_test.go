package server_test

import (
	"net/url"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestCanvasCardDetailHeadingH2 ensures that opening a card block on its own
// /canvas/{id} page renders the card's title as an h2 inside <article>, not
// h3. The card component defaults level to 3, so without the fix in expanded()
// the hierarchy would skip from h1 (the page heading) directly to h3. This is
// acceptance item 1 and 2 of task: heading hierarchy must be sequential H1 → H2.
func TestCanvasCardDetailHeadingH2(t *testing.T) {
	a, h := newApp(t)

	// Add a card block via the assistant so it lands on the canvas like real usage.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{
			"title": "My card title",
			"body":  "Some body text.",
		}}),
		{Text: "Added a card."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}})

	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	cardID := ""
	for _, b := range blocks.Records {
		if b.Fields["component"] == "card" {
			cardID = b.ID
		}
	}
	if cardID == "" {
		t.Fatal("no card block was added")
	}

	doc := parse(t, get(t, h, "/canvas/"+cardID))

	// There must be exactly one h1 (the page title) and no h3 inside the article.
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Errorf("expected exactly one h1 on the card detail page, got %d", len(h1s))
	}

	h3s := doc.Elements("h3")
	if len(h3s) > 0 {
		t.Errorf("card title inside <article> should not be h3 (skips from h1 to h3), got %d h3 elements", len(h3s))
	}

	// The card's heading inside <article class="sw-card"> must be h2.
	h2s := doc.Elements("h2")
	if len(h2s) == 0 {
		t.Fatal("expected an h2 for the card title inside <article>, got none — hierarchy skips h1 → h3")
	}

	// The h2 should contain the card's title text.
	found := false
	for _, n := range h2s {
		if text := htmltest.Text(n); text == "My card title" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an h2 with text \"My card title\" inside <article>, got %v", func() []string {
			var out []string
			for _, n := range h2s {
				out = append(out, htmltest.Text(n))
			}
			return out
		}())
	}

	// The article element exists on the detail page.
	articles := doc.WithAttr("data-component", "card")
	if len(articles) == 0 {
		t.Fatal("expected one card <article> on the detail page")
	}
}
