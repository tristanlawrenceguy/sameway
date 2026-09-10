package tokens_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
)

// The palette contract. Every text role must reach 7:1 (WCAG 2.2 AAA) on
// every surface it can appear on; non-text indicators must reach 3:1.
// Change tokens.json freely; this test tells you what broke.

type pair struct {
	fg, bg string
	min    float64
	why    string
}

var pairs = []pair{
	{"fg", "bg", 7, "body text"},
	{"fg", "bg-muted", 7, "body text on muted surface"},
	{"fg", "bg-raised", 7, "body text on cards"},
	{"fg-muted", "bg", 7, "secondary text"},
	{"fg-muted", "bg-muted", 7, "secondary text on muted surface"},
	{"fg-muted", "bg-raised", 7, "secondary text on cards"},
	{"accent", "bg", 7, "links"},
	{"accent", "bg-muted", 7, "links on muted surface"},
	{"accent-fg", "accent", 7, "primary button label"},
	{"human", "bg", 7, "human provenance text"},
	{"human", "human-soft", 7, "human badge"},
	{"assistant", "bg", 7, "assistant provenance text"},
	{"assistant", "assistant-soft", 7, "assistant badge"},
	{"system", "bg", 7, "system provenance text"},
	{"system", "system-soft", 7, "system badge"},
	{"success", "bg", 7, "success text"},
	{"success", "success-soft", 7, "success alert"},
	{"warning", "bg", 7, "warning text"},
	{"warning", "warning-soft", 7, "warning alert"},
	{"danger", "bg", 7, "error text"},
	{"danger", "danger-soft", 7, "error alert"},
	{"info", "bg", 7, "info text"},
	{"info", "info-soft", 7, "info alert"},
	// Block tones tint a whole surface, so body text has to hold on each.
	{"fg", "accent-soft", 7, "body text on an accent-toned block"},
	{"fg", "success-soft", 7, "body text on a success-toned block"},
	{"fg", "warning-soft", 7, "body text on a warning-toned block"},
	{"fg", "danger-soft", 7, "body text on a danger-toned block"},
	{"fg", "info-soft", 7, "body text on an info-toned block"},
	{"fg-muted", "accent-soft", 7, "secondary text on an accent-toned block"},
	{"fg-muted", "info-soft", 7, "secondary text on an info-toned block"},
	{"border-strong", "bg", 3, "control borders (1.4.11)"},
	{"focus", "bg", 3, "focus ring on page"},
	{"focus", "bg-muted", 3, "focus ring on muted surface"},
	// The ring never touches a control directly: focus-halo (the page colour)
	// sits between them, so the ring only needs contrast with the halo and
	// the page, never with the control's own colour (WCAG 2.4.13 adjacency).
	{"focus", "focus-halo", 3, "focus ring against its halo"},
	{"focus", "bg-raised", 3, "focus ring on cards"},
}

func TestPaletteMeetsAAAInBothThemes(t *testing.T) {
	src, err := os.ReadFile("../../design/tokens/tokens.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Color map[string]map[string]string `json:"color"`
	}
	if err := json.Unmarshal(src, &doc); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		for _, p := range pairs {
			fg, ok1 := doc.Color[p.fg][theme]
			bg, ok2 := doc.Color[p.bg][theme]
			if !ok1 || !ok2 {
				t.Errorf("%s: missing token %s or %s", theme, p.fg, p.bg)
				continue
			}
			ratio := contrast(fg, bg)
			if ratio < p.min {
				t.Errorf("%s: %s on %s is %.2f:1, needs %.0f:1 (%s)", theme, p.fg, p.bg, ratio, p.min, p.why)
			}
		}
	}
}

func TestEveryColourHasBothThemes(t *testing.T) {
	src, _ := os.ReadFile("../../design/tokens/tokens.json")
	var doc struct {
		Color map[string]map[string]string `json:"color"`
	}
	json.Unmarshal(src, &doc)
	for name, v := range doc.Color {
		if v["light"] == "" || v["dark"] == "" {
			t.Errorf("color.%s needs light and dark", name)
		}
	}
}

func luminance(hex string) float64 {
	c := func(i int) float64 {
		v, _ := strconv.ParseUint(hex[i:i+2], 16, 8)
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*c(1) + 0.7152*c(3) + 0.0722*c(5)
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// ExampleReport prints the matrix so `go test -v -run Example` documents the palette.
func Example_report() {
	src, _ := os.ReadFile("../../design/tokens/tokens.json")
	var doc struct {
		Color map[string]map[string]string `json:"color"`
	}
	json.Unmarshal(src, &doc)
	fmt.Println("light fg/bg:", math.Round(contrast(doc.Color["fg"]["light"], doc.Color["bg"]["light"])*10)/10 >= 7)
	// Output: light fg/bg: true
}
