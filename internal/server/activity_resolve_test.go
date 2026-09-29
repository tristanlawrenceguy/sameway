package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestUnresolvedNoteDetailShowsRawID seeds an activity entry where target is
// "note", detail equals the note's raw id, and the referenced note has a title.
// The current say() only resolves entry targets, so this pins the bug: the log
// still shows the raw ID in visible text until resolution is generalised to all
// content types. This covers acceptance item 1 (no raw IDs in headings).
func TestUnresolvedNoteDetailShowsRawID(t *testing.T) {
	a, h := newApp(t)

	// Seed a note with a title so we can later verify the resolved title appears.
	note, err := a.Store.Create("note", map[string]any{
		"title": "Test Validation",
		"body":  "a note created without going through create_record",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an activity entry where detail = raw note id (the bug condition).
	// This simulates what happens when a record is stored with no title and
	// recordTitle falls back to the id.
	chat.Record(a.Store, "assistant", chat.Change{
		Action:    "created",
		Component: "note",
		ID:        note.ID,
		Detail:    note.ID, // raw ID — simulates recordTitle fallback
		Href:      "/t/note/" + note.ID,
	})

	body := get(t, h, "/activity").Body.String()

	// Acceptance 1: the activity log must not show a raw database id.
	if strings.Contains(said(body), note.ID) {
		t.Errorf("the activity log should not show a raw database id %q for a non-entry target\n\nwant heading: Assistant created note Test Validation\ngot body:\n%s", note.ID, truncate(body))
	}

	// The resolved title must appear in the h3.
	if !anyH3Says(body, "Assistant created note Test Validation") {
		t.Errorf("the h3 should show the resolved title, not the raw id\n\nwant: Assistant created note Test Validation\ngot body:\n%s", truncate(body))
	}
}

// TestUnresolvedRecordComponentShowsLabelAndID seeds an activity entry where
// target is "record" and detail looks like "note <raw-id>" — the output of
// Summarise for a record component whose referenced note has no title prop. The
// current say() does not handle this case, so the log shows "Assistant added
// record note <id>". After the fix it should show "Assistant added note Test
// Validation" (the word "record" is replaced by the content type name and the
// id resolves to a title). This covers acceptance items 1 and 2.
func TestUnresolvedRecordComponentShowsLabelAndID(t *testing.T) {
	a, h := newApp(t)

	// Seed a note with a title so we can verify it appears after resolution.
	note, err := a.Store.Create("note", map[string]any{
		"title": "Test Validation",
		"body":  "a record block reference",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantDetail := "note " + note.ID // what Summarise("record",{type:"note",record: id}) produces

	// Seed an activity entry mimicking what add_component(record, ...) writes.
	chat.Record(a.Store, "assistant", chat.Change{
		Action:    "added",
		Component: "record",
		ID:        note.ID,
		Detail:    wantDetail, // "<type> <raw_id>" from Summarise
		Href:      "/t/note/" + note.ID,
	})

	body := get(t, h, "/activity").Body.String()
	s := said(body)

	// Acceptance 2: the word "record" must not appear as a generic label.
	if strings.Contains(s, "Assistant added record ") {
		t.Errorf("the activity log should not show 'record' as a generic label in headings\n\nwant heading: Assistant added note Test Validation\ngot body:\n%s", truncate(body))
	}

	// Acceptance 1: the raw id must not appear in visible text.
	if strings.Contains(s, note.ID) {
		t.Errorf("the activity log should not show a raw database id %q\n\nwant heading: Assistant added note Test Validation\ngot body:\n%s", note.ID, truncate(body))
	}

	// The heading must contain the content type name "note" replacing "record".
	if !anyH3Says(body, "Assistant added note Test Validation") {
		t.Errorf("the h3 should replace 'record' with the content type name and resolve the id to a title\n\nwant: Assistant added note Test Validation\ngot body:\n%s", truncate(body))
	}

	// The old buggy heading must not appear.
	if anyH3Says(body, "Assistant added record") {
		t.Errorf("the h3 must not say 'record' as a target\nwant: Assistant added note Test Validation\ngot body:\n%s", truncate(body))
	}
}

// TestResolvedRecordWithExistingTitleStillWorks seeds an activity entry where
// the detail is already a readable title (not a raw id). This ensures the fix
// does not break records that were created with titles — they should render as
// before: "Assistant created note Seeds to buy".
func TestResolvedRecordWithExistingTitleStillWorks(t *testing.T) {
	a, h := newApp(t)

	note, err := a.Store.Create("note", map[string]any{
		"title": "Seeds to buy",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an activity entry where detail is already the title (normal case).
	chat.Record(a.Store, "assistant", chat.Change{
		Action:    "created",
		Component: "note",
		ID:        note.ID,
		Detail:    "Seeds to buy", // already a readable title
		Href:      "/t/note/" + note.ID,
	})

	body := get(t, h, "/activity").Body.String()

	if !anyH3Says(body, "Assistant created note Seeds to buy") {
		t.Errorf("existing resolved titles should still appear correctly\n\nwant: Assistant created note Seeds to buy\ngot body:\n%s", truncate(body))
	}

	// The raw id must not leak.
	if strings.Contains(said(body), note.ID) {
		t.Errorf("the activity log should not show a raw database id even for normal entries\n%s", said(body))
	}
}
