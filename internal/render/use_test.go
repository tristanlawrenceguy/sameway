package render_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// Every built-in component says when a person is served by it and when
// they are not. That is the thought behind the design system written down
// once, and it is what the assistant builds from; a component without it
// would be a shape the model has to guess the use of.
func TestEveryComponentSaysWhenItServesAPerson(t *testing.T) {
	reg := builtins(t)
	for _, c := range reg.Components() {
		u := c.Manifest.Use
		if u == nil || len(u.When) < 20 || len(u.Not) < 10 {
			t.Errorf("%s: manifest needs use.when and use.not, the thought behind it, got %+v", c.Manifest.Name, u)
		}
	}
	_ = render.Registry{}
}
