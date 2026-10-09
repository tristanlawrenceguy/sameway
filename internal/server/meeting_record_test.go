package server_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A meeting someone records asks for it: recorded here, a reminder as it
// starts that opens its page ready to record; recorded by the meeting app,
// one as it ends to add what the app made. Before, the page is the
// meeting; once it is over with no recording, the page asks for one.
func TestAMeetingAsksForItsRecording(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	soon := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Minute)
	ev, err := a.Store.Create("event", map[string]any{"title": "Stand-up", "starts": soon.Format(time.RFC3339), "ends": soon.Add(30 * time.Minute).Format(time.RFC3339)})
	if err != nil {
		t.Fatal(err)
	}
	call := func(how string) string {
		raw, _ := json.Marshal(map[string]any{"event": ev.ID, "how": how})
		text, isErr := a.Chat.Call("record_meeting", raw)
		if isErr {
			t.Fatal(text)
		}
		return text
	}
	call("here")
	call("app")
	if again := call("app"); !strings.Contains(again, "already") {
		t.Errorf("asked twice, it is one reminder: %s", again)
	}
	reminders := chat.RemindersAbout(a.Store, chat.RecordAbout(ev.ID))
	if len(reminders) != 2 {
		t.Fatalf("one as it starts, one as it ends: %d", len(reminders))
	}
	at := map[string]string{}
	for _, r := range reminders {
		at[r.Fields["title"].(string)] = r.Fields["at"].(string)
	}
	if at["Record Stand-up"] != soon.Format(time.RFC3339) || at["Add the recording of Stand-up"] != soon.Add(30*time.Minute).Format(time.RFC3339) {
		t.Errorf("at its start and its end: %v", at)
	}

	page := "/t/event/" + ev.ID
	if strings.Contains(get(t, h, page).Body.String(), "meeting-recording") {
		t.Error("before it, the meeting's page is the meeting")
	}
	asked := get(t, h, page+"?show=recording").Body.String()
	if !strings.Contains(asked, `action="/t/file/upload?meeting=`+ev.ID+`"`) || !strings.Contains(asked, "sw-voice__also") {
		t.Errorf("opened, it records the microphone and the computer's sound, or takes a file: %.2000s", asked)
	}

	// It is over, with no recording: the page asks by itself.
	past := time.Now().Add(-2 * time.Hour).UTC()
	a.Store.Update("event", ev.ID, map[string]any{"starts": past.Format(time.RFC3339), "ends": past.Add(30 * time.Minute).Format(time.RFC3339)})
	if over := get(t, h, page).Body.String(); !strings.Contains(over, "It has ended with no recording") {
		t.Error("over and wanted, the page asks for the recording")
	}

	// What is added there is the meeting's.
	body, ct := multipartFile(t, "standup.vtt", "WEBVTT\n\n00:00:01.000 --> 00:00:03.000\n<v Ann>Morning.\n", nil)
	do(t, h, "POST", "/t/file/upload?meeting="+ev.ID, body, ct)
	got, _ := a.Store.Get("event", ev.ID)
	file, _ := got.Fields["recording"].(string)
	if file == "" {
		t.Fatal("the file added on the meeting's page is its recording")
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if log[0].Fields["target_id"] != ev.ID {
		t.Error("giving it to the meeting is a change to undo like any other")
	}
	if strings.Contains(get(t, h, page).Body.String(), "It has ended with no recording") {
		t.Error("with its recording, the page no longer asks")
	}
}
