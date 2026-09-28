package render_test

// A button is heard as a toggle only when it is one: aria-pressed is there
// when pressed is given, true or false, and not on any other button.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

func TestOnlyAToggleButtonSaysPressed(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		props map[string]any
		want  string
	}{
		{map[string]any{"label": "Save note"}, ""},
		{map[string]any{"label": "Send", "type": "submit"}, ""},
		{map[string]any{"label": "Show done tasks", "pressed": true}, `aria-pressed="true"`},
		{map[string]any{"label": "Show done tasks", "pressed": false}, `aria-pressed="false"`},
	} {
		out, err := reg.Render("button", c.props)
		if err != nil {
			t.Fatalf("render %v: %v", c.props, err)
		}
		got := string(out)
		if c.want == "" && strings.Contains(got, "aria-pressed") {
			t.Errorf("%v: a button that is not a toggle should have no aria-pressed:\n%s", c.props, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%v: want %s:\n%s", c.props, c.want, got)
		}
	}
}
