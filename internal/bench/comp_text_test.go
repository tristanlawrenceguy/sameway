package bench

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// words are the blocks made of words and structure, built from a line in
// the prompt and, when the props do not fit, a refusal that gives the
// schema: a heading and a paragraph, a card that leads somewhere, a
// warning, a picture with its description, a part that opens and closes.
func words() []request {
	return []request{
		{name: "ct-heading-text", say: "On this page add a heading that says Garden plan, and under it a paragraph: We plant the beans in May.",
			check: func(a *app.App, _ string) string {
				return all(want(propHas(a, "heading", "text", "garden plan") || propHas(a, "text", "content", "garden plan"), "no heading says Garden plan"),
					want(propHas(a, "text", "content", "beans in may"), "no text block holds the paragraph"))
			}},
		{name: "ct-card-link", say: "Add a card to this page titled Seed catalogue that links to https://example.org/seeds",
			check: func(a *app.App, _ string) string {
				for _, p := range everyProps(a, "card") {
					if low(p["title"]) == "seed catalogue" {
						return want(strings.Contains(low(p["href"]), "example.org/seeds"), "the card does not link to the catalogue: "+jsonOf(p))
					}
				}
				return "no card titled Seed catalogue"
			}},
		{name: "ct-alert", say: "Put a warning on this page that says: The water is off on Friday.",
			check: func(a *app.App, _ string) string {
				for _, p := range everyProps(a, "alert") {
					if strings.Contains(low(p["message"])+low(p["title"]), "water is off") {
						return want(p["kind"] == "warning" || p["kind"] == "danger", "the alert is not a warning: "+jsonOf(p))
					}
				}
				return "no alert says the water is off"
			}},
		{name: "ct-image", say: "Show the picture at /files/fern.jpg on this page. It is the fern on the kitchen shelf with two new fronds.",
			check: func(a *app.App, _ string) string {
				for _, p := range everyProps(a, "image") {
					if strings.Contains(low(p["src"]), "/files/fern.jpg") {
						return want(strings.Contains(low(p["alt"]), "fern"), "the picture has no description: "+jsonOf(p))
					}
				}
				return "no image block shows the fern"
			}},
		{name: "ct-open-close", say: "Add a section to this page called Packing list that opens and closes, closed to start with. Inside it: Tent, stove, map.",
			check: func(a *app.App, _ string) string {
				for _, p := range everyProps(a, "disclosure") {
					if strings.Contains(low(p["label"]), "packing list") {
						return all(want(strings.Contains(low(p["slot"]), "stove"), "the section does not hold the list: "+jsonOf(p)),
							want(p["open"] != true, "the section starts open"))
					}
				}
				return "no disclosure called Packing list"
			}},
	}
}

// everyProps is the props of every block of a component.
func everyProps(a *app.App, component string) []map[string]any {
	var out []map[string]any
	for _, b := range list(a, records.BlockType) {
		if b.Fields["component"] != component {
			continue
		}
		p, _ := b.Fields["props"].(map[string]any)
		if s, ok := b.Fields["props"].(string); ok {
			json.Unmarshal([]byte(s), &p)
		}
		out = append(out, p)
	}
	return out
}

// propHas is whether a block of the component holds the words in a prop.
func propHas(a *app.App, component, prop, words string) bool {
	for _, p := range everyProps(a, component) {
		if strings.Contains(low(p[prop]), words) {
			return true
		}
	}
	return false
}

func low(v any) string {
	if v == nil {
		return ""
	}
	return strings.ToLower(fmt.Sprint(v))
}
