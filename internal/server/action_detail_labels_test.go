package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestActionDetailLedeDoesNotShowRawStatusRunningIntoTimestamp asserts that the
// lede paragraph on an action detail page does not show the status value "Show"
// running into the timestamp. When show=true, a clear separator must appear
// between the mark form and the time (acceptance item 1).
func TestActionDetailLedeDoesNotShowRawStatusRunningIntoTimestamp(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
	}), &act)

	// Enable show so the mark appears in the lede.
	postForm(t, h, "/t/action/"+act.ID+"/props", url.Values{"prop-show": {"true"}})

	rec := get(t, h, "/t/action/"+act.ID)
	body := rec.Body.String()

	// Extract just the lede paragraph.
	ledeStart := strings.Index(body, `<p class="sw-lede">`)
	if ledeStart < 0 {
		t.Fatal("action detail has no lede")
	}
	ledeEnd := strings.Index(body[ledeStart:], "</p>")
	if ledeEnd < 0 {
		t.Fatalf("could not find end of lede\n%s", truncate(body))
	}
	lede := body[ledeStart : ledeStart+ledeEnd]

	// The mark form closes and the timestamp span opens with no separator.
	// They should be separated by at least one space character so that
	// screen readers do not concatenate "ShowStarted" and a person cannot
	// parse where the status ends and the time begins.
	if strings.Contains(lede, `</form><span class="sw-detail__when`) {
		t.Error("action detail lede shows \"Show\" running into the timestamp with no separator; expect at least one space between the mark form and the time span")
	}
}

// TestActionDetailDoesNotShowRawUrlLabel asserts that the action detail page
// does not display "Url" as a visible field label (acceptance item 2). The
// field should use plain language such as "URL".
func TestActionDetailDoesNotShowRawUrlLabel(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
		"url":   "https://example.com/ping",
	}), &act)

	body := get(t, h, "/t/action/"+act.ID).Body.String()

	if strings.Contains(body, "<dt>Url</dt>") {
		t.Error("action detail must not show \"Url\" as a field label; use plain language like \"URL\" or \"Webhook address\"")
	}
}

// TestActionDetailDoesNotShowRawMethodLabel asserts that the action detail page
// does not display "Method" as a visible raw field label (acceptance item 3).
// The term should be readable and clearly associated with its HTTP method value.
func TestActionDetailDoesNotShowRawMethodLabel(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title":  "Ping the alarm",
		"method": "POST",
	}), &act)

	body := get(t, h, "/t/action/"+act.ID).Body.String()

	if strings.Contains(body, "<dt>Method</dt>") {
		t.Error("action detail must not show \"Method\" as a raw field label; use a readable term")
	}
}

// TestActionDetailDoesNotShowRawEveryLabel asserts that the action detail page
// does not display "Every" as a visible field label (acceptance item 5). The
// field should use plain language such as "Schedule" or "Runs every".
func TestActionDetailDoesNotShowRawEveryLabel(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
	}), &act)

	body := get(t, h, "/t/action/"+act.ID).Body.String()

	if strings.Contains(body, "<dt>Every</dt>") {
		t.Error("action detail must not show \"Every\" as a field label; use plain language like \"Schedule\" or \"Runs every\"")
	}
}

// TestActionDetailLedeDoesNotShowCreatedLabel asserts that the action detail
// page lede does not contain "Created" as a field label (acceptance item 4).
func TestActionDetailLedeDoesNotShowCreatedLabel(t *testing.T) {
	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Ping the alarm",
	}), &act)

	body := get(t, h, "/t/action/"+act.ID).Body.String()

	if strings.Contains(body, ">Created ") {
		t.Error("action detail lede should not show \"Created\" as a label; use natural phrasing like \"Started\"")
	}
}
