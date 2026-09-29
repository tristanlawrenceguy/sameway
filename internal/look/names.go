package look

import (
	"fmt"
	"sort"
	"strings"
)

// sameNames is the controls on a page that a person moving by controls,
// or an agent finding one by its role and name, cannot tell apart: two of
// one kind with one name within one landmark, and links of one name that
// go to different places anywhere on the page (WCAG 2.4.6, 2.4.9). What is
// hidden is not met, and radios are named by their choices, which repeat
// from one question to the next by design. Each problem starts "same
// name", so a page that shows examples side by side on purpose can tell
// them from the rest.
func sameNames(o *Outline, scopes []int) []string {
	type key struct {
		scope      int
		kind, name string
	}
	count := map[key]int{}
	var order []key
	places := map[string]map[string]bool{}
	var links []string
	for i, c := range o.Controls {
		name := strings.ToLower(strings.Join(strings.Fields(c.Name), " "))
		if c.Hidden || name == "" || c.Kind == "radio" {
			continue
		}
		if c.Kind == "link" {
			if places[name] == nil {
				places[name] = map[string]bool{}
				links = append(links, name)
			}
			places[name][c.Href] = true
			continue
		}
		k := key{scopes[i], c.Kind, name}
		if count[k] == 0 {
			order = append(order, k)
		}
		count[k]++
	}
	var out []string
	for _, k := range order {
		if n := count[k]; n > 1 {
			where := "outside any landmark"
			if k.scope >= 0 && k.scope < len(o.Landmarks) {
				l := o.Landmarks[k.scope]
				where = "in " + strings.TrimSpace(l.Role+" "+l.Label)
			}
			out = append(out, fmt.Sprintf("same name: %d %s controls named %q %s", n, k.kind, k.name, where))
		}
	}
	for _, name := range links {
		if set := places[name]; len(set) > 1 {
			hrefs := make([]string, 0, len(set))
			for h := range set {
				hrefs = append(hrefs, h)
			}
			sort.Strings(hrefs)
			out = append(out, fmt.Sprintf("same name: links named %q go to %d places: %s", name, len(set), strings.Join(hrefs, ", ")))
		}
	}
	return out
}
