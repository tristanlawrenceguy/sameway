package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// written_by says who wrote a record's words in plain words, from the
// activity log: the owner, the owner's token, another person by name, an
// import, the assistant; and "not known" when the log is silent.
// Only what came from beyond the owner and their assistant is outside.
func TestWrittenBySaysWhoInPlainWords(t *testing.T) {
	svc := newFullService(t)
	svc.Owner = records.Visitor{Login: "me@example.com"}
	note := func(title string) string {
		rec, err := svc.Store.Create("note", map[string]any{"title": title})
		if err != nil {
			t.Fatal(err)
		}
		return rec.ID
	}
	mine, token, bobs, imported, assisted, silent := note("Mine"), note("Token"), note("Bob's"), note("Mail"), note("Assisted"), note("Old")
	log := func(actor, id string, c records.Change) {
		c.Component, c.ID = "note", id
		if c.Action == "" {
			c.Action = "created"
		}
		records.Record(svc.Store, actor, c)
	}
	log("human", mine, records.Change{ByLogin: "me@example.com", Via: "phone"})
	log("human", token, records.Change{Via: records.ThroughAPI})
	log("human", bobs, records.Change{By: "Bob", ByLogin: "bob@example.com"})
	log("human", assisted, records.Change{})
	log("assistant", assisted, records.Change{Action: "updated"})
	records.Record(svc.Store, "human", records.Change{Action: "imported", Component: "note", Detail: "1 notes from inbox.mbox", Before: records.Imported("note", []string{imported})})

	w := svc.Writers()
	for id, want := range map[string]records.Writer{
		mine:     {Words: "the owner"},
		token:    {Words: "the owner's API token"},
		bobs:     {Words: "Bob, another person", Outside: true},
		imported: {Words: "an import from inbox.mbox", Outside: true},
		assisted: {Words: "the owner, then the assistant"},
		silent:   {Words: "not known: nothing in the activity log says"},
	} {
		if got := w.OfID("note", id); got != want {
			t.Errorf("%s: got %+v, want %+v", id, got, want)
		}
	}
	if got := svc.For(records.Visitor{Access: records.Public}).Writers().OfID("note", bobs).Words; got != "another person" {
		t.Errorf("the internet is not told another person's name: %q", got)
	}
}
