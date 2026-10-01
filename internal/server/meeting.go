package server

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A meeting is an event with its recording: its page plays the recording
// with its transcript, and until it is written up offers to have the
// assistant write it up (chat/meeting.go). A recording with a transcript
// that no meeting has offers the same. Both are parts of the page
// (parts.go), recording and write-up, off until the assistant sees a
// reason or the person keeps them on.

// meetingExtras is what an event's page shows of its recording.
func (s *Server) meetingExtras(r *http.Request, rec *store.Record) string {
	id, _ := rec.Fields["recording"].(string)
	if id == "" {
		return s.recordingToAdd(r, rec)
	}
	file, err := s.app.Store.Get(FileType, id)
	if err != nil || !isRecording(file) {
		return ""
	}
	_, here := s.shown(r)
	page := "/t/" + chat.EventType + "/" + rec.ID
	var b strings.Builder
	if summary, _ := rec.Fields["summary"].(string); strings.TrimSpace(summary) == "" && len(s.heard(file)) > 0 && changes(r) && s.showing(r, WriteUpPart) {
		b.WriteString(writeUpOffer("Write up this meeting (/t/"+chat.EventType+"/"+rec.ID+") from its recording (/t/"+FileType+"/"+id+")", "Ask the assistant to write it up", s))
		b.WriteString(s.fewer(page, WriteUpPart, "the offer to write it up", here))
	}
	if s.showing(r, RecordingPart) {
		b.WriteString(string(s.component("media", s.recordingOf(file))))
		b.WriteString(s.fewer(page, RecordingPart, "the recording", here))
	}
	return b.String()
}

// recordingOffer is a recording's page offering a write-up, when it has a
// transcript and no meeting has it yet.
func (s *Server) recordingOffer(r *http.Request, file *store.Record) string {
	if !changes(r) || !s.showing(r, WriteUpPart) || len(s.heard(file)) == 0 {
		return ""
	}
	t, ok := s.app.Types.Get(chat.EventType)
	if !ok {
		return ""
	}
	if _, has := t.Field("recording"); !has {
		return ""
	}
	if had, _ := query.Filter(s.app.Store, t, []string{"recording=" + file.ID}, "", 1, time.Now()); len(had) > 0 {
		return ""
	}
	return writeUpOffer("Write up the meeting in this recording (/t/"+FileType+"/"+file.ID+")", "Write up the meeting", s)
}

func writeUpOffer(ask, label string, s *Server) string {
	return `<p class="sw-muted">The assistant writes up a meeting from what was said: a summary, what was decided and the tasks that came up, each linked to where it was said, for you to check. One Undo takes it back.</p><p>` +
		string(s.component("link", map[string]any{"href": "/chat?prompt=" + url.QueryEscape(ask+": a short summary, what was decided and the tasks that came up."), "label": label, "look": "button"})) + `</p>`
}

// changes says whether whoever asked may change the workspace.
func changes(r *http.Request) bool {
	a := chat.VisitorOf(r.Context()).Access
	return a != chat.View && a != chat.Public
}

// recordingToAdd is a meeting with no recording yet: record it here, the
// microphone and this computer's sound for a call, or add what the
// meeting app made. It is the recording part, off at rest; but once a
// meeting someone asked to have recorded is over without one, the page
// says so by itself, since that is the moment it is wanted.
func (s *Server) recordingToAdd(r *http.Request, ev *store.Record) string {
	if !changes(r) {
		return ""
	}
	page := "/t/" + chat.EventType + "/" + ev.ID
	over := false
	if st, err := time.Parse(time.RFC3339, str(ev.Fields["starts"], "")); err == nil {
		over = time.Now().After(st.Add(chat.MeetingLength(ev)))
	}
	wanted := over && len(chat.RemindersAbout(s.app.Store, chat.RecordAbout(ev.ID))) > 0
	if !s.showing(r, RecordingPart) && !wanted {
		return ""
	}
	say := "Record it here: the microphone, and with the box ticked this computer's sound, the other people on a call (share the call's tab, or the entire screen for an app such as Teams or Zoom, with its sound). Headphones keep the call out of the microphone. Or add the recording or transcript the meeting app makes, afterwards."
	if over {
		say = "It has ended with no recording. Add the recording or transcript the meeting app made, a .vtt from Teams or Zoom or the audio, or one made on another device." + s.fetchSaid(ev)
	}
	_, here := s.shown(r)
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="meeting-recording-title"><h2 id="meeting-recording-title">Recording</h2><p>` + template.HTMLEscapeString(say) + `</p>`)
	b.WriteString(string(s.component("upload", map[string]any{"label": "Add the recording", "action": "/t/file/upload?meeting=" + ev.ID,
		"from": page + "?show=recording", "id": "meeting-recording", "sound": "computer",
		"hint": "A recording, or a transcript as a .vtt or .srt file. Up to 4 GB."})))
	b.WriteString(s.fewer(page, RecordingPart, "the recording", here) + `</section>`)
	return b.String()
}

// toMeeting gives a meeting the file just added as its recording, as the
// person's own change, when it has none; it says what it did.
func (s *Server) toMeeting(r *http.Request, eventID string, file *store.Record) string {
	ev, err := s.app.Store.Get(chat.EventType, eventID)
	if err != nil {
		return ""
	}
	if had, _ := ev.Fields["recording"].(string); had != "" {
		return " The meeting has a recording already, so this one is in your files."
	}
	if _, _, err := chat.WriteAs(s.app.Store, s.who(r), "updated", chat.EventType, eventID, map[string]any{"recording": file.ID}); err != nil {
		return " It could not be given to the meeting: " + err.Error()
	}
	return " It is the meeting's recording now."
}
