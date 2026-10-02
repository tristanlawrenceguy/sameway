package server_test

// Tests for raw field labels on the action detail page. The schema file
// examples/workspaces/starter/schema/action.yaml defines fields url, method
// and every without human-readable label overrides, so the detail page shows
// "Url", "Method" and "Every" as <dt> values (backlog 0620). Acceptance items
// 2–5 of that item.

import (
	"net/http"
	"strings"
	"testing"
)

// TestActionDetailDoesNotShowRawUrlLabel asserts that the action detail page
// does not display "Url" as a visible field label when the url field has a value.
// The schema should provide a plain-language label such as "URL". (Acceptance 2.)
func TestActionDetailDoesNotShowRawUrlLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Ping the alarm",
		"url":   "https://example.com/ping",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if strings.Contains(page, "<dt>Url</dt>") {
		t.Error("action detail must not show \"Url\" as a field label; use plain language like \"URL\"")
	}
}

// TestActionDetailDoesNotShowRawMethodLabel asserts that the action detail page
// does not display "Method" as a raw field label in its definition list. The
// term should be readable and clearly associated with its HTTP method value.
// (Acceptance 3.)
func TestActionDetailDoesNotShowRawMethodLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Ping the alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if strings.Contains(page, "<dt>Method</dt>") {
		t.Error("action detail must not show \"Method\" as a raw field label; use a readable term like \"HTTP method\"")
	}
}

// TestActionDetailDoesNotShowRawEveryLabel asserts that the action detail page
// does not display "Every" as a visible field label in its definition list. The
// field should use plain language such as "Schedule". (Acceptance 5.)
func TestActionDetailDoesNotShowRawEveryLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Ping the alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if strings.Contains(page, "<dt>Every</dt>") {
		t.Error("action detail must not show \"Every\" as a field label; use plain language like \"Schedule\"")
	}
}

// TestActionDetailShowLabelDoesNotRunIntoTimestamp asserts that the lede paragraph
// on an action detail page has a separator between the status mark form and the
// timestamp span, so "Show" does not run into "Started". (Acceptance 1.)
func TestActionDetailShowLabelDoesNotRunIntoTimestamp(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{"title": "Ping the alarm"})
	if err != nil {
		t.Fatal(err)
	}

	postForm(t, h, "/t/action/"+rec.ID+"/props", map[string][]string{"prop-show": {"true"}})

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	ledeStart := strings.Index(page, `<p class="sw-lede">`)
	if ledeStart < 0 {
		t.Fatal("action detail has no lede")
	}
	ledeEnd := strings.Index(page[ledeStart:], "</p>")
	if ledeEnd < 0 {
		t.Fatalf("could not find end of lede\n%s", truncate(page))
	}
	lede := page[ledeStart : ledeStart+ledeEnd]

	if strings.Contains(lede, `</form><span class="sw-detail__when`) {
		t.Error("action detail lede shows \"Show\" running into the timestamp with no separator; expect at least one space between the mark form and the time span")
	}
}

// TestActionDetailHasCorrectUrlLabel asserts that when a label override is in
// place, the action detail page renders "URL" as the <dt> for the url field.
func TestActionDetailHasCorrectUrlLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Ping the alarm",
		"url":   "https://example.com/ping",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if !strings.Contains(page, "<dt>URL</dt>") {
		t.Errorf("action detail should show \"<dt>URL</dt>\" for the url field; page body:\n%s", truncate(page))
	}
}

// TestActionDetailHasCorrectMethodLabel asserts that when a label override is in
// place, the action detail page renders "HTTP method" as the <dt> for the method field.
func TestActionDetailHasCorrectMethodLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title":  "Ping the alarm",
		"method": "GET",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if !strings.Contains(page, "<dt>HTTP method</dt>") {
		t.Errorf("action detail should show \"<dt>HTTP method</dt>\" for the method field; page body:\n%s", truncate(page))
	}
}

// TestActionDetailHasCorrectEveryLabel asserts that when a label override is in
// place, the action detail page renders "Schedule" as the <dt> for the every field.
func TestActionDetailHasCorrectEveryLabel(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Ping the alarm",
		"every": "day",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if !strings.Contains(page, "<dt>Schedule</dt>") {
		t.Errorf("action detail should show \"<dt>Schedule</dt>\" for the every field; page body:\n%s", truncate(page))
	}
}

// TestActionDetailLabelOverridesCoversAllFields verifies all three label fixes
// are present at once: URL, HTTP method, and Schedule (no Url, Method, Every).
func TestActionDetailLabelOverridesCoversAllFields(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title":  "Ping the alarm",
		"url":    "https://example.com/ping",
		"method": "GET",
		"every":  "day",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	checks := []struct {
		notWant string
		want    string
	}{
		{"<dt>Url</dt>", "<dt>URL</dt>"},
		{"<dt>Method</dt>", "<dt>HTTP method</dt>"},
		{"<dt>Every</dt>", "<dt>Schedule</dt>"},
	}

	for _, c := range checks {
		if strings.Contains(page, c.notWant) {
			t.Errorf("action detail must not show %q; found in:\n%s", c.notWant, truncate(page))
		}
		if !strings.Contains(page, c.want) {
			t.Errorf("action detail should show %q for the field label; found in:\n%s", c.want, truncate(page))
		}
	}
}

// TestActionDetailAcceptsUrlAndMethodFields verifies that creating an action with
// url and method fields via the API succeeds and those values appear on the page.
func TestActionDetailAcceptsUrlAndMethodFields(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title":  "Webhook to example",
		"url":    "https://example.com/hook",
		"method": "POST",
	}), &act)

	rec := get(t, h, "/t/action/"+act.ID)

	body := rec.Body.String()

	if strings.Contains(body, "<dt>Url</dt>") {
		t.Error("action detail must not show \"<dt>Url</dt>\"")
	}
	if strings.Contains(body, "<dt>Method</dt>") {
		t.Error("action detail must not show \"<dt>Method</dt>\"")
	}

	if !strings.Contains(body, "<dt>URL</dt>") {
		t.Errorf("action detail should show \"<dt>URL</dt>\"; page body:\n%s", truncate(body))
	}
	if !strings.Contains(body, "<dt>HTTP method</dt>") {
		t.Errorf("action detail should show \"<dt>HTTP method</dt>\"; page body:\n%s", truncate(body))
	}
}

// TestActionDetailDumpDtLabels is a debug-only test that prints all <dt> elements
// from an action detail page to verify what labels are actually rendered.
func TestActionDetailDumpDtLabels(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{"title": "Debug"})
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/action/"+rec.ID).Body.String()
	start := 0
	for {
		idx := strings.Index(page[start:], "<dt>")
		if idx < 0 {
			break
		}
		end := strings.Index(page[start+idx:], "</dt>")
		t.Logf("dt: %s", page[start+idx:start+idx+end+5])
		start = start + idx + end + 5
	}
}
