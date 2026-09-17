package query_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func tasks(t *testing.T) (*store.Store, *schema.Type) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "task.yaml"), []byte("name: task\ntitle: title\nfields:\n  title: {type: string, required: true}\n  done: {type: bool, default: false}\n  due: {type: datetime}\n  effort: {type: int}\n  tags: {type: list, of: string}\n  notes: {type: markdown}\n"), 0o644)
	types, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	typ, _ := types.Get("task")
	return st, typ
}

// The same few words pick records by state, by date relative to today, by
// words in a field, by a tag, and by what is empty; the order and the
// limit are part of the ask.
func TestOneGrammarPicksRecords(t *testing.T) {
	st, typ := tasks(t)
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	day := func(d int) string { return now.AddDate(0, 0, d).Format(time.RFC3339) }
	st.Create("task", map[string]any{"title": "Dig the pond", "due": day(-1), "effort": 3, "tags": []string{"garden"}})
	st.Create("task", map[string]any{"title": "Order compost", "due": day(2), "effort": 1, "tags": []string{"garden", "shopping"}, "notes": "From the usual place."})
	st.Create("task", map[string]any{"title": "Call the dentist", "due": day(10), "effort": 1, "done": true})
	st.Create("task", map[string]any{"title": "Plant garlic", "effort": 2})

	titles := func(where []string, order string, limit int) string {
		t.Helper()
		recs, err := query.Filter(st, typ, where, order, limit, now)
		if err != nil {
			t.Fatalf("%v: %v", where, err)
		}
		var out []string
		for _, r := range recs {
			out = append(out, r.Fields["title"].(string))
		}
		return strings.Join(out, "|")
	}
	cases := []struct {
		where []string
		order string
		limit int
		want  string
	}{
		{[]string{"done=false", "due<today"}, "", 0, "Dig the pond"},
		{[]string{"due<+7d", "due!="}, "due", 0, "Dig the pond|Order compost"},
		{[]string{"due>today"}, "-due", 0, "Call the dentist|Order compost"},
		{[]string{"due="}, "", 0, "Plant garlic"},
		{[]string{"title~POND"}, "", 0, "Dig the pond"},
		{[]string{"tags=shopping"}, "", 0, "Order compost"},
		{[]string{"tags~gar"}, "title", 0, "Dig the pond|Order compost"},
		{[]string{"effort>=2"}, "-effort", 1, "Dig the pond"},
		{[]string{"notes~usual"}, "", 0, "Order compost"},
		{[]string{"done!=true", "effort<=1"}, "", 0, "Order compost"},
		{[]string{"due=" + now.AddDate(0, 0, 2).Format("2006-01-02")}, "", 0, "Order compost"},
		{nil, "title", 2, "Call the dentist|Dig the pond"},
	}
	for _, c := range cases {
		if got := titles(c.where, c.order, c.limit); got != c.want {
			t.Errorf("where %v order %q limit %d: got %q, want %q", c.where, c.order, c.limit, got, c.want)
		}
	}
}

// A wrong field or a shapeless condition is an error that names what the
// type has, so whoever typed it can fix it without guessing.
func TestAWrongConditionSaysWhatItCouldBe(t *testing.T) {
	st, typ := tasks(t)
	if _, err := query.Filter(st, typ, []string{"owner=me"}, "", 0, time.Now()); err == nil || !strings.Contains(err.Error(), `no field "owner"`) || !strings.Contains(err.Error(), "title, done, due") {
		t.Errorf("an unknown field lists the fields: %v", err)
	}
	if _, err := query.Filter(st, typ, []string{"just words"}, "", 0, time.Now()); err == nil || !strings.Contains(err.Error(), "not a condition") || !strings.Contains(err.Error(), "today") {
		t.Errorf("a shapeless condition shows the grammar: %v", err)
	}
	if _, err := query.Filter(st, typ, nil, "-owner", 0, time.Now()); err == nil || !strings.Contains(err.Error(), "order by") {
		t.Errorf("an unknown order field is refused: %v", err)
	}
}

// A ref is picked by the id it holds or by the title of what it points at.
func TestARefMatchesByIdOrByTitle(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "project.yaml"), []byte("name: project\ntitle: title\nfields:\n  title: {type: string, required: true}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "task.yaml"), []byte("name: task\ntitle: title\nfields:\n  title: {type: string, required: true}\n  project: {type: ref, to: project}\n"), 0o644)
	types, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	garden, _ := st.Create("project", map[string]any{"title": "Garden"})
	house, _ := st.Create("project", map[string]any{"title": "House"})
	st.Create("task", map[string]any{"title": "Dig the pond", "project": garden.ID})
	st.Create("task", map[string]any{"title": "Paint the hall", "project": house.ID})
	st.Create("task", map[string]any{"title": "Call the dentist"})
	typ, _ := types.Get("task")
	for _, c := range []struct {
		where []string
		want  string
	}{
		{[]string{"project=" + garden.ID}, "Dig the pond"},
		{[]string{"project~gard"}, "Dig the pond"},
		{[]string{"project=House"}, "Paint the hall"},
		{[]string{"project="}, "Call the dentist"},
		{[]string{"project!=" + garden.ID}, "Call the dentist|Paint the hall"},
	} {
		recs, err := query.Filter(st, typ, c.where, "title", 0, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, r := range recs {
			got = append(got, r.Fields["title"].(string))
		}
		if strings.Join(got, "|") != c.want {
			t.Errorf("%v: got %q, want %q", c.where, strings.Join(got, "|"), c.want)
		}
	}
}
