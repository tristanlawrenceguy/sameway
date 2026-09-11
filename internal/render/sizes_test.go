package render_test

import (
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// calendarProps is one month with two events, rendered at each size.
func calendarProps(detail string) map[string]any {
	return map[string]any{
		"month": "2026-09", "today": "2026-09-11", "detail": detail, "upcoming": 3,
		"caption": "September 2026",
		"events": []any{
			map[string]any{"date": "2026-09-11", "time": "10:00", "label": "Design review"},
			map[string]any{"date": "2026-09-14", "label": "Rent due"},
		},
	}
}

// TestCalendarSpansItsSizes is the range the same component has to cover: a
// dot with a number that only says something needs attention, an agenda of
// what is next, a month, and a full page. Every size says the same thing to
// a screen reader that it says on screen; none of them drops the content.
func TestCalendarSpansItsSizes(t *testing.T) {
	reg := builtins(t)
	for _, tc := range []struct {
		detail string
		want   func(t *testing.T, doc *htmltest.Doc, out string)
	}{
		{"glance", func(t *testing.T, doc *htmltest.Doc, out string) {
			// The number alone means nothing without words. The digit is
			// decorative and the real name carries the month and the count.
			text := htmltest.Text(doc.Root)
			if text != "2September 2026: 2 coming up" {
				t.Errorf("glance reads as %q", text)
			}
			if len(doc.Elements("table")) != 0 {
				t.Errorf("glance must not be a table")
			}
		}},
		{"brief", func(t *testing.T, doc *htmltest.Doc, out string) {
			items := doc.WithAttr("data-event", "")
			if len(items) != 2 {
				t.Errorf("brief should list what is coming up, got %d items", len(items))
			}
			if len(doc.Elements("table")) != 0 {
				t.Errorf("brief must not be a table")
			}
		}},
		{"full", func(t *testing.T, doc *htmltest.Doc, out string) {
			if len(doc.Elements("table")) != 1 || len(doc.Elements("caption")) != 1 {
				t.Errorf("full should be one captioned table")
			}
			if len(doc.WithAttr("aria-current", "date")) != 1 {
				t.Errorf("full should mark today with aria-current=date")
			}
			var headers int
			doc.Walk(func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "th" {
					if scope, _ := htmltest.Attr(n, "scope"); scope == "col" {
						headers++
					}
				}
			})
			if headers != 7 {
				t.Errorf("full should have seven column headers, got %d", headers)
			}
		}},
		{"page", func(t *testing.T, doc *htmltest.Doc, out string) {
			if len(doc.Elements("table")) != 1 {
				t.Errorf("page should still be one table")
			}
			if len(doc.WithAttr("data-detail", "page")) == 0 {
				t.Errorf("page should say which size it is, so an agent can tell")
			}
		}},
	} {
		t.Run(tc.detail, func(t *testing.T) {
			out, err := reg.Render("calendar", calendarProps(tc.detail))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := htmltest.Parse(string(out))
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.WithAttr("data-detail", tc.detail)) == 0 {
				t.Errorf("%s: the rendered size should be readable from the markup", tc.detail)
			}
			tc.want(t, doc, string(out))
		})
	}
}

// TestCalendarGlanceStaysQuietWhenNothingIsComingUp: an attention dot that
// is always lit is not an attention dot.
func TestCalendarGlanceStaysQuietWhenNothingIsComingUp(t *testing.T) {
	reg := builtins(t)
	props := calendarProps("glance")
	props["events"] = []any{}
	out, err := reg.Render("calendar", props)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := htmltest.Parse(string(out))
	if err != nil {
		t.Fatal(err)
	}
	if got := htmltest.Text(doc.Root); got != "0September 2026: 0 coming up" {
		t.Errorf("empty glance reads as %q", got)
	}
}
