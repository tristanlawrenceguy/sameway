package server_test

import (
	"strings"
	"testing"
)

// Tests for redundant information on record detail pages (task 0176).
// These pin that a screen reader user does not hear the same fact twice.

// TestDetailTitleAppearsOnce checks that a note's title is spoken only once
// on its detail page — in the h1 heading, and nowhere else including crumbs.
// This covers acceptance item 1 (title duplication) and item 4 for notes.
func TestDetailTitleAppearsOnce(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Plan the garden",
	})
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/note/"+rec.ID).Body.String()

	// The title must appear in the h1 heading.
	if !strings.Contains(page, "<h1") || !strings.Contains(page, ">Plan the garden</h1>") {
		t.Errorf("the detail page should have an h1 with the title\n%s", truncate(page))
	}

	// The crumb must not repeat the title — it should only link to Notes.
	if strings.Contains(page, `aria-current="page">Plan the garden`) ||
		strings.Contains(page, ">Plan the garden</li>") {
		t.Errorf("the crumb should not repeat the record title; screen readers would hear it twice\n%s", truncate(page))
	}

	// The body must contain the title at most once as visible content.
	// We count occurrences between HTML tags only (not inside attributes).
	body := extractBody(page)
	count := countVisibleText(body, "Plan the garden")
	if count != 1 {
		t.Errorf("the title 'Plan the garden' appears %d time(s) in the body; it should appear exactly once (in h1)\n%s", count, truncate(body))
	}
}

// extractBody returns everything between <main> and </main>.
func extractBody(s string) string {
	mainOpen := strings.Index(s, `<main id="main"`)
	if mainOpen < 0 {
		return ""
	}
	bodyStart := strings.IndexByte(s[mainOpen:], '>')
	if bodyStart < 0 {
		return ""
	}
	mainEnd := strings.Index(s[mainOpen+bodyStart+1:], "</main>")
	if mainEnd < 0 {
		return s[mainOpen+bodyStart+2:]
	}
	return s[mainOpen+bodyStart+2 : mainOpen+bodyStart+2+mainEnd]
}

// countVisibleText counts how many times text appears as visible content
// (between > and <) in the given HTML string, skipping attribute values.
func countVisibleText(html, want string) int {
	count := 0
	rest := html
	for {
		open := strings.Index(rest, ">")
		if open < 0 {
			break
		}
		close := strings.Index(rest[open+1:], "<")
		if close < 0 {
			// text after last tag — count remaining occurrences
			count += strings.Count(rest[open+1:], want)
			break
		}
		text := rest[open+1 : open+1+close]
		// Remove attribute values (content between quotes inside tags).
		// This prevents counting text that appears in aria-label, href, etc.
		cleaned := removeAttrValues(text)
		count += strings.Count(cleaned, want)
		rest = rest[open+1+close:]
	}
	return count
}

// removeAttrValues strips quoted attribute values from a text segment that was
// between > and <. This handles cases where visible-text detection would
// accidentally include aria-label or other attribute content.
func removeAttrValues(s string) string {
	var result strings.Builder
	inQuote := false
	quoteChar := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote {
			if ch == quoteChar && (i == 0 || s[i-1] != '\\') {
				inQuote = false
			}
			continue // skip content inside quotes
		}
		if ch == '"' || ch == '\'' {
			inQuote = true
			quoteChar = ch
			continue
		}
		result.WriteByte(ch)
	}
	return result.String()
}
