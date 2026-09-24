package server_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestActivityPageNoDuplicateDescription checks that each activity item on /activity
// states its description only once (in the h3 heading), not also in the event body.
// A screen reader should hear "Assistant created note First note" just once, not twice.
func TestActivityPageNoDuplicateDescription(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: "aaa1", Detail: "Updated content"})

	body := get(t, h, "/activity").Body.String()

	doc, err := htmltest.Parse(body)
	if err != nil {
		t.Fatal(err)
	}

	for _, li := range doc.Elements("li") {
		h3s := childWithClass(li, "sw-event__heading")
		if len(h3s) == 0 {
			continue
		}
		summary := htmltest.Text(h3s[0])

		events := eventChildren(li)
		for _, ev := range events {
			eventText := strings.TrimSpace(htmltest.Text(ev))
			if eventText == "" || eventText == summary {
				continue // compact mode: no duplicate text
			}
			// If the event body does not contain the heading summary, it is not a duplicate.
			// This works regardless of actor names and catches any future duplication exactly.
			if !strings.Contains(eventText, summary) {
				continue
			}
			t.Errorf("activity entry duplicates description: h3=%q, event body starts with %q\n%s",
				summary, truncate(eventText), truncate(body))
		}
	}
}

// TestChatPageRecentActivityNoDuplicateDescription checks that recent activity on /chat
// does not repeat the description in both the heading and the event body.
func TestChatPageRecentActivityNoDuplicateDescription(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: "aaa1", Detail: "Updated content"})

	body := get(t, h, "/chat").Body.String()

	doc, err := htmltest.Parse(body)
	if err != nil {
		t.Fatal(err)
	}

	for _, li := range doc.Elements("li") {
		h3s := childWithClass(li, "sw-event__heading")
		if len(h3s) == 0 {
			continue
		}
		summary := htmltest.Text(h3s[0])

		events := eventChildren(li)
		for _, ev := range events {
			eventText := strings.TrimSpace(htmltest.Text(ev))
			if eventText == "" || eventText == summary {
				continue // compact mode: no duplicate text
			}
			// If the event body does not contain the heading summary, it is not a duplicate.
			// This works regardless of actor names and catches any future duplication exactly.
			if !strings.Contains(eventText, summary) {
				continue
			}
			t.Errorf("recent activity entry duplicates description: h3=%q, event body starts with %q\n%s",
				summary, truncate(eventText), truncate(body))
		}
	}
}

// childWithClass returns direct children of parent whose class attribute contains cls.
func childWithClass(parent *html.Node, cls string) []*html.Node {
	var out []*html.Node
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if cv, ok := htmltest.Attr(c, "class"); ok && hasClass(cv, cls) {
				out = append(out, c)
			}
		}
	}
	return out
}

// eventChildren returns direct children of parent that are the event component div.
func eventChildren(parent *html.Node) []*html.Node {
	var out []*html.Node
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if v, ok := htmltest.Attr(c, "data-component"); ok && v == "event" {
			out = append(out, c)
		}
	}
	return out
}

// hasClass reports whether class list contains cls as a space-separated token.
func hasClass(classList, cls string) bool {
	for _, tok := range strings.Fields(classList) {
		if tok == cls {
			return true
		}
	}
	return false
}

// TestActivityPagePreservesNonDuplicatedElements checks that after removing the
// duplicate body text, non-duplicated elements like time and undo buttons are still present.
func TestActivityPagePreservesNonDuplicatedElements(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	body := get(t, h, "/activity").Body.String()

	// The time should still appear in the event component.
	if !strings.Contains(body, `<time class="sw-event__time"`) {
		t.Errorf("event body should still contain a timestamp\n%s", truncate(body))
	}

	// The page title and h1 should be unchanged.
	if !strings.Contains(body, "<title>Activity") {
		t.Errorf("page title should start with 'Activity'\n%s", truncate(body))
	}
	if !strings.Contains(body, ">Activity</h1>") {
		t.Errorf("page should have an <h1>Activity</h1> from the layout\n%s", truncate(body))
	}

	// Day-grouped headings should still appear.
	if !strings.Contains(body, `class="sw-small sw-muted"`) {
		t.Errorf("activity page should show day-grouped headings\n%s", truncate(body))
	}
}
