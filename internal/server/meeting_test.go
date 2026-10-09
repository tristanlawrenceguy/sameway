package server_test

import (
	"strings"
	"testing"
)

// A meeting's page plays its recording, each line of the transcript one
// a link can land on, and offers a write-up until there is one; a
// recording no meeting has offers it too.
func TestAMeetingsPageOffersItsWriteUp(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	file, err := a.Store.Create("file", map[string]any{"title": "Stand-up", "kind": "audio", "name": "standup.m4a",
		"text": "[0:00] **Ann:** Morning.\n\n[0:12] **Ben:** We ship on Friday."})
	if err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/t/file/"+file.ID).Body.String(); strings.Contains(page, "Write up the meeting") {
		t.Error("the offer is a part, off at rest")
	}
	if page := get(t, h, "/t/file/"+file.ID+"?show=write-up").Body.String(); !strings.Contains(page, "Write up the meeting") {
		t.Errorf("a recording no meeting has offers a write-up")
	}
	ev, err := a.Store.Create("event", map[string]any{"title": "Monday stand-up", "recording": file.ID})
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/event/"+ev.ID+"?show=recording&show=write-up").Body.String()
	if !strings.Contains(page, `id="media-`+file.ID+`-at-12"`) || !strings.Contains(page, "Ask the assistant to write it up") {
		t.Errorf("the meeting plays its recording, lines anchored, and offers a write-up: %.3000s", page)
	}
	if page := get(t, h, "/t/file/"+file.ID+"?show=write-up").Body.String(); strings.Contains(page, "Write up the meeting") {
		t.Error("a recording a meeting has is written up there, not offered again")
	}
	a.Store.Update("event", ev.ID, map[string]any{"summary": "Ship Friday."})
	if page := get(t, h, "/t/event/"+ev.ID+"?show=write-up").Body.String(); strings.Contains(page, "Ask the assistant to write it up") {
		t.Error("a meeting written up is not offered again")
	}
}
