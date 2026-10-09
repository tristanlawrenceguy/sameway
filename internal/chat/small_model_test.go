package chat_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The shapes Claude Haiku reached for in a real workspace: a record's
// fields beside type, inside a JSON string, under set. Each is taken as
// meant.
func TestLooseArgumentsAreTakenAsMeant(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "title": "Beside"}),
		call("create_record", map[string]any{"type": "note", "set": `{"title":"In a string under set"}`}),
		call("create_record", map[string]any{"type": "note", "fields": `{"title":"Fields as a string"}`}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "make three notes"); err != nil {
		t.Fatal(err)
	}
	notes, _ := svc.Store.List("note", store.ListOptions{})
	titles := map[string]bool{}
	for _, n := range notes {
		titles[n.Fields["title"].(string)] = true
	}
	for _, want := range []string{"Beside", "In a string under set", "Fields as a string"} {
		if !titles[want] {
			t.Errorf("no note %q; made %v", want, titles)
		}
	}
}

// A model is told of recording a meeting when it makes an event, not in
// every prompt.
func TestAnEventSaysHowToRecordIt(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Launch", "starts": "2026-10-05 10:00"}}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "a meeting on Monday, record it"); err != nil {
		t.Fatal(err)
	}
	if got := lastToolResult(m.seen[1]); got.IsError || !strings.Contains(got.Content, "call the record_meeting tool yourself now with event") {
		t.Errorf("making an event says how to record it: %+v", got)
	}
}

// A model counts days badly; Sameway says each day back with its weekday,
// and when the person named a weekday nothing falls on, which days do.
func TestADayIsSaidBackWithItsWeekday(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Clock = func() time.Time { return time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC) } // a Friday
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Pricing", "starts": "2026-10-07 14:00"}}),
		call("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Pricing again", "starts": "2026-10-06 14:00"}}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "A meeting next Tuesday at 2pm about pricing"); err != nil {
		t.Fatal(err)
	}
	wrong := lastToolResult(m.seen[1]).Content
	if !strings.Contains(wrong, "starts Wednesday 7 October 2026, 14:00") || !strings.Contains(wrong, "the coming Tuesday is 6 October") {
		t.Errorf("a day off is said, with the right one: %s", wrong)
	}
	right := lastToolResult(m.seen[2]).Content
	if !strings.Contains(right, "starts Tuesday 6 October 2026, 14:00") || strings.Contains(right, "the coming Tuesday") {
		t.Errorf("a right day is only said back: %s", right)
	}
}

// A model asks after words a record holds, not its exact title.
func TestFindRecordsMatchesEveryWordAnywhere(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Store.Create("note", map[string]any{"title": "Boiler", "body": "The engineer comes on Thursday."})
	svc.Store.Create("note", map[string]any{"title": "Garden", "body": "Plant garlic."})
	raw := []byte(`{"type":"note","query":"boiler engineer"}`)
	if text, _ := svc.Call("find_records", raw); !strings.Contains(text, "Boiler") || strings.Contains(text, "Garden") {
		t.Errorf("every word, in the title or the words: %s", text)
	}
}

// A record made with the title of one already there says so.
func TestARecordOfTheSameTitleIsSaid(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	have, _ := svc.Store.Create("note", map[string]any{"title": "Chapter 1"})
	raw := []byte(`{"type":"note","fields":{"title":"chapter 1"}}`)
	if text, _ := svc.Call("create_record", raw); !strings.Contains(text, "titled the same was there already, "+have.ID) {
		t.Errorf("the one there is named: %s", text)
	}
}

// A search with a word no record uses still finds what has the rest.
func TestSearchFallsBackToSomeOfTheWords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Store.Create("note", map[string]any{"title": "Boiler", "body": "The engineer comes on Thursday."})
	if text, _ := svc.Call("search", []byte(`{"query":"boiler engineer visit"}`)); !strings.Contains(text, "Nothing has every word") || !strings.Contains(text, "Boiler") {
		t.Errorf("some of the words, said so: %s", text)
	}
}

// An action that could never run is refused with how to write it: what
// is a kind of record, and a condition goes in only.
func TestAnActionThatCouldNeverRunIsRefused(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	raw := []byte(`{"type":"action","fields":{"title":"Tell","kind":"webhook","url":"https://example.com/h","when":"changed","what":"status=done"}}`)
	if text, isErr := svc.Call("create_record", raw); !isErr || !strings.Contains(text, `a condition goes in only, as ["status=done"]`) {
		t.Errorf("what is a kind, a condition goes in only: %s", text)
	}
	raw = []byte(`{"type":"action","fields":{"title":"Tell","kind":"webhook","url":"https://example.com/h","when":"changed","what":"task","only":["colour=red"]}}`)
	if _, isErr := svc.Call("create_record", raw); !isErr {
		t.Error("a condition the kind cannot meet is refused")
	}
	raw = []byte(`{"type":"action","fields":{"title":"Tell","kind":"webhook","url":"https://example.com/h","when":"changed","what":"task","only":["status=done"]}}`)
	if text, isErr := svc.Call("create_record", raw); isErr {
		t.Errorf("a right one is kept: %s", text)
	}
}

// A timed thing given a day alone becomes all day, as a person setting
// tomorrow means; the result says the time went, so a model moving it
// can put the time back.
func TestADayAloneSaysTheTimeWent(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	ev, _ := svc.Store.Create("event", map[string]any{"title": "Team lunch", "starts": "2026-10-08 12:30"})
	raw := []byte(`{"type":"event","id":"` + ev.ID + `","fields":{"starts":"2026-10-09"}}`)
	if text, _ := svc.Call("update_record", raw); !strings.Contains(text, "starts was at 12:30 and is now the whole day") {
		t.Errorf("the time going is said: %s", text)
	}
}

// Nothing found says the day searched with its weekday, so a day counted
// wrong shows.
func TestNothingFoundSaysTheDay(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Store.Create("task", map[string]any{"title": "Pay rent", "due": "2026-10-08"})
	if text, _ := svc.Call("find_records", []byte(`{"type":"task","where":["due=2026-10-10"]}`)); !strings.Contains(text, "Sat 10 Oct") {
		t.Errorf("the weekday is said: %s", text)
	}
}
