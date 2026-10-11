package server_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
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

// A template says before what its tab will show, in the order a person
// reads an index; after, what it made, with one Undo that takes the whole
// tab back; and a second use names the tab of its own, not a count.
func TestATemplateIsSaidBeforeAndAfterAndUndoneInOne(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	page := get(t, h, "/templates").Body.String()
	if !strings.Contains(page, "Its tab shows: Still to do, Dates and Done.") {
		t.Errorf("a template says what its tab shows: %s", truncate(page))
	}
	last := -1
	for _, title := range []string{"Budget", "Clients", "Health", "Job search", "The move", "Reading", "Recipes", "Studies", "This week", "The trip", "Writing"} {
		at := strings.Index(page, "<h2>"+title+"</h2>")
		if at < last {
			t.Errorf("%s is out of order, as an index files it", title)
		}
		last = at
	}
	if strings.Contains(page, "the person") {
		t.Error("the page speaks to the person, not about them")
	}

	res := postForm(t, h, "/templates/use", url.Values{"name": {"jobs"}})
	tab := strings.TrimPrefix(strings.SplitN(res.Header().Get("Location"), "?", 2)[0], "/c/")
	told := after(t, h, res).Body.String()
	if !strings.Contains(told, "Job search is a tab of its own now. It keeps applications.") {
		t.Errorf("the outcome says what was made: %s", truncate(told))
	}
	undo := regexp.MustCompile(`action="(/activity/[^"]+/undo)"`).FindStringSubmatch(told)
	if undo == nil {
		t.Fatalf("the outcome carries Undo: %s", truncate(told))
	}

	again := get(t, h, "/templates").Body.String()
	if !strings.Contains(again, `href="/c/`+tab+`">Job search</a> tab already`) {
		t.Errorf("the page says the tab is there already: %s", truncate(again))
	}
	postForm(t, h, "/templates/use", url.Values{"name": {"jobs"}})
	var names []string
	for _, c := range a.Chat.Canvases() {
		names = append(names, c.Name)
	}
	if !strings.Contains(strings.Join(names, "|"), "|Job search 2") {
		t.Errorf("a second use is Job search 2: %v", names)
	}

	postForm(t, h, undo[1], url.Values{"from": {"/templates"}})
	if a.Chat.HasCanvas(tab) {
		t.Error("one Undo takes the tab back")
	}
	all, _ := a.Store.List(records.BlockType, store.ListOptions{})
	if left := records.OnCanvas(all, tab); len(left) > 0 {
		t.Errorf("and every block on it: %d left", len(left))
	}
	if _, ok := a.Store.Types().Get("application"); !ok {
		t.Error("the kind it made stays, with anything kept in it")
	}
}

// An agent presses the same button through look and reads, where it
// lands, the words a person is told.
func TestAnAgentUsesATemplateThroughLook(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	res := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"path": "/templates/use", "method": "POST", "form": map[string]any{"name": "move"}})
	body := res.Body.String()
	if !strings.Contains(body, `"landed": "/c/`) || !strings.Contains(body, "The move is a tab of its own now.") {
		t.Errorf("look lands on the tab and says what was made: %s", truncate(body))
	}
}
