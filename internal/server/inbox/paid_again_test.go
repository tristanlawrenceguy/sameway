package inbox_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Sorting set up before it knew of them gains, once, a tag for what is paid
// again and again and an action suggesting a subscription from each.
func TestSortingGainsPaidAgainOnce(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	srv.SortPaidAgain()
	if tags, _ := a.Store.List(records.TagType, store.ListOptions{}); len(tags) != 0 {
		t.Fatal("without sorting set up, nothing is added")
	}
	srv.SetUpSorting()
	srv.SortPaidAgain()
	srv.SortPaidAgain()
	tags, _ := a.Store.List(records.TagType, store.ListOptions{})
	paid := 0
	for _, tg := range tags {
		if tg.Fields["name"] == "paid again" {
			paid++
		}
	}
	if paid != 1 {
		t.Errorf("the tag is added once, got %d", paid)
	}
	actions, _ := a.Store.List(records.ActionType, store.ListOptions{})
	suggest := 0
	for _, act := range actions {
		if act.Fields["kind"] == "classify" && act.Fields["what"] == "email" && !strings.Contains(strings.Join(records.StringList(act.Fields["tags"]), ","), "paid again") {
			t.Errorf("sorting email gives the tag: %v", act.Fields["tags"])
		}
		if act.Fields["make"] == "subscription" {
			suggest++
		}
	}
	if suggest != 1 {
		t.Errorf("one action suggests a subscription, got %d", suggest)
	}
}
