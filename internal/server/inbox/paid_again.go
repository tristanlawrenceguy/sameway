package inbox

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What is paid again and again comes in as receipts and renewal notices,
// which sorting otherwise leaves as nothing to do. A tag for them, and an
// action that suggests a subscription from each, are added to sorting
// once: to a new setup, and to one made before them. A person who removed
// sorting is not given them back, and what they change after is theirs.

const (
	paidAgainSet  = "sorting:paid-again"
	paidAgainTag  = "paid again"
	paidAgainMean = "A receipt, renewal notice or free-trial email for something paid again and again: a subscription, membership or plan. Not a one-off purchase."
)

// SortPaidAgain adds the paid again tag and its suggestion to sorting, once.
func (s *Service) SortPaidAgain() {
	if s.app.Store.Meta(paidAgainSet) != "" {
		return
	}
	for _, t := range []string{records.TagType, records.ActionType, "email", "subscription"} {
		if _, ok := s.app.Types.Get(t); !ok {
			return
		}
	}
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	var sorting *store.Record
	for _, a := range actions {
		if a.Fields["kind"] == "classify" && a.Fields["what"] == "email" {
			sorting = a
		}
	}
	if sorting == nil {
		return // not set up yet, or removed: SetUpSorting comes first
	}
	s.app.Store.SetMeta(paidAgainSet, "yes")
	who := records.Who{Actor: "system", Via: "setting up sorting"}
	have := false
	for _, n := range s.tagNames() {
		have = have || strings.EqualFold(n, paidAgainTag)
	}
	if !have {
		records.WriteAs(s.app.Store, who, "created", records.TagType, "", map[string]any{"name": paidAgainTag, "means": paidAgainMean})
	}
	tags := records.StringList(sorting.Fields["tags"])
	if len(tags) > 0 { // an empty list sorts by every tag, this one too
		var given []any
		for _, t := range tags {
			if strings.EqualFold(t, paidAgainTag) {
				given = nil
				break
			}
			given = append(given, t)
		}
		if given != nil {
			records.WriteAs(s.app.Store, who, "updated", records.ActionType, sorting.ID, map[string]any{"tags": append(given, paidAgainTag)})
		}
	}
	records.WriteAs(s.app.Store, who, "created", records.ActionType, "", map[string]any{
		"title": "Suggest a subscription from an email paid again", "kind": "suggest", "make": "subscription",
		"when": "changed", "what": "email", "only": []any{"tags=" + paidAgainTag}, "check": true, "examples": 10})
}
