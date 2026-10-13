package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// A person is heard as the label and their name, once, and nothing of the
// dot. The name sits in its own bdi, so a name written right to left does
// not carry the words and punctuation around it along with it.
func TestPersonIsHeardByNameOnce(t *testing.T) {
	t.Parallel()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ label, name, heard string }{
		{"For", "Hana", "For Hana"},
		{"", "Hana", "Hana"},
		{"For", "مريم", "For مريم"},
	} {
		out, err := reg.Render("person", map[string]any{"label": c.label, "name": c.name, "colour": 2})
		if err != nil {
			t.Fatal(err)
		}
		doc, err := htmltest.Parse(string(out))
		if err != nil {
			t.Fatal(err)
		}
		chip := doc.WithAttr("data-component", "person")[0]
		if got := strings.Join(strings.Fields(htmltest.VisibleText(chip)), " "); got != c.heard {
			t.Errorf("heard %q, want %q", got, c.heard)
		}
		bdi := doc.Elements("bdi")
		if len(bdi) != 1 || htmltest.Text(bdi[0]) != c.name {
			t.Errorf("the name alone, %q, is isolated in a bdi:\n%s", c.name, out)
		}
		dots := doc.WithAttr("class", "sw-person__dot")
		if len(dots) != 1 {
			t.Fatalf("one dot:\n%s", out)
		}
		if v, _ := htmltest.Attr(dots[0], "aria-hidden"); v != "true" {
			t.Errorf("the dot is hidden from screen readers:\n%s", out)
		}
	}
}
