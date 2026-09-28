package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestChartSummaryHasDescriptiveLabel verifies that a chart rendered on the
// canvas with full detail produces a <summary> whose text is descriptive —
// derived from the chart's caption, not the hardcoded word "Numbers". This
// prevents duplicate accessible names when multiple charts appear on one page.
func TestChartSummaryHasDescriptiveLabel(t *testing.T) {
	_, h := newApp(t)
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	for _, task := range []map[string]any{
		{"title": "a", "done": true, "project": garden.ID},
		{"title": "b", "done": true, "project": garden.ID},
	} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", task), http.StatusCreated)
	}

	// A chart with an explicit caption gets that caption on its summary.
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "chart",
		"props":     map[string]any{"type": "task", "by": "project", "caption": "Tasks by project"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()

	// The summary text must contain the caption, not just "Numbers".
	if !strings.Contains(page, `<summary class="sw-pressable">Tasks by project</summary>`) {
		t.Errorf("the <summary> should say %q (the chart's caption), not a generic word", "Tasks by project")
	}

	// The summary must NOT be the hardcoded generic label.
	if strings.Contains(page, `<details class="sw-chart__numbers"><summary class="sw-pressable">Numbers</summary>`) {
		t.Error("the <summary> should not say the hardcoded word \"Numbers\"; it should use the chart's caption")
	}
}
