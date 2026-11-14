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
	paidAgainMean = "A receipt, renewal notice, price change, free-trial or cancellation email for something paid again and again: a subscription, membership or plan. Not a one-off purchase."
)

// What sorting first said, changed since: a workspace whose words are
// still these is given the new ones; words the person wrote are theirs.
var paidAgainWas = map[string][2]string{
	paidAgainTag: {"A receipt, renewal notice or free-trial email for something paid again and again: a subscription, membership or plan. Not a one-off purchase.", paidAgainMean},
	"to do":      {"Something I have to do: pay, reply, book, bring, send, sign, renew, attend, buy or call. Not newsletters, adverts, receipts for what is paid, or notices that need nothing from me.", starterTags[0].means},
}

// SortPaidAgain adds the paid again tag and its suggestion to sorting,
// once, and brings what it added before up to date.
func (s *Service) SortPaidAgain() {
	if s.app.Store.Meta(paidAgainSet) != "" {
		s.paidAgainSince()
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
	s.app.Store.SetMeta(paidAgainSet, "changes one kept")
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
		"when": "changed", "what": "email", "only": []any{"tags=" + paidAgainTag}, "check": true, "examples": 10, "update": true})
}

// paidAgainSince gives sorting set up before them the words that keep a
// renewal off the list of things to do and take in cancellations, and
// has the suggestion change a subscription already kept rather than make
// a second. What the person changed is left as it is.
func (s *Service) paidAgainSince() {
	who := records.Who{Actor: "system", Via: "setting up sorting"}
	tags, _ := s.app.Store.List(records.TagType, store.ListOptions{})
	for _, t := range tags {
		name, _ := t.Fields["name"].(string)
		if was, ok := paidAgainWas[strings.ToLower(strings.TrimSpace(name))]; ok && t.Fields["means"] == was[0] {
			records.WriteAs(s.app.Store, who, "updated", records.TagType, t.ID, map[string]any{"means": was[1]})
		}
	}
	if s.app.Store.Meta(paidAgainSet) == "changes one kept" {
		return // done once: a person who turned it off since has it off
	}
	s.app.Store.SetMeta(paidAgainSet, "changes one kept")
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "suggest" && a.Fields["make"] == "subscription" {
			records.WriteAs(s.app.Store, who, "updated", records.ActionType, a.ID, map[string]any{"update": true})
		}
	}
}
