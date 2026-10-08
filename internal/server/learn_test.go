package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Asked for often enough, a part is offered for good, with the reason in
// one sentence: yes turns it on and can be undone; no is not asked again;
// and someone let in is not watched.
func TestAPartAskedForOftenIsOffered(t *testing.T) {
	a, h := newApp(t)
	var ids []string
	for _, title := range []string{"Call plumber", "Order compost", "Dig the pond", "Fix the gate"} {
		rec, _ := a.Store.Create("task", map[string]any{"title": title})
		ids = append(ids, rec.ID)
	}
	editor := records.Visitor{Name: "Bob", Login: "bob@example.com", Access: records.Edit}
	for _, id := range ids[:3] {
		as(t, h, editor, http.MethodGet, "/t/task/"+id+"?show=fields", "", "")
	}
	if ps, _ := a.Store.List(records.ProposalType, store.ListOptions{}); len(ps) != 0 {
		t.Fatal("someone let in is not watched")
	}
	get(t, h, "/t/task/"+ids[0]+"?show=fields")
	get(t, h, "/t/task/"+ids[1]+"?show=fields")
	third := get(t, h, "/t/task/"+ids[2]+"?show=fields").Body.String()
	ps, _ := a.Store.List(records.ProposalType, store.ListOptions{})
	if len(ps) != 1 {
		t.Fatalf("the third time, one offer: %d", len(ps))
	}
	if !strings.Contains(third, "Show every field on every page it fits?") || !strings.Contains(third, "You have opened every field on a task&#39;s page 3 times this week.") || strings.Contains(third, "This cannot be undone") {
		t.Errorf("the offer is on the page, with its reason\n%s", truncate(third))
	}
	get(t, h, "/t/task/"+ids[3]+"?show=fields")
	if ps, _ := a.Store.List(records.ProposalType, store.ListOptions{}); len(ps) != 1 {
		t.Errorf("asked once only: %d offers", len(ps))
	}
	res := postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil)
	if res.Code >= 400 || !strings.Contains(a.Workspace.Config.UI.Show, "fields") {
		t.Fatalf("yes turns it on for the workspace: %d %q", res.Code, a.Workspace.Config.UI.Show)
	}
	entries, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
	for _, e := range entries {
		if e.Fields["action"] == "set" && e.Fields["target"] == "ui.show" {
			if !a.Records.Undoable(e) {
				t.Error("and it can be undone")
			}
			return
		}
	}
	t.Error("the yes is in the log")
}
