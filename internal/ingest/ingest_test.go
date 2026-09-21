package ingest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func starter(t *testing.T) *store.Store {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// A CSV is read with whatever delimiter it uses, its columns matched to a
// type by name and by what they plainly are, and one record made per row;
// a person already here by email is not made twice.
func TestACSVBecomesPeople(t *testing.T) {
	st := starter(t)
	person, _ := st.Types().Get("person")
	tb, err := ingest.Read("contacts.csv", []byte("Full name;E-mail;Mobile;Company;Job title;Tags\nSandra Lee;sandra@example.com;+44 7700 900123;Acme;Ops;client, north\nTom Ash;tom@example.com;;Bee Ltd;;\n;;;;;\n"))
	if err != nil {
		t.Fatal(err)
	}
	if tb.Source != "csv" || len(tb.Rows) != 2 || tb.Columns[1] != "E-mail" {
		t.Fatalf("a semicolon CSV with a blank line reads as two rows, got %+v", tb)
	}
	m := ingest.Guess(person, tb.Columns)
	want := map[string]string{"Full name": "name", "E-mail": "email", "Mobile": "phone", "Company": "organisation", "Job title": "role", "Tags": "tags"}
	for col, field := range want {
		if m[col] != field {
			t.Errorf("column %q should feed %s, got %q", col, field, m[col])
		}
	}
	r := ingest.Import(st, person, tb, m)
	if r.Made != 2 || r.Skipped != 0 {
		t.Fatalf("two people made: %s", r)
	}
	sandra, _ := st.Get("person", r.IDs[0])
	tags, _ := sandra.Fields["tags"].([]any)
	if sandra.Fields["name"] != "Sandra Lee" || sandra.Fields["organisation"] != "Acme" || len(tags) != 2 {
		t.Errorf("the fields land, tags as a list: %v", sandra.Fields)
	}
	again := ingest.Import(st, person, tb, m)
	if again.Made != 0 || again.Skipped != 2 || !strings.Contains(again.String(), "already here") {
		t.Errorf("the same file again makes nobody twice: %s", again)
	}
}

// A vCard is contacts too: the formatted name or the name parts, the
// first email and phone, the organisation, the title, the note, folded
// lines unfolded.
func TestAVCardBecomesPeople(t *testing.T) {
	vcf := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Lee;Sandra;;;\r\nFN:Sandra Lee\r\nORG:Acme;Ops\r\nTITLE:Head of Ops\r\nEMAIL;TYPE=INTERNET:sandra@example.com\r\nTEL;TYPE=CELL:+44 7700 900123\r\nNOTE:Met at the\r\n  garden show\\, likes tea\r\nEND:VCARD\r\nBEGIN:VCARD\r\nVERSION:2.1\r\nN:Ash;Tom\r\nTEL:0117 496 0000\r\nEND:VCARD\r\n"
	tb, err := ingest.Read("contacts.vcf", []byte(vcf))
	if err != nil {
		t.Fatal(err)
	}
	if len(tb.Rows) != 2 || tb.Rows[0]["name"] != "Sandra Lee" || tb.Rows[0]["organisation"] != "Acme Ops" || tb.Rows[0]["notes"] != "Met at the garden show, likes tea" || tb.Rows[1]["name"] != "Tom Ash" {
		t.Errorf("two contacts, read whole: %+v", tb.Rows)
	}
	st := starter(t)
	person, _ := st.Types().Get("person")
	r := ingest.Import(st, person, tb, ingest.Guess(person, tb.Columns))
	if r.Made != 2 {
		t.Errorf("both made: %s", r)
	}
}

// A mailbox becomes interactions: one per message, an email, when it was
// sent, its subject and the start of its text, each linked to the person
// it was from, made when new and found by email when not.
func TestAMailboxBecomesInteractionsWithPeople(t *testing.T) {
	mbox, _ := os.ReadFile(filepath.Join("testdata", "two.mbox"))
	st := starter(t)
	st.Create("person", map[string]any{"name": "Sandra Lee", "email": "Sandra@example.com"})
	tb, err := ingest.Read("mail.mbox", mbox)
	if err != nil {
		t.Fatal(err)
	}
	if len(tb.Rows) != 2 || tb.Rows[0]["summary"] != "September hours" || tb.Rows[0]["from_email"] != "sandra@example.com" || !strings.HasPrefix(tb.Rows[0]["notes"], "Hi, here are") || tb.Rows[1]["from_name"] != "Tom Ash" {
		t.Fatalf("two messages, read whole: %+v", tb.Rows)
	}
	inter, _ := st.Types().Get("interaction")
	m := ingest.Guess(inter, tb.Columns)
	if m["kind"] != "kind" || m["at"] != "at" || m["summary"] != "summary" || m["notes"] != "notes" {
		t.Errorf("the mailbox columns feed the interaction fields: %v", m)
	}
	r := ingest.Import(st, inter, tb, m)
	if r.Made != 2 || r.Linked != 2 {
		t.Fatalf("two interactions, both with a person: %s", r)
	}
	people, _ := st.List("person", store.ListOptions{})
	if len(people) != 2 {
		t.Errorf("Sandra was found by email and Tom was made, so two people, got %d", len(people))
	}
	first, _ := st.Get("interaction", r.IDs[0])
	if first.Fields["kind"] != "email" || first.Fields["person"] == "" || !strings.HasPrefix(first.Fields["at"].(string), "2026-09-0") {
		t.Errorf("an email interaction at its time with its person: %v", first.Fields)
	}
}

// A call log is a CSV of numbers and times; each row becomes a call with
// the person the number belongs to.
func TestACallLogBecomesCalls(t *testing.T) {
	st := starter(t)
	st.Create("person", map[string]any{"name": "Sandra Lee", "phone": "07700 900123"})
	tb, _ := ingest.Read("calls.csv", []byte("Number,Name,Date,Duration,Type\n+447700900123,Sandra,2026-09-18 14:02,00:05:12,Incoming\n01174960000,,2026-09-19 09:30,00:01:00,Outgoing\n"))
	inter, _ := st.Types().Get("interaction")
	m := ingest.Guess(inter, tb.Columns)
	m["Type"] = ""
	r := ingest.Import(st, inter, tb, m)
	if r.Made != 2 || r.Linked != 2 {
		t.Fatalf("two calls, each with someone: %s", r)
	}
	first, _ := st.Get("interaction", r.IDs[0])
	people, _ := st.List("person", store.ListOptions{})
	sandra := ""
	for _, p := range people {
		if p.Fields["name"] == "Sandra Lee" {
			sandra = p.ID
		}
	}
	if len(people) != 2 || first.Fields["person"] != sandra {
		t.Errorf("the number with spaces and the number with the country code meet, and the unknown number is a new person: %v among %d people", first.Fields, len(people))
	}
	if first.Fields["summary"] == nil {
		t.Error("a call with no subject still has a title")
	}
}
