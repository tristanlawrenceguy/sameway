package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Activity page tests for JSON sanitisation in "said" entries. These fetch
// the real pages (/activity and /t/activity/<id>) after seeding records via
// records.Record (the same way chat does). Acceptance items 1, 2 & 4.

func TestActivityPageSaidWithJSONDetailHidesMachineSyntax(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	records.Record(a.Store, "assistant", records.Change{
		Action: "said",
		Detail: `{"type": "task", "record": "abc123"}`,
	})

	body := get(t, h, "/activity").Body.String()
	pageText := said(body)

	if strings.Contains(pageText, "{") || strings.Contains(pageText, `"`) {
		t.Errorf("activity page must not show raw JSON characters in said entries: %q", pageText)
	}
	if !strings.Contains(pageText, "[content omitted]") {
		t.Errorf("activity page should show [content omitted] instead of the JSON block: %q", pageText)
	}
}

func TestActivityDetailPageSaidWithJSONDetailHidesMachineSyntax(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	entryID := records.Record(a.Store, "assistant", records.Change{
		Action: "said",
		Detail: `{"type": "task", "record": "abc123"}`,
	})

	body := get(t, h, "/t/activity/"+entryID).Body.String()
	pageText := said(body)

	if strings.Contains(pageText, "{") || strings.Contains(pageText, `"`) {
		t.Errorf("activity detail page must not show raw JSON characters in said entries: %q", pageText)
	}
	if !strings.Contains(pageText, "[content omitted]") {
		t.Errorf("activity detail page should show [content omitted] instead of the JSON block: %q", pageText)
	}
}

func TestActivityPageSaidWithNormalTextIsUnchanged(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	records.Record(a.Store, "assistant", records.Change{
		Action: "said",
		Detail: "remind me to take a break",
	})

	body := get(t, h, "/activity").Body.String()
	pageText := said(body)

	if !strings.Contains(pageText, "remind me to take a break") {
		t.Errorf("normal human text must be preserved in activity page: %q", pageText)
	}
}
