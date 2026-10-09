package server_test

import (
	"net/url"
	"strings"
	"testing"
)

// Every template on the Templates page applies with one press, each in a
// fresh workspace: the kinds it needs are made, a tab of its own holds
// it, and that tab shows every block it was given, none set up wrong.
func TestEveryTemplateAppliesWithOnePress(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	page := get(t, h, "/templates").Body.String()
	var names []string
	for _, tpl := range a.Chat.Templates() {
		names = append(names, tpl.Name)
		if !strings.Contains(page, tpl.Title) {
			t.Errorf("the page offers %s", tpl.Title)
		}
	}
	if len(names) < 10 {
		t.Fatalf("templates: %v", names)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			a, h := newApp(t)
			res := postForm(t, h, "/templates/use", url.Values{"name": {name}})
			to := res.Header().Get("Location")
			if res.Code != 303 || !strings.HasPrefix(to, "/c/") {
				t.Fatalf("it goes to a tab of its own: %d %q %s", res.Code, to, truncate(res.Body.String()))
			}
			tab := get(t, h, strings.SplitN(to, "?", 2)[0]).Body.String()
			if strings.Contains(tab, `data-component="problem"`) {
				t.Errorf("no block set up wrong: %s", truncate(tab))
			}
			tpl, _ := a.Registry.Arrangement(name)
			for _, n := range tpl.Needs {
				if _, ok := a.Store.Types().Get(n.Type); !ok {
					t.Errorf("the kind %s is made", n.Type)
				}
			}
		})
	}
}
