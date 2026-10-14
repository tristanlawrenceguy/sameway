package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// What a person writes takes its direction from its own first letter: a
// title in Hebrew or Arabic reads right to left with its ? at its end, and
// does not pull the words beside it out of order (design/foundations/text.md).
func TestWrittenTextTakesItsOwnDirection(t *testing.T) {
	t.Parallel()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	for name, props := range map[string]map[string]any{
		"link":       {"label": "מה קורה?", "href": "/t/note/a"},
		"heading":    {"text": "ما الجديد؟", "level": 2},
		"record":     {"type": "note", "record": "a", "title": "מה קורה?", "detail": "brief", "level": 3},
		"text-field": {"label": "Title", "name": "title", "value": "מה קורה?"},
		"textarea":   {"label": "Words", "name": "words", "rows": 3, "value": "ما الجديد؟"},
	} {
		out, err := reg.Render(name, props)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(string(out), `dir="auto"`) {
			t.Errorf("%s does not take its direction from its text:\n%s", name, out)
		}
	}
}
