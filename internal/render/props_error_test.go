package render_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// Props that do not fit are one error with two readers: a person gets a
// line per problem by the prop's readable name and no schema words; the
// one fixing the call gets each problem whole, and every prop there is.
func TestPropsThatDoNotFitAreKeptWhole(t *testing.T) {
	t.Parallel()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	c, _ := reg.Get("button")
	_, err := c.Validate(map[string]any{"text": "Save", "variant": "loud"})
	var pe *render.PropsError
	if !errors.As(err, &pe) {
		t.Fatalf("expected a *render.PropsError, got %T %v", err, err)
	}
	for _, raw := range []string{"additional properties", "not allowed"} {
		if strings.Contains(err.Error(), raw) {
			t.Errorf("a person reads no schema words, got %q", err.Error())
		}
	}
	if !strings.Contains(err.Error(), "Text: something I don't recognise") || !strings.Contains(err.Error(), "Label") {
		t.Errorf("a person reads each problem by its name, got %q", err.Error())
	}
	got := map[string]render.PropProblem{}
	for _, p := range pe.Problems {
		got[p.Kind+" "+p.Prop] = p
	}
	if _, ok := got["unknown text"]; !ok {
		t.Errorf("text is an unknown prop: %+v", pe.Problems)
	}
	if _, ok := got["missing label"]; !ok {
		t.Errorf("label is missing: %+v", pe.Problems)
	}
	if p := got["enum variant"]; strings.Join(p.Allowed, ",") != `"primary","secondary","danger","quiet"` || p.Got != `"loud"` {
		t.Errorf("variant says what it may be and what it was: %+v", p)
	}
	if pe.Component != "button" || strings.Join(pe.Required, ",") != "label" || !strings.Contains(strings.Join(pe.Props, ","), "label,name") {
		t.Errorf("the error names the component, what it requires and every prop: %+v", pe)
	}
}
