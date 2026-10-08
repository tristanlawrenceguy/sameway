package records

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// watches refuses an action that could never run on its own: what names
// no kind of record, or only holds a condition that kind cannot meet. A
// model asked to act when a task is done wrote what "status=done" three
// times in three, and the action was kept and never ran.
func watches(st *store.Store, fields map[string]any) error {
	what, _ := fields["what"].(string)
	if strings.TrimSpace(what) == "" {
		return nil
	}
	t, ok := st.Types().Get(what)
	if !ok {
		var kinds []string
		for _, n := range st.Types().Names() {
			if ct, _ := st.Types().Get(n); ct != nil && ct.Content() {
				kinds = append(kinds, n)
			}
		}
		why := fmt.Sprintf("what: %q is not a kind of record; what is the kind the action watches (%s)", what, strings.Join(kinds, ", "))
		if strings.ContainsAny(what, "=<>") {
			why += fmt.Sprintf(", and a condition goes in only, as [%q]", what)
		}
		return fmt.Errorf("%s", why)
	}
	if _, err := query.ParseAll(t, StringList(fields["only"])); err != nil {
		return fmt.Errorf("only: %v", err)
	}
	return nil
}

// StringList is a stored list of words as strings, each trimmed, with
// the empty ones and anything not a string left out.
func StringList(v any) []string {
	var out []string
	switch l := v.(type) {
	case []any:
		for _, x := range l {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	case []string:
		out = l
	}
	return out
}
