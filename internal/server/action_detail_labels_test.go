package server_test

// Tests for raw field labels on the action detail page. The schema file
// examples/workspaces/starter/schema/action.yaml defines fields url, method
// and every without human-readable label overrides, so the detail page shows
// "Url", "Method" and "Every" as <dt> values (backlog 0620). Acceptance items
// 2–5 of that item.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestActionDetailDoesNotShowRawUrlLabel asserts that the action detail page
// does not display "Url" as a visible field label when the url field has a value.
// The schema should provide a plain-language label such as "URL". (Acceptance 2.)
func TestActionDetailDoesNotShowRawUrlLabel(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestActionDetailLedeDoesNotShowCreatedLabel asserts that the action detail page
// lede does not contain "Created" as a visible label (Acceptance 4). The text
// should read naturally, e.g. "Started …".
func TestActionDetailLedeDoesNotShowCreatedLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Ping the alarm",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if strings.Contains(page, ">Created ") {
		t.Error("action detail lede should not show \"Created\" as a label; use natural phrasing like \"Started\"")
	}
}

// TestActionDetailLedeSpacing ensures the status checkbox and its timestamp are
// separated by whitespace so they do not merge into one accessibility name.
func TestActionDetailLedeSpacing(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Check the thermostat",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// The lede should not contain </form><span class="sw-detail__when
	// without a space between them.
	if strings.Contains(page, "</form><span class=\"sw-detail__when") {
		t.Error("action detail lede has no whitespace between the checkbox form and timestamp span")
	}
}

// TestActionDetailShowLabelDoesNotRunIntoTimestamp verifies that the raw status
// value (e.g. "Show" or "Done") does not run directly into the timestamp in the
// action detail page lede. This confirms the spacing fix between box and facts.
func TestActionDetailShowLabelDoesNotRunIntoTimestamp(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title": "Verify agent status",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// When the checkbox is unchecked the lede shows "Show" followed by a space
	// and then the timestamp. They must not merge into one word in the HTML.
	if strings.Contains(page, "</form><span class=\"sw-detail__when") {
		t.Error("the raw status value runs directly into the timestamp span; there should be whitespace between them")
	}

	_ = url.QueryEscape // use the import to avoid unused import error
}

// TestActionDetailShowsCorrectLabels asserts that the action detail page renders
// human-readable labels for all three schema fields: URL, HTTP method, and
// Schedule. (Acceptance 2–3, 5.)
func TestActionDetailShowsCorrectLabels(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title":  "Full test action",
		"url":    "https://example.com/hook",
		"method": "GET",
		"every":  "day",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	for _, want := range []string{"<dt>URL</dt>", "<dt>HTTP method</dt>", "<dt>Schedule</dt>"} {
		if !strings.Contains(page, want) {
			t.Errorf("action detail should contain %q; page body:\n%s", want, truncate(page))
		}
	}

	for _, raw := range []string{"<dt>Url</dt>", "<dt>Method</dt>", "<dt>Every</dt>"} {
		if strings.Contains(page, raw) {
			t.Errorf("action detail must not show %q; use plain language", raw)
		}
	}
}

// TestActionDetailLabelCombination verifies that both negative and positive
// assertions hold simultaneously: no raw labels appear and all correct labels
// do. This is the comprehensive acceptance test for backlog 0620.
func TestActionDetailLabelCombination(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{
		"title":  "Combined label test",
		"url":    "https://example.com/test",
		"method": "GET",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	// Negative checks: raw labels must not appear.
	rawLabels := []string{"<dt>Url</dt>", "<dt>Method</dt>", "<dt>Every</dt>"}
	for _, raw := range rawLabels {
		if strings.Contains(page, raw) {
			t.Errorf("raw label %q found in action detail page", raw)
		}
	}

	// Positive checks: correct labels must appear.
	correctLabels := map[string]string{
		"<dt>URL</dt>":         "url field",
		"<dt>HTTP method</dt>": "method field",
	}
	for want, desc := range correctLabels {
		if !strings.Contains(page, want) {
			t.Errorf("action detail should show %q for the %s; found in:\n%s", want, desc, truncate(page))
		}
	}
}

// TestActionDetailAcceptsUrlAndMethodFields verifies that creating an action with
// url and method fields via the API succeeds and those values appear on the page.
func TestActionDetailAcceptsUrlAndMethodFields(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
