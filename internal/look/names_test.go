package look_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// Two controls of one kind and name within a landmark, and links of one
// name going to different places, are what an agent's getByRole cannot
// tell apart; look says so. The same name in two landmarks, links of one
// name to one place, radios and what is hidden are not.
func TestLookSaysWhatCannotBeToldApart(t *testing.T) {
	src := `<!doctype html><html><head><title>Tasks</title></head><body><main><h1>Tasks</h1>
<ol><li><form><label><input type="checkbox" name="d"> Done Call plumber</label></form><a href="/t/task/a">Call plumber</a></li>
<li><form><label><input type="checkbox" name="d"> Done Call plumber</label></form><a href="/t/task/b">Call plumber</a></li></ol>
<a href="/t/task">See the list</a><a href="/t/task">See the list</a>
<fieldset><legend>Status</legend><label><input type="radio" name="s"> Yes</label></fieldset><fieldset><legend>Pinned</legend><label><input type="radio" name="p"> Yes</label></fieldset>
<div hidden><button>Save</button></div><button>Save</button>
<section aria-label="Up next"><button>Remove</button></section><section aria-label="Notes"><button>Remove</button></section>
</main></body></html>`
	o, err := look.Page(src)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(o.Problems, "\n")
	for _, want := range []string{
		`same name: 2 checkbox controls named "done call plumber" in main`,
		`same name: links named "call plumber" go to 2 places: /t/task/a, /t/task/b`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("look should say %q, said:\n%s", want, got)
		}
	}
	if len(o.Problems) != 2 {
		t.Errorf("only those two, not links to one place, radios, hidden controls or one name in two regions:\n%s", got)
	}
}
