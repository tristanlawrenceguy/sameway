package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestDesignPageExampleHeadings checks that every example on /design is
// wrapped in an sw-example div that contains its own <h4> heading, giving
// screen reader users a distinguishing context for inputs with identical labels.
func TestDesignPageExampleHeadings(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/design")
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)

	examples := doc.WithAttr("class", "sw-example")
	if len(examples) == 0 {
		t.Fatal("/design has no sw-example wrappers")
	}

	for _, ex := range examples {
		subdoc := &htmltest.Doc{Root: ex}
		h4s := subdoc.Elements("h4")
		if len(h4s) < 1 {
			t.Errorf("sw-example should contain at least one h4, got %d", len(h4s))
			continue
		}
		text := htmltest.Text(h4s[0])
		if text == "" {
			t.Error("h4 inside sw-example must have visible text")
		}
		// Every heading should name the component, then an em-dash (—), then the example name.
		if !strings.Contains(text, " ") || !strings.Contains(text, "\u2014") {
			t.Errorf("h4 text %q should contain a space and em-dash (component — example)", text)
		}
	}
}

// TestDesignPageDatepickerExamplesHaveHeadings checks that each datepicker
// example on /design has its own <h4> heading that distinguishes it from the
// others, so screen reader users can tell "Due date" from "Start" from
// "Deadline (required)".
func TestDesignPageDatepickerExamplesHaveHeadings(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/design")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	for _, wantHeading := range []string{
		"datepicker — default",
		"datepicker — chosen",
		"datepicker — error",
	} {
		if !strings.Contains(body, "<h4 class=\"sw-small\">"+wantHeading+"</h4>") {
			t.Errorf("body should contain an h4 with %q for the datepicker example; "+
				"this is how screen reader users distinguish duplicate input labels like "+
				"'Due date', 'Start', and 'Deadline (required)'", wantHeading)
		}
	}

	// Also assert that no sw-example wrapper lacks an h4, since that would leave
	// some inputs without any distinguishing context.
	examples := docWithAttr(body, "class", "sw-example")
	for _, ex := range examples {
		if !strings.Contains(ex, "<h4") {
			t.Error("every sw-example must contain an h4 heading so screen readers " +
				"can distinguish between inputs with identical labels")
		}
	}
}

// docWithAttr is a lightweight helper to find elements by class in raw HTML.
func docWithAttr(src, key, value string) []string {
	var out []string
	start := 0
	for {
		idx := strings.Index(src[start:], `class="`+value+`"`)
		if idx == -1 {
			break
		}
		base := start + idx
		end := strings.Index(src[base:], "</div>")
		if end == -1 {
			out = append(out, src[base:])
			break
		}
		out = append(out, src[base:base+end])
		start = base + end + 6
	}
	return out
}
