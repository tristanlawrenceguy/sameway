package inbox

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What is paid again and again, and what is bought once, comes in as
// receipts and notices, by email or as a photo shared from the phone,
// which sorting otherwise leaves as nothing to do. For each, a tag and
// actions suggesting a record of it (a subscription, a purchase) are
// added to sorting, in steps, each once: to a new setup, and to one made
// before them. A person who removed sorting is not given them back, and
// what they change after is theirs.

// kept is one kind of record sorting keeps from what comes in.
type kept struct {
	set, tag, means, make, what string
}

var keptFromMail = []kept{
	{set: "sorting:paid-again", tag: "paid again", make: "subscription", what: "a subscription",
		means: "A receipt, renewal notice, price change, free-trial or cancellation email for something paid again and again: a subscription, membership or plan. Not a one-off purchase."},
	{set: "sorting:bought", tag: "bought", make: "purchase", what: "a purchase",
		means: "A receipt, order confirmation, delivery or return for something bought once: a thing or a repair, not a subscription, a bill or a booking."},
}

// The steps, as each kind's note says how far it came: its tag and its
// suggestion from email; that suggestion changing one already kept; the
// same from what is shared.
const (
	stepEmail  = "yes"
	stepShared = "from what is shared too"
)

// What sorting first said, changed since: a workspace whose words are
// still these is given the new ones; words the person wrote are theirs.
var sortingWas = map[string][2]string{
	fold.Key("paid again"): {"A receipt, renewal notice or free-trial email for something paid again and again: a subscription, membership or plan. Not a one-off purchase.", keptFromMail[0].means},
	fold.Key("to do"):      {"Something I have to do: pay, reply, book, bring, send, sign, renew, attend, buy or call. Not newsletters, adverts, receipts for what is paid, or notices that need nothing from me.", starterTags[0].means},
}

// SortPaidAgain brings sorting up to date with what it keeps from what
// comes in: each kind's steps once, and Sameway's own words where they
// are still its own. Reading email runs it.
func (s *Service) SortPaidAgain() {
	s.sortingWords()
	s.tasksChangeOneKept() // bills.go
	for _, k := range keptFromMail {
		s.sortKept(k)
	}
}

// sortingAction is the classify action for email, or for what is shared,
// that SetUpSorting made, or nil when it was removed or never made.
func (s *Service) sortingAction(what string, shared bool) *store.Record {
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "classify" && a.Fields["what"] == what && strings.Contains(strings.Join(records.StringList(a.Fields["only"]), ","), "tags=shared") == shared {
			return a
		}
	}
	return nil
}

// sortKept takes one kind through the steps it has not had.
func (s *Service) sortKept(k kept) {
	step := s.app.Store.Meta(k.set)
	if step == stepShared {
		return
	}
	for _, t := range []string{records.TagType, records.ActionType, "email", "note", k.make} {
		if _, ok := s.app.Types.Get(t); !ok {
			return
		}
	}
	who := records.Who{Actor: "system", Via: "setting up sorting"}
	switch step {
	case "":
		email := s.sortingAction("email", false)
		if email == nil {
			return // not set up yet, or removed: SetUpSorting comes first
		}
		s.keptTag(who, k)
		s.giveTag(who, email, k.tag)
		s.keptSuggestion(who, k, "email")
	case stepEmail: // made before a suggestion changed one kept
		s.changeOneKept(who, k.make, "email")
	}
	// A paper receipt photographed and shared from the phone is sorted as
	// an email is.
	if shared := s.sortingAction("note", true); shared != nil {
		s.giveTag(who, shared, k.tag)
		s.keptSuggestion(who, k, "note")
	}
	s.app.Store.SetMeta(k.set, stepShared)
}

// keptTag makes a kind's tag, unless the person has one by its name.
func (s *Service) keptTag(who records.Who, k kept) {
	for _, n := range s.tagNames() {
		if fold.Key(n) == fold.Key(k.tag) {
			return
		}
	}
	records.WriteAs(s.app.Store, who, "created", records.TagType, "", map[string]any{"name": k.tag, "means": k.means})
}

// keptSuggestion is the action suggesting a kind's record from an email,
// or from what is shared, with its tag.
func (s *Service) keptSuggestion(who records.Who, k kept, what string) {
	only, from := []any{"tags=" + k.tag}, "an email"
	if what == "note" {
		only, from = []any{"tags=shared", "tags=" + k.tag}, "something shared"
	}
	records.WriteAs(s.app.Store, who, "created", records.ActionType, "", map[string]any{
		"title": "Suggest " + k.what + " from " + from + " " + k.tag, "kind": "suggest", "make": k.make,
		"when": "changed", "what": what, "only": only, "check": true, "examples": 10, "update": true})
}

// changeOneKept has the suggestions of a kind from what comes in change
// one already kept.
func (s *Service) changeOneKept(who records.Who, make, what string) {
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "suggest" && a.Fields["make"] == make && a.Fields["what"] == what {
			records.WriteAs(s.app.Store, who, "updated", records.ActionType, a.ID, map[string]any{"update": true})
		}
	}
}

// giveTag has a classify action that names its tags give this one too;
// one naming none gives every tag already.
func (s *Service) giveTag(who records.Who, action *store.Record, tag string) {
	tags := records.StringList(action.Fields["tags"])
	if len(tags) == 0 {
		return
	}
	var given []any
	for _, t := range tags {
		if fold.Key(t) == fold.Key(tag) {
			return
		}
		given = append(given, t)
	}
	records.WriteAs(s.app.Store, who, "updated", records.ActionType, action.ID, map[string]any{"tags": append(given, tag)})
}

// sortingWords gives sorting set up before them the words that keep a
// renewal off the list of things to do and take in cancellations, where
// they are still Sameway's.
func (s *Service) sortingWords() {
	who := records.Who{Actor: "system", Via: "setting up sorting"}
	tags, _ := s.app.Store.List(records.TagType, store.ListOptions{})
	for _, t := range tags {
		name, _ := t.Fields["name"].(string)
		if was, ok := sortingWas[fold.Key(name)]; ok && t.Fields["means"] == was[0] {
			records.WriteAs(s.app.Store, who, "updated", records.TagType, t.ID, map[string]any{"means": was[1]})
		}
	}
}
