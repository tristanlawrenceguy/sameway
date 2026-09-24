package render

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// A chart's bars, lines and target are graphics a person has to see to
// read the chart, so each colour they can take stands 3:1 against the
// page and the card, in light and in dark (WCAG 1.4.11). They take a
// list's colour, the accent, or the warning for a target.
func TestChartColoursStandOutFromThePage(t *testing.T) {
	raw, err := os.ReadFile("../../design/tokens/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	themes := map[string]string{"light": block(css, ":root {"), "dark": block(css, `:root[data-theme="dark"] {`)}
	marks := []string{"list-1", "list-2", "list-3", "list-4", "list-5", "list-6", "accent", "warning"}
	for theme, vars := range themes {
		for _, ground := range []string{"bg", "bg-raised"} {
			g := color(t, vars, ground)
			for _, m := range marks {
				if r := contrast(color(t, vars, m), g); r < 3 {
					t.Errorf("%s: %s on %s is %.2f:1; a chart mark needs 3:1", theme, m, ground, r)
				}
			}
		}
	}
}

func block(css, start string) string {
	i := strings.Index(css, start)
	if i < 0 {
		return ""
	}
	j := strings.Index(css[i:], "}")
	return css[i : i+j]
}

func color(t *testing.T, vars, name string) [3]float64 {
	t.Helper()
	m := regexp.MustCompile(`--sw-color-` + regexp.QuoteMeta(name) + `:\s*#([0-9a-fA-F]{6});`).FindStringSubmatch(vars)
	if m == nil {
		t.Fatalf("no --sw-color-%s", name)
	}
	var c [3]float64
	for i := 0; i < 3; i++ {
		v, _ := strconv.ParseUint(m[1][i*2:i*2+2], 16, 8)
		c[i] = float64(v) / 255
	}
	return c
}

func luminance(c [3]float64) float64 {
	var l [3]float64
	for i, v := range c {
		if v <= 0.03928 {
			l[i] = v / 12.92
		} else {
			l[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*l[0] + 0.7152*l[1] + 0.0722*l[2]
}

func contrast(a, b [3]float64) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}
