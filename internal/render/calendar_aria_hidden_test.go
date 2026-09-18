package render_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestCalendarNoAriaHiddenWeekdayHeaders renders a full-detail calendar and
// asserts that the weekday header cells carry no inner span with
// aria-hidden="true".  Each <th scope="col"> must have the abbreviation as
// direct text content (visible to everyone), plus an aria-label on the <th>
// itself for the full name.  Regression guard for backlog item 0297,
// acceptance items 1 and 2.
func TestCalendarNoAriaHiddenWeekdayHeaders(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("calendar", map[string]any{
		"month":  "2026-09",
		"today":  "2026-09-11",
		"detail": "full",
		"events": []any{
			map[string]any{"date": "2026-09-11", "label": "Design review"},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	doc, err := htmltest.Parse(string(got))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	thElements := doc.Elements("th")
	if len(thElements) == 0 {
		t.Fatal("output lacks <th> elements; calendar not rendered as table?\ngot:\n" + string(got))
	}

	for _, th := range thElements {
		scope, _ := htmltest.Attr(th, "scope")
		if scope != "col" {
			continue // only check column headers
		}

		// The <th> must have an aria-label with the full day name.
		label, hasLabel := htmltest.Attr(th, "aria-label")
		if !hasLabel {
			t.Errorf("<th scope=\"col\"> is missing aria-label (full weekday name)\nfull output:\n%s", got)
			continue
		}

		// The full day names are the only possible values.
		fullNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		found := false
		for _, fn := range fullNames {
			if label == fn {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("aria-label on <th scope=\"col\"> is %q; want a full weekday name\nfull output:\n%s", label, got)
		}

		// There must be no span with aria-hidden inside this <th>.
		walkChildren(th, func(n *html.Node) {
			if n.Data == "span" {
				for _, a := range n.Attr {
					if a.Key == "aria-hidden" && a.Val == "true" {
						t.Errorf("weekday header <th> contains span with aria-hidden=\"true\" (%q); visible text must not be hidden from AT\nfull output:\n%s", htmltest.Text(n), got)
					}
				}
			}
		})

		// The direct text content of the <th> must be a weekday abbreviation.
		text := strings.TrimSpace(htmltest.Text(th))
		abbrevs := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		found = false
		for _, ab := range abbrevs {
			if text == ab {
				found = true
				break
			}
		}
		if !found && len(text) > 0 {
			t.Errorf("weekday header <th> contains %q; expected a weekday abbreviation (Mon/Sun)\nfull output:\n%s", text, got)
		}
	}
}

// walkChildren visits every descendant of n (including n itself).
func walkChildren(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkChildren(c, fn)
	}
}

// TestCalendarGlanceCountNotAriaHidden renders a calendar in glance detail and
// asserts that the count span does NOT carry aria-hidden="true", so sighted
// users seeing the number badge also have it accessible to screen readers.
// Regression guard for backlog item 0297, acceptance items 1 and 3.
func TestCalendarGlanceCountNotAriaHidden(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("calendar", map[string]any{
		"month":  "2026-09",
		"detail": "glance",
		"today":  "2026-09-11",
		"events": []any{
			map[string]any{"date": "2026-09-11", "time": "14:00", "label": "Design review"},
			map[string]any{"date": "2026-09-14", "label": "Standup"},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	doc, err := htmltest.Parse(string(got))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	countElements := doc.WithAttr("class", "sw-calendar__count")
	if len(countElements) == 0 {
		t.Fatal("output lacks <span class=\"sw-calendar__count\">; glance not rendered?\ngot:\n" + string(got))
	}

	for _, span := range countElements {
		for _, a := range span.Attr {
			if a.Key == "aria-hidden" && a.Val == "true" {
				t.Errorf("glance count span has aria-hidden=\"true\" but the number is visually rendered; screen reader users cannot hear it\nfull output:\n%s", got)
			}
		}

		text := htmltest.Text(span)
		if text != "2" {
			t.Errorf("count span text is %q; want \"2\" (two events)\nfull output:\n%s", text, got)
		}
	}
}
