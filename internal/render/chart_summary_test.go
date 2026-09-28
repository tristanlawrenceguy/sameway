package render_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestChartSummaryUsesCaption ensures every chart rendered at full detail
// produces a <summary> whose text reflects what the chart counts, not the
// generic word "Numbers". Each example has a caption set; the summary on
// the disclosure must use it (or an explicit summary prop), so screen
// reader users can distinguish between multiple charts on /design.
func TestChartSummaryUsesCaption(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	for _, c := range reg.Components() {
		if c.Manifest.Name != "chart" {
			continue
		}
		for _, ex := range c.Manifest.Examples {
			out, err := c.Render(ex.Props)
			if err != nil {
				t.Fatalf("%s/%s: %v", c.Manifest.Name, ex.Name, err)
			}
			got := string(out)

			// Only check charts that render a <details>/<summary> block.
			// Detail "full" (the default), not glance/brief/page/empty/problem.
			if !strings.Contains(got, `<details class="sw-chart__numbers">`) {
				continue
			}

			caption, _ := ex.Props["caption"].(string)
			t.Logf("example %q: caption=%q", ex.Name, caption)
			if caption == "" {
				continue // no caption to check against
			}
			expectedSummary := fmt.Sprintf("<summary class=\"sw-pressable\">%s</summary>", caption)
			if !strings.Contains(got, expectedSummary) {
				t.Errorf("%s/%s: the <summary> should say %q (the chart's caption), not a generic word",
					c.Manifest.Name, ex.Name, caption)
			}
		}
	}
}

// TestChartSummaryPropOverridesCaption ensures that when a chart has an
// explicit summary prop, it uses that instead of the caption. This allows
// special cases where the disclosure label needs to differ from the title.
func TestChartSummaryPropOverridesCaption(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	// Render a chart with both caption and summary props.
	out, err := reg.Render("chart", map[string]any{
		"detail":  "full",
		"caption": "Tasks done each week",
		"summary": "How many tasks per week",
		"series": []map[string]any{
			{"label": "W34", "value": 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := string(out)
	if !strings.Contains(got, `<summary class="sw-pressable">How many tasks per week</summary>`) {
		t.Errorf("the <summary> should use the explicit summary prop %q", "How many tasks per week")
	}
}
